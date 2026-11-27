package oidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/niudaii/goutil/seckit/httpx"
)

const (
	defaultJWKSCacheTTL = 15 * time.Minute
	maxJWKSBytes        = 1 << 20
)

type jwksCache struct {
	url    string
	client *http.Client
	ttl    time.Duration
	now    func() time.Time

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

func newJWKSCache(jwksURL string, client *http.Client, now func() time.Time) *jwksCache {
	if now == nil {
		now = time.Now
	}
	return &jwksCache{
		url:    jwksURL,
		client: client,
		ttl:    defaultJWKSCacheTTL,
		now:    now,
		keys:   make(map[string]*rsa.PublicKey),
	}
}

func (j *jwksCache) rsaKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	now := j.now()
	j.mu.Lock()
	cached, ok := j.keys[kid]
	fresh := now.Before(j.expiresAt)
	j.mu.Unlock()
	if ok && fresh {
		return cached, nil
	}

	if err := j.refresh(ctx); err != nil {
		if ok {
			return cached, nil
		}
		return nil, err
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	key, ok := j.keys[kid]
	if !ok {
		return nil, fmt.Errorf("oidc: JWKS missing key %q", kid)
	}
	return key, nil
}

func (j *jwksCache) refresh(ctx context.Context) error {
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Use string `json:"use"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	_, _, err := httpx.DoJSON(ctx, j.client, http.MethodGet, j.url, nil, &doc, nil, maxJWKSBytes)
	if err != nil {
		return fmt.Errorf("oidc: fetch JWKS: %w", err)
	}

	parsed := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, item := range doc.Keys {
		if item.Kty != "RSA" || item.Kid == "" || item.N == "" || item.E == "" {
			continue
		}
		if item.Use != "" && item.Use != "sig" {
			continue
		}
		pub, err := parseRSAPublicKey(item.N, item.E)
		if err != nil {
			continue
		}
		parsed[item.Kid] = pub
	}
	if len(parsed) == 0 {
		return errors.New("oidc: JWKS contained no usable RSA signing keys")
	}

	j.mu.Lock()
	j.keys = parsed
	j.expiresAt = j.now().Add(j.ttl)
	j.mu.Unlock()
	return nil
}

func parseRSAPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil || len(nBytes) == 0 {
		return nil, errors.New("invalid RSA modulus")
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil || len(eBytes) == 0 || len(eBytes) > 8 {
		return nil, errors.New("invalid RSA exponent")
	}
	var exp uint64
	for _, b := range eBytes {
		exp = (exp << 8) | uint64(b)
	}
	if exp < 3 {
		return nil, errors.New("invalid RSA exponent value")
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: int(exp),
	}, nil
}

func (c *Client) verifyKeyFunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		switch token.Method.Alg() {
		case jwt.SigningMethodHS256.Alg():
			return []byte(c.clientSecret), nil
		case jwt.SigningMethodRS256.Alg():
			if !c.allowRS256 || c.jwks == nil {
				return nil, fmt.Errorf("oidc: signing algorithm %s not allowed", token.Method.Alg())
			}
			kid, _ := token.Header["kid"].(string)
			if kid == "" {
				return nil, errors.New("oidc: RS256 ID token missing kid header")
			}
			return c.jwks.rsaKey(ctx, kid)
		default:
			return nil, fmt.Errorf("oidc: unexpected signing method %v", token.Header["alg"])
		}
	}
}
