// Package oidc provides a stateless AES-256-GCM encrypted cookie OpenID Connect client
// supporting both net/http and Gin consumers.
package oidc

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/niudaii/goutil/seckit/httpx"
)

const (
	// APIKeyHeader is the standard header for machine-to-machine API key authentication.
	APIKeyHeader = "X-API-Key"
	// SubjectContextKey stores the authenticated subject in request contexts.
	SubjectContextKey = "auth.subject"

	defaultLoginPath      = "/api/auth/oidc/login"
	defaultCallbackPath   = "/api/auth/oidc/callback"
	defaultLogoutPath     = "/api/auth/logout"
	defaultStatusPath     = "/api/auth/status"
	defaultTransactionTTL = 10 * time.Minute
	defaultSessionTTL     = 12 * time.Hour
	defaultClockSkew      = time.Minute
	maxCookieValueLen     = 4096
	maxTokenBody          = 1 << 20
	maxRevocationEntries  = 8192
)

type subjectContextKeyType struct{}

var strictHostPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)

// Config configures an OIDC client instance.
type Config struct {
	Issuer               string
	ClientID             string
	ClientSecret         string
	RedirectURI          string
	AllowedCallbackPaths []string
	StrictHTTPSRoot      bool
	RejectWhitespace     bool
	MinSecretLen         int
	AllowRS256           bool

	AllowedSubjects  []string
	RequireWhitelist bool
	SubjectPattern   *regexp.Regexp

	CookiePrefix      string
	LoginCookieName   string
	SessionCookieName string
	CookieDomainSep   string

	LoginPath         string
	LogoutPath        string
	StatusPath        string
	HomePath          string
	SuccessRedirect   string
	FailureRedirect   string
	RedirectStatus    int
	ProviderErrorKind string
	AllowEmptyOrigin  bool

	TransactionTTL time.Duration
	SessionTTL     time.Duration
	ClockSkew      time.Duration

	Now            func() time.Time
	HTTPClient     *http.Client
	OnLoginSuccess func(subject string)
	OnLoginFailure func(reason string)
	OnLogout       func()
}

type loginFlow struct {
	ID        string `json:"id"`
	State     string `json:"s"`
	Nonce     string `json:"n"`
	ExpiresAt int64  `json:"e"`
}

type sessionPayload struct {
	ID        string `json:"id"`
	FlowID    string `json:"f,omitempty"`
	Subject   string `json:"u"`
	ExpiresAt int64  `json:"e"`
}

// Client implements stateless OpenID Connect authorization code flow with
// AES-256-GCM encrypted cookies and bounded local revocation defense.
type Client struct {
	issuer            string
	authURL           string
	tokenURL          string
	clientID          string
	clientSecret      string
	redirectURI       string
	origin            string
	rootCallback      bool
	secure            bool
	allowRS256        bool
	allowedSubjects   map[string]bool
	subjectPattern    *regexp.Regexp
	loginCookie       string
	sessionCookie     string
	loginPath         string
	callbackPath      string
	logoutPath        string
	statusPath        string
	homePath          string
	successRedirect   string
	failureRedirect   string
	redirectStatus    int
	providerErrorKind string
	allowEmptyOrigin  bool
	transactionTTL    time.Duration
	sessionTTL        time.Duration
	clockSkew         time.Duration
	now               func() time.Time
	httpClient        *http.Client
	cookieCipher      cipher.AEAD
	jwks              *jwksCache
	onLoginSuccess    func(string)
	onLoginFailure    func(string)
	onLogout          func()

	mu              sync.Mutex
	consumedFlows   map[string]time.Time
	inFlightFlows   map[string]struct{}
	revokedFlows    map[string]time.Time
	revokedSessions map[string]time.Time
}

// New validates cfg and creates a stateless OIDC client.
func New(cfg Config) (*Client, error) {
	issuer := strings.TrimRight(strings.TrimSpace(cfg.Issuer), "/")
	if issuer == "" {
		return nil, errors.New("oidc: issuer URL is required")
	}

	rawID := cfg.ClientID
	rawSecret := cfg.ClientSecret
	if cfg.RejectWhitespace && (strings.ContainsAny(rawID, " \t\r\n") || strings.ContainsAny(rawSecret, " \t\r\n")) {
		return nil, errors.New("oidc: client credentials must not contain whitespace")
	}
	clientID := strings.TrimSpace(rawID)
	secret := strings.TrimSpace(rawSecret)
	if clientID == "" || secret == "" {
		return nil, errors.New("oidc: client ID and client secret are required")
	}
	if strings.ContainsAny(clientID, ":\r\n") || strings.ContainsAny(secret, "\r\n") {
		return nil, errors.New("oidc: invalid client credentials format")
	}
	if cfg.MinSecretLen > 0 && len(secret) < cfg.MinSecretLen {
		return nil, fmt.Errorf("oidc: client secret must be at least %d bytes", cfg.MinSecretLen)
	}

	redirect := strings.TrimSpace(cfg.RedirectURI)
	origin, secure, rootCallback, cbPath, err := validateRedirectURI(redirect, cfg.AllowedCallbackPaths, cfg.StrictHTTPSRoot)
	if err != nil {
		return nil, err
	}

	allowedSubjects, err := parseAllowedSubjects(cfg.AllowedSubjects, cfg.RequireWhitelist, cfg.SubjectPattern)
	if err != nil {
		return nil, err
	}

	prefix := strings.TrimSpace(cfg.CookiePrefix)
	if prefix == "" {
		prefix = "sec_kit"
	}
	loginCookie := cfg.LoginCookieName
	if loginCookie == "" {
		loginCookie = prefix + "_oidc"
	}
	sessionCookie := cfg.SessionCookieName
	if sessionCookie == "" {
		sessionCookie = prefix + "_session"
	}
	domainSep := cfg.CookieDomainSep
	if domainSep == "" {
		domainSep = prefix + " OpenID cookie encryption v1"
	}

	cookieKey := hmac.New(sha256.New, []byte(secret))
	cookieKey.Write([]byte(domainSep))
	block, err := aes.NewCipher(cookieKey.Sum(nil))
	if err != nil {
		return nil, fmt.Errorf("oidc: init cookie cipher: %w", err)
	}
	cookieCipher, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("oidc: init cookie AEAD: %w", err)
	}

	loginPath := defaultIfEmpty(cfg.LoginPath, defaultLoginPath)
	logoutPath := defaultIfEmpty(cfg.LogoutPath, defaultLogoutPath)
	statusPath := defaultIfEmpty(cfg.StatusPath, defaultStatusPath)
	homePath := defaultIfEmpty(cfg.HomePath, "/")
	successRedirect := defaultIfEmpty(cfg.SuccessRedirect, homePath)
	failureRedirect := cfg.FailureRedirect
	if failureRedirect == "" {
		failureRedirect = homePath + "?login_error="
	}
	redirectStatus := cfg.RedirectStatus
	if redirectStatus == 0 {
		redirectStatus = http.StatusFound
	}
	providerErrorKind := defaultIfEmpty(cfg.ProviderErrorKind, "denied")

	txTTL := cfg.TransactionTTL
	if txTTL <= 0 {
		txTTL = defaultTransactionTTL
	}
	sessTTL := cfg.SessionTTL
	if sessTTL <= 0 {
		sessTTL = defaultSessionTTL
	}
	clockSkew := cfg.ClockSkew
	if clockSkew <= 0 {
		clockSkew = defaultClockSkew
	}
	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		// Leave Transport nil so tests swapping http.DefaultTransport work seamlessly.
		httpClient = httpx.New(httpx.WithTimeout(10*time.Second), httpx.WithNoRedirect())
	}

	var jwks *jwksCache
	if cfg.AllowRS256 {
		jwks = newJWKSCache(issuer+"/jwks", httpClient, nowFn)
	}

	return &Client{
		issuer:            issuer,
		authURL:           issuer + "/authorize",
		tokenURL:          issuer + "/token",
		clientID:          clientID,
		clientSecret:      secret,
		redirectURI:       redirect,
		origin:            origin,
		rootCallback:      rootCallback,
		secure:            secure,
		allowRS256:        cfg.AllowRS256,
		allowedSubjects:   allowedSubjects,
		subjectPattern:    cfg.SubjectPattern,
		loginCookie:       loginCookie,
		sessionCookie:     sessionCookie,
		loginPath:         loginPath,
		callbackPath:      cbPath,
		logoutPath:        logoutPath,
		statusPath:        statusPath,
		homePath:          homePath,
		successRedirect:   successRedirect,
		failureRedirect:   failureRedirect,
		redirectStatus:    redirectStatus,
		providerErrorKind: providerErrorKind,
		allowEmptyOrigin:  cfg.AllowEmptyOrigin,
		transactionTTL:    txTTL,
		sessionTTL:        sessTTL,
		clockSkew:         clockSkew,
		now:               nowFn,
		httpClient:        httpClient,
		cookieCipher:      cookieCipher,
		jwks:              jwks,
		onLoginSuccess:    cfg.OnLoginSuccess,
		onLoginFailure:    cfg.OnLoginFailure,
		onLogout:          cfg.OnLogout,
		consumedFlows:     make(map[string]time.Time),
		inFlightFlows:     make(map[string]struct{}),
		revokedFlows:      make(map[string]time.Time),
		revokedSessions:   make(map[string]time.Time),
	}, nil
}

func validateRedirectURI(redirect string, allowedPaths []string, strictHTTPSRoot bool) (origin string, secure, rootCallback bool, cbPath string, err error) {
	callback, err := url.Parse(redirect)
	if err != nil || callback.Hostname() == "" || callback.User != nil || callback.ForceQuery ||
		callback.RawQuery != "" || strings.Contains(redirect, "#") || callback.RawPath != "" || callback.Opaque != "" ||
		(callback.Scheme != "https" && callback.Scheme != "http") {
		return "", false, false, "", errors.New("oidc: redirect URI must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}

	if strictHTTPSRoot {
		if callback.Scheme != "https" || callback.Path != "/" {
			return "", false, false, "", errors.New("oidc: redirect URI must be an HTTPS root URL")
		}
		if callback.Host != callback.Hostname() || callback.Host != strings.ToLower(callback.Host) || !strictHostPattern.MatchString(callback.Host) {
			return "", false, false, "", errors.New("oidc: redirect URI must use a lowercase ASCII hostname without an explicit port")
		}
		return "https://" + callback.Host, true, true, "/", nil
	}

	paths := allowedPaths
	if len(paths) == 0 {
		paths = []string{defaultCallbackPath}
	}
	pathAllowed := false
	for _, p := range paths {
		if callback.Path == p {
			pathAllowed = true
			break
		}
	}
	if !pathAllowed {
		return "", false, false, "", fmt.Errorf("oidc: redirect URI path %q is not allowed", callback.Path)
	}
	rootCallback = callback.Path == "" || callback.Path == "/"
	cbPath = callback.Path
	if cbPath == "" {
		cbPath = "/"
	}

	secure = callback.Scheme == "https"
	hostname := callback.Hostname()
	if strings.IndexFunc(hostname, func(ch rune) bool { return ch > 0x7f }) >= 0 {
		return "", false, false, "", errors.New("oidc: redirect URI must use an ASCII hostname")
	}
	hostname = strings.ToLower(hostname)
	ip := net.ParseIP(hostname)
	if !secure && hostname != "localhost" && (ip == nil || (!ip.IsLoopback() && !ip.IsPrivate())) {
		return "", false, false, "", errors.New("oidc: HTTPS callback is required except for localhost, loopback, or private IPs")
	}

	originHost := hostname
	if strings.HasPrefix(callback.Host, "[") {
		if ip == nil || !strings.Contains(hostname, ":") {
			return "", false, false, "", errors.New("oidc: invalid IPv6 callback host")
		}
		if ipv4 := ip.To4(); ipv4 != nil {
			originHost = fmt.Sprintf("[::ffff:%x:%x]", uint16(ipv4[0])<<8|uint16(ipv4[1]), uint16(ipv4[2])<<8|uint16(ipv4[3]))
		} else {
			originHost = "[" + ip.String() + "]"
		}
	} else if strings.Contains(hostname, ":") {
		return "", false, false, "", errors.New("oidc: invalid callback host")
	}

	if port := callback.Port(); port != "" {
		number, err := strconv.ParseUint(port, 10, 16)
		if err != nil || number == 0 {
			return "", false, false, "", errors.New("oidc: invalid callback port")
		}
		if (secure && number != 443) || (!secure && number != 80) {
			originHost += ":" + strconv.FormatUint(number, 10)
		}
	} else if strings.HasSuffix(callback.Host, ":") {
		return "", false, false, "", errors.New("oidc: invalid callback port")
	}

	return callback.Scheme + "://" + originHost, secure, rootCallback, cbPath, nil
}

func parseAllowedSubjects(subjects []string, require bool, pattern *regexp.Regexp) (map[string]bool, error) {
	if len(subjects) == 0 {
		if require {
			return nil, errors.New("oidc: allowed subjects whitelist is required")
		}
		return nil, nil
	}
	allowed := make(map[string]bool, len(subjects))
	for _, raw := range subjects {
		sub := strings.TrimSpace(raw)
		if sub == "*" {
			return nil, errors.New("oidc: allowed subjects must not contain wildcard '*'")
		}
		if sub == "" {
			continue
		}
		if pattern != nil && !pattern.MatchString(sub) {
			return nil, fmt.Errorf("oidc: allowed subject %q does not match required format", sub)
		}
		allowed[sub] = true
	}
	if len(allowed) == 0 {
		return nil, errors.New("oidc: allowed subjects must contain at least one entry when configured")
	}
	return allowed, nil
}

// ClientID returns the configured OpenID Connect client ID.
func (c *Client) ClientID() string { return c.clientID }

// RedirectURI returns the configured callback URL.
func (c *Client) RedirectURI() string { return c.redirectURI }

// Origin returns the canonical scheme://host[:port] derived from RedirectURI.
func (c *Client) Origin() string { return c.origin }

// Secure reports whether the callback uses HTTPS (and cookies carry __Host- and Secure).
func (c *Client) Secure() bool { return c.secure }

// RootCallback reports whether RedirectURI points to "/" (or "").
func (c *Client) RootCallback() bool { return c.rootCallback }

// LoginPath returns the configured login endpoint path.
func (c *Client) LoginPath() string { return c.loginPath }

// CallbackPath returns the configured callback endpoint path.
func (c *Client) CallbackPath() string { return c.callbackPath }

// LogoutPath returns the configured logout endpoint path.
func (c *Client) LogoutPath() string { return c.logoutPath }

// StatusPath returns the configured auth status endpoint path.
func (c *Client) StatusPath() string { return c.statusPath }

// AllowedSubject reports whether subject is permitted (always true if no whitelist was configured).
func (c *Client) AllowedSubject(subject string) bool {
	if c.allowedSubjects == nil {
		return true
	}
	return c.allowedSubjects[subject]
}

// CookieName returns the wire cookie name (prefixed with __Host- when Secure is true).
func (c *Client) CookieName(base string) string {
	if c.secure && !strings.HasPrefix(base, "__Host-") {
		return "__Host-" + base
	}
	return base
}

// LoginCookieName returns the wire name of the login transaction cookie.
func (c *Client) LoginCookieName() string { return c.CookieName(c.loginCookie) }

// SessionCookieName returns the wire name of the authenticated session cookie.
func (c *Client) SessionCookieName() string { return c.CookieName(c.sessionCookie) }

// SetNowForTest overrides the clock function (intended for tests).
func (c *Client) SetNowForTest(fn func() time.Time) {
	if fn != nil {
		c.now = fn
	}
}

// SetTokenURLForTest overrides the token endpoint URL (intended for tests).
func (c *Client) SetTokenURLForTest(rawURL string) {
	c.tokenURL = rawURL
}

// SealSessionForTest seals a signed session cookie value for unit tests.
func (c *Client) SealSessionForTest(subject string, expires time.Time) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}
	return c.sealPayload(c.SessionCookieName(), sessionPayload{
		ID:        id,
		Subject:   subject,
		ExpiresAt: expires.UnixNano(),
	})
}

// Register mounts login, callback (if non-root), logout, and status handlers on mux.
func (c *Client) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+c.loginPath, c.HandleLogin)
	if !c.rootCallback {
		mux.HandleFunc("GET "+c.callbackPath, c.HandleCallback)
	}
	mux.HandleFunc("POST "+c.logoutPath, func(w http.ResponseWriter, r *http.Request) {
		if !c.SameOrigin(w, r) {
			return
		}
		c.HandleLogout(w, r)
	})
	mux.HandleFunc("GET "+c.statusPath, c.HandleStatus)
}

// HandleHome runs root callbacks if present, redirects unauthenticated users to login
// unless login_error is active, or delegates to serveIndex.
func (c *Client) HandleHome(w http.ResponseWriter, r *http.Request, serveIndex http.HandlerFunc) {
	SetNoStoreHeaders(w)
	if c.rootCallback && HasCallbackParams(r) {
		c.HandleCallback(w, r)
		return
	}
	if c.Subject(r) == "" {
		switch r.URL.Query().Get("login_error") {
		case "denied", "failed", "expired":
		default:
			c.HandleLogin(w, r)
			return
		}
	}
	serveIndex(w, r)
}

// HandleStatus responds with JSON authentication status.
func (c *Client) HandleStatus(w http.ResponseWriter, r *http.Request) {
	SetNoStoreHeaders(w)
	subject := c.Subject(r)
	writeAuthJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"enabled":       true,
			"authenticated": subject != "",
			"subject":       subject,
		},
	})
}

// HandleLogin starts an OIDC authorization code transaction with an encrypted binding cookie.
func (c *Client) HandleLogin(w http.ResponseWriter, r *http.Request) {
	SetNoStoreHeaders(w)
	flowID, err := randomID()
	if err != nil {
		c.redirectFailure(w, r, "failed")
		return
	}
	state, err := randomID()
	if err != nil {
		c.redirectFailure(w, r, "failed")
		return
	}
	nonce, err := randomID()
	if err != nil {
		c.redirectFailure(w, r, "failed")
		return
	}

	// If browser already had a pending flow cookie, revoke it locally to match single-pending-flow semantics.
	if prev, ok := c.readFlowCookie(r); ok && prev.ID != "" {
		c.revokeFlow(prev.ID, time.Unix(0, prev.ExpiresAt))
	}

	now := c.now()
	flow := loginFlow{
		ID:        flowID,
		State:     state,
		Nonce:     nonce,
		ExpiresAt: now.Add(c.transactionTTL).UnixNano(),
	}
	if err := c.setEncryptedCookie(w, c.LoginCookieName(), flow, int(c.transactionTTL.Seconds())); err != nil {
		c.redirectFailure(w, r, "failed")
		return
	}

	query := url.Values{
		"response_type": {"code"},
		"scope":         {"openid"},
		"client_id":     {c.clientID},
		"redirect_uri":  {c.redirectURI},
		"state":         {state},
		"nonce":         {nonce},
	}
	http.Redirect(w, r, c.authURL+"?"+query.Encode(), http.StatusFound)
}

// HandleCallback validates the encrypted login cookie, exchanges the authorization code,
// verifies the ID token, and issues a stateless AES-256-GCM session cookie.
func (c *Client) HandleCallback(w http.ResponseWriter, r *http.Request) {
	SetNoStoreHeaders(w)
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		c.redirectFailure(w, r, "expired")
		return
	}

	flow, ok := c.readFlowCookie(r)
	state := query.Get("state")
	if !ok || len(query["state"]) != 1 || subtle.ConstantTimeCompare([]byte(state), []byte(flow.State)) != 1 {
		c.redirectFailure(w, r, "expired")
		return
	}

	expires := time.Unix(0, flow.ExpiresAt)
	now := c.now()
	if !now.Before(expires) {
		c.clearCookie(w, c.LoginCookieName())
		c.redirectFailure(w, r, "expired")
		return
	}

	if !c.claimFlow(flow.ID, expires, now) {
		c.redirectFailure(w, r, "expired")
		return
	}
	defer c.releaseInFlightFlow(flow.ID)
	c.clearCookie(w, c.LoginCookieName())

	if query.Has("error") {
		c.redirectFailure(w, r, c.providerErrorKind)
		return
	}
	code := query.Get("code")
	if len(query["code"]) != 1 || code == "" || len(code) > 4096 {
		c.redirectFailure(w, r, "failed")
		return
	}

	subject, err := c.exchange(r.Context(), code, flow.Nonce)
	if err != nil {
		if c.onLoginFailure != nil {
			c.onLoginFailure(err.Error())
		}
		c.redirectFailure(w, r, "failed")
		return
	}
	if !c.AllowedSubject(subject) {
		if c.onLoginFailure != nil {
			c.onLoginFailure("subject not allowed: " + subject)
		}
		c.redirectFailure(w, r, "denied")
		return
	}

	if c.isFlowRevoked(flow.ID, c.now()) {
		c.redirectFailure(w, r, "expired")
		return
	}

	sessID, err := randomID()
	if err != nil {
		c.redirectFailure(w, r, "failed")
		return
	}
	// Revoke any previous session cookie carried on this browser.
	if prevSess, ok := c.readSessionCookie(r); ok && prevSess.ID != "" {
		c.revokeSession(prevSess.ID, time.Unix(0, prevSess.ExpiresAt))
	}

	sessNow := c.now()
	sess := sessionPayload{
		ID:        sessID,
		FlowID:    flow.ID,
		Subject:   subject,
		ExpiresAt: sessNow.Add(c.sessionTTL).UnixNano(),
	}
	if err := c.setEncryptedCookie(w, c.SessionCookieName(), sess, int(c.sessionTTL.Seconds())); err != nil {
		c.redirectFailure(w, r, "failed")
		return
	}
	if c.onLoginSuccess != nil {
		c.onLoginSuccess(subject)
	}
	http.Redirect(w, r, c.successRedirect, c.redirectStatus)
}

// RevokeRequest revokes the request's session and/or pending login flow locally and clears cookies.
func (c *Client) RevokeRequest(w http.ResponseWriter, r *http.Request) {
	if sess, ok := c.readSessionCookie(r); ok && sess.ID != "" {
		c.revokeSession(sess.ID, time.Unix(0, sess.ExpiresAt))
	}
	if flow, ok := c.readFlowCookie(r); ok && flow.ID != "" {
		c.revokeFlow(flow.ID, time.Unix(0, flow.ExpiresAt))
	}
	c.ClearCookies(w)
	if c.onLogout != nil {
		c.onLogout()
	}
}

// ClearCookies expires both session and login cookies in the browser.
func (c *Client) ClearCookies(w http.ResponseWriter) {
	c.clearCookie(w, c.SessionCookieName())
	c.clearCookie(w, c.LoginCookieName())
}

// HandleLogout revokes the current session, clears cookies, and responds with HTTP 200 JSON.
func (c *Client) HandleLogout(w http.ResponseWriter, r *http.Request) {
	SetNoStoreHeaders(w)
	c.RevokeRequest(w, r)
	writeAuthJSON(w, http.StatusOK, map[string]any{"code": http.StatusOK})
}

// Subject validates the encrypted session cookie on r and returns the authenticated subject or "".
func (c *Client) Subject(r *http.Request) string {
	return c.checkSession(nil, r)
}

// SubjectAndClearInvalid validates the session cookie and expires it on w if invalid/expired/revoked.
func (c *Client) SubjectAndClearInvalid(w http.ResponseWriter, r *http.Request) string {
	return c.checkSession(w, r)
}

func (c *Client) checkSession(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(c.SessionCookieName())
	if err != nil || cookie.Value == "" {
		return ""
	}
	var sess sessionPayload
	if !c.openPayload(c.SessionCookieName(), cookie.Value, &sess) {
		if w != nil {
			c.clearCookie(w, c.SessionCookieName())
		}
		return ""
	}
	now := c.now()
	expires := time.Unix(0, sess.ExpiresAt)
	if !now.Before(expires) || sess.Subject == "" || !c.AllowedSubject(sess.Subject) ||
		c.isSessionOrFlowRevoked(sess.ID, sess.FlowID, now) {
		if w != nil {
			c.clearCookie(w, c.SessionCookieName())
		}
		return ""
	}
	return sess.Subject
}

// OriginAllowed validates that non-GET/HEAD requests satisfy origin and Sec-Fetch-Site checks.
func (c *Client) OriginAllowed(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		return false
	}
	origin := r.Header.Get("Origin")
	if c.allowEmptyOrigin && origin == "" {
		return true
	}
	return origin == c.origin
}

// SameOrigin enforces OriginAllowed and writes 403 JSON on rejection.
func (c *Client) SameOrigin(w http.ResponseWriter, r *http.Request) bool {
	SetNoStoreHeaders(w)
	if !c.OriginAllowed(r) {
		writeAuthJSON(w, http.StatusForbidden, map[string]any{"code": http.StatusForbidden, "message": "不允许跨站请求"})
		return false
	}
	return true
}

// Require wraps next to require a valid OIDC session and same-origin for mutating requests.
func (c *Client) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetNoStoreHeaders(w)
		subject := c.Subject(r)
		if subject == "" {
			writeAuthJSON(w, http.StatusUnauthorized, map[string]any{"code": http.StatusUnauthorized, "message": "请先登录"})
			return
		}
		if !c.SameOrigin(w, r) {
			return
		}
		next.ServeHTTP(w, r.WithContext(WithSubject(r.Context(), subject)))
	})
}

// ValidateAPIKey verifies an X-API-Key header in constant time (using SHA-256 digests)
// and rejects duplicate header instances.
func ValidateAPIKey(r *http.Request, configuredKey string) (provided bool, valid bool) {
	values := r.Header.Values(APIKeyHeader)
	if len(values) == 0 {
		return false, false
	}
	if len(values) == 1 && configuredKey != "" {
		want := sha256.Sum256([]byte(configuredKey))
		got := sha256.Sum256([]byte(values[0]))
		if subtle.ConstantTimeCompare(got[:], want[:]) == 1 {
			return true, true
		}
	}
	return true, false
}

// WithSubject returns a copy of ctx carrying the authenticated subject.
func WithSubject(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, subjectContextKeyType{}, subject)
}

// SubjectFromContext extracts the authenticated subject written by Require.
func SubjectFromContext(ctx context.Context) string {
	v, _ := ctx.Value(subjectContextKeyType{}).(string)
	return v
}

// HasCallbackParams reports whether the request query contains OIDC callback parameters.
func HasCallbackParams(r *http.Request) bool {
	q := r.URL.Query()
	return q.Has("code") || q.Has("state") || q.Has("error")
}

// SetNoStoreHeaders sets Cache-Control: no-store and Referrer-Policy: no-referrer.
func SetNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

type idClaims struct {
	jwt.RegisteredClaims
	Nonce string `json:"nonce"`
	Party string `json:"azp"`
}

func (c *Client) exchange(ctx context.Context, code, nonce string) (string, error) {
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {c.redirectURI},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", errors.New("invalid token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.New("token endpoint unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token endpoint rejected request (HTTP %d)", resp.StatusCode)
	}

	body, err := httpx.ReadBodyLimit(resp.Body, maxTokenBody)
	var result struct {
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
	}
	if err != nil || json.Unmarshal(body, &result) != nil || result.Error != "" || result.IDToken == "" {
		return "", errors.New("invalid token response")
	}

	methods := []string{jwt.SigningMethodHS256.Alg()}
	if c.allowRS256 {
		methods = append(methods, jwt.SigningMethodRS256.Alg())
	}

	claims := &idClaims{}
	now := c.now()
	token, err := jwt.ParseWithClaims(
		result.IDToken,
		claims,
		c.verifyKeyFunc(ctx),
		jwt.WithValidMethods(methods),
		jwt.WithIssuer(c.issuer),
		jwt.WithAudience(c.clientID),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(c.clockSkew),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil || !token.Valid || claims.Subject == "" || len(claims.Subject) > 255 ||
		strings.IndexFunc(claims.Subject, func(ch rune) bool { return ch > 0x7f }) >= 0 ||
		(c.subjectPattern != nil && !c.subjectPattern.MatchString(claims.Subject)) ||
		len(claims.Audience) != 1 || claims.IssuedAt == nil ||
		now.Sub(claims.IssuedAt.Time) > c.transactionTTL ||
		(claims.Party != "" && claims.Party != c.clientID) ||
		subtle.ConstantTimeCompare([]byte(nonce), []byte(claims.Nonce)) != 1 {
		return "", errors.New("invalid ID token")
	}
	return claims.Subject, nil
}

func (c *Client) setEncryptedCookie(w http.ResponseWriter, cookieName string, payload any, maxAge int) error {
	encoded, err := c.sealPayload(cookieName, payload)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
	return nil
}

func (c *Client) clearCookie(w http.ResponseWriter, cookieName string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (c *Client) sealPayload(cookieName string, payload any) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sealed := c.cookieCipher.Seal(nil, nil, data, []byte(cookieName))
	encoded := base64.RawURLEncoding.EncodeToString(sealed)
	if len(encoded) > maxCookieValueLen {
		return "", errors.New("oidc: sealed cookie exceeds maximum length")
	}
	return encoded, nil
}

func (c *Client) openPayload(cookieName, value string, dst any) bool {
	if value == "" || len(value) > maxCookieValueLen {
		return false
	}
	sealed, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return false
	}
	plain, err := c.cookieCipher.Open(nil, nil, sealed, []byte(cookieName))
	if err != nil {
		return false
	}
	return json.Unmarshal(plain, dst) == nil
}

func (c *Client) readFlowCookie(r *http.Request) (loginFlow, bool) {
	cookie, err := r.Cookie(c.LoginCookieName())
	if err != nil {
		return loginFlow{}, false
	}
	var flow loginFlow
	if !c.openPayload(c.LoginCookieName(), cookie.Value, &flow) {
		return loginFlow{}, false
	}
	return flow, flow.ID != ""
}

func (c *Client) readSessionCookie(r *http.Request) (sessionPayload, bool) {
	cookie, err := r.Cookie(c.SessionCookieName())
	if err != nil {
		return sessionPayload{}, false
	}
	var sess sessionPayload
	if !c.openPayload(c.SessionCookieName(), cookie.Value, &sess) {
		return sessionPayload{}, false
	}
	return sess, sess.ID != ""
}

func (c *Client) claimFlow(flowID string, expires, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	pruneExpiryMap(c.consumedFlows, now, maxRevocationEntries)
	pruneExpiryMap(c.revokedFlows, now, maxRevocationEntries)
	if _, consumed := c.consumedFlows[flowID]; consumed {
		return false
	}
	if _, revoked := c.revokedFlows[flowID]; revoked {
		return false
	}
	if _, active := c.inFlightFlows[flowID]; active {
		return false
	}
	c.consumedFlows[flowID] = expires
	c.inFlightFlows[flowID] = struct{}{}
	return true
}

func (c *Client) releaseInFlightFlow(flowID string) {
	c.mu.Lock()
	delete(c.inFlightFlows, flowID)
	c.mu.Unlock()
}

func (c *Client) revokeFlow(flowID string, expires time.Time) {
	if flowID == "" {
		return
	}
	now := c.now()
	if !now.Before(expires) {
		expires = now.Add(c.sessionTTL)
	}
	c.mu.Lock()
	pruneExpiryMap(c.revokedFlows, now, maxRevocationEntries)
	c.revokedFlows[flowID] = expires
	c.mu.Unlock()
}

func (c *Client) revokeSession(sessionID string, expires time.Time) {
	if sessionID == "" {
		return
	}
	now := c.now()
	if !now.Before(expires) {
		expires = now.Add(c.sessionTTL)
	}
	c.mu.Lock()
	pruneExpiryMap(c.revokedSessions, now, maxRevocationEntries)
	c.revokedSessions[sessionID] = expires
	c.mu.Unlock()
}

func (c *Client) isFlowRevoked(flowID string, now time.Time) bool {
	if flowID == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	exp, ok := c.revokedFlows[flowID]
	if ok && !now.Before(exp) {
		delete(c.revokedFlows, flowID)
		return false
	}
	return ok
}

func (c *Client) isSessionOrFlowRevoked(sessionID, flowID string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if exp, ok := c.revokedSessions[sessionID]; ok {
		if now.Before(exp) {
			return true
		}
		delete(c.revokedSessions, sessionID)
	}
	if flowID != "" {
		if exp, ok := c.revokedFlows[flowID]; ok {
			if now.Before(exp) {
				return true
			}
			delete(c.revokedFlows, flowID)
		}
	}
	return false
}

func (c *Client) redirectFailure(w http.ResponseWriter, r *http.Request, kind string) {
	http.Redirect(w, r, c.failureRedirect+kind, c.redirectStatus)
}

func pruneExpiryMap(m map[string]time.Time, now time.Time, limit int) {
	for k, exp := range m {
		if !now.Before(exp) {
			delete(m, k)
		}
	}
	for len(m) >= limit {
		var oldestKey string
		var oldestExp time.Time
		for k, exp := range m {
			if oldestKey == "" || exp.Before(oldestExp) {
				oldestKey, oldestExp = k, exp
			}
		}
		if oldestKey == "" {
			break
		}
		delete(m, oldestKey)
	}
}

func randomID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func defaultIfEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func writeAuthJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
