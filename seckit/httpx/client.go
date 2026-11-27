 // Package httpx provides standardized net/http client construction, retry/backoff,
// redirect control, bounded response reading, and JSON request helpers.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultTimeout is applied when no explicit timeout option is provided.
const DefaultTimeout = 15 * time.Second

const maxRetryAfterSeconds = 60

// ErrBodyTooLarge is returned when an HTTP response body exceeds the configured byte limit.
var ErrBodyTooLarge = errors.New("httpx: response body exceeds limit")

// NoRedirect is a CheckRedirect policy that prevents automatic following of 3xx redirects.
func NoRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

type config struct {
	timeout       time.Duration
	checkRedirect func(*http.Request, []*http.Request) error
	proxyURL      string
	transport     http.RoundTripper
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	userAgent     string
	headers       map[string]string
}

// Option configures an *http.Client built by New or NewWithError.
type Option func(*config)

// WithTimeout overrides the client request timeout (defaults to DefaultTimeout).
func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithNoRedirect disables automatic following of HTTP redirects.
func WithNoRedirect() Option {
	return func(c *config) {
		c.checkRedirect = NoRedirect
	}
}

// WithMaxRedirects allows at most n redirects before failing.
func WithMaxRedirects(n int) Option {
	return func(c *config) {
		c.checkRedirect = func(_ *http.Request, via []*http.Request) error {
			if len(via) > n {
				return fmt.Errorf("httpx: stopped after %d redirects", n)
			}
			return nil
		}
	}
}

// WithProxyURL configures an HTTP/HTTPS proxy URL for the client transport.
func WithProxyURL(raw string) Option {
	return func(c *config) {
		c.proxyURL = strings.TrimSpace(raw)
	}
}

// WithTransport sets the underlying RoundTripper used by the client.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *config) {
		c.transport = rt
	}
}

// WithRetry enables retries on transport failures, HTTP 429, and HTTP 5xx responses
// when the request body is nil or repeatable via GetBody.
func WithRetry(maxRetries int, baseDelay, maxDelay time.Duration) Option {
	return func(c *config) {
		if maxRetries < 0 {
			maxRetries = 0
		}
		c.maxRetries = maxRetries
		c.baseDelay = baseDelay
		c.maxDelay = maxDelay
	}
}

// WithUserAgent sets a default User-Agent header when the request does not already specify one.
func WithUserAgent(ua string) Option {
	return func(c *config) {
		c.userAgent = ua
	}
}

// WithHeaders sets default headers applied when not already present on the request.
func WithHeaders(h map[string]string) Option {
	return func(c *config) {
		if len(h) == 0 {
			return
		}
		c.headers = make(map[string]string, len(h))
		for k, v := range h {
			c.headers[k] = v
		}
	}
}

// New builds a configured *http.Client, ignoring configuration errors such as an invalid proxy URL.
// Use NewWithError when untrusted or user-supplied proxy URLs must be validated.
func New(opts ...Option) *http.Client {
	client, _ := newClient(opts, true)
	return client
}

// NewWithError builds a configured *http.Client and validates options such as proxy URLs.
func NewWithError(opts ...Option) (*http.Client, error) {
	return newClient(opts, false)
}

func newClient(opts []Option, ignoreProxyErr bool) (*http.Client, error) {
	cfg := config{timeout: DefaultTimeout}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	rt := cfg.transport
	if cfg.proxyURL != "" {
		parsed, err := url.Parse(cfg.proxyURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			if !ignoreProxyErr {
				return nil, fmt.Errorf("httpx: invalid proxy URL %q", cfg.proxyURL)
			}
		} else {
			tr := cloneTransport(rt)
			tr.Proxy = http.ProxyURL(parsed)
			rt = tr
		}
	}

	// Keep Transport nil when no proxy/retry/header/transport options were set so
	// test harnesses that swap http.DefaultTransport continue to intercept requests.
	if cfg.maxRetries > 0 || cfg.userAgent != "" || len(cfg.headers) > 0 {
		rt = &roundTripper{
			base:       rt,
			maxRetries: cfg.maxRetries,
			baseDelay:  cfg.baseDelay,
			maxDelay:   cfg.maxDelay,
			userAgent:  cfg.userAgent,
			headers:    cfg.headers,
		}
	}

	return &http.Client{
		Timeout:       cfg.timeout,
		CheckRedirect: cfg.checkRedirect,
		Transport:     rt,
	}, nil
}

func cloneTransport(rt http.RoundTripper) *http.Transport {
	if t, ok := rt.(*http.Transport); ok && t != nil {
		return t.Clone()
	}
	if t, ok := http.DefaultTransport.(*http.Transport); ok && t != nil {
		return t.Clone()
	}
	return &http.Transport{}
}

type roundTripper struct {
	base       http.RoundTripper
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	userAgent  string
	headers    map[string]string
}

func (rt *roundTripper) nextTransport() http.RoundTripper {
	if rt.base != nil {
		return rt.base
	}
	if http.DefaultTransport != nil {
		return http.DefaultTransport
	}
	return http.DefaultTransport
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	baseReq := req
	if rt.userAgent != "" || len(rt.headers) > 0 {
		cloned := req.Clone(req.Context())
		if rt.userAgent != "" && cloned.Header.Get("User-Agent") == "" {
			cloned.Header.Set("User-Agent", rt.userAgent)
		}
		for k, v := range rt.headers {
			if cloned.Header.Get(k) == "" {
				cloned.Header.Set(k, v)
			}
		}
		baseReq = cloned
	}

	canRetryBody := baseReq.Body == nil || baseReq.GetBody != nil
	attempts := 1
	if rt.maxRetries > 0 && canRetryBody {
		attempts += rt.maxRetries
	}

	next := rt.nextTransport()
	for attempt := 0; attempt < attempts; attempt++ {
		attemptReq := baseReq
		if attempt > 0 {
			attemptReq = baseReq.Clone(baseReq.Context())
			if baseReq.GetBody != nil {
				body, err := baseReq.GetBody()
				if err != nil {
					return nil, err
				}
				attemptReq.Body = body
			}
		}

		resp, err := next.RoundTrip(attemptReq)
		lastAttempt := attempt == attempts-1
		if !shouldRetry(resp, err) || lastAttempt {
			return resp, err
		}

		delay := rt.backoffDelay(resp, attempt)
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			_ = resp.Body.Close()
		}

		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-baseReq.Context().Done():
				timer.Stop()
				return nil, baseReq.Context().Err()
			case <-timer.C:
			}
		} else if err := baseReq.Context().Err(); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("httpx: exhausted retries")
}

func shouldRetry(resp *http.Response, err error) bool {
	if err != nil {
		return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
	}
	return resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
}

func (rt *roundTripper) backoffDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		if raw := strings.TrimSpace(resp.Header.Get("Retry-After")); raw != "" {
			if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
				if secs > maxRetryAfterSeconds {
					secs = maxRetryAfterSeconds
				}
				return time.Duration(secs) * time.Second
			}
		}
	}
	if rt.baseDelay <= 0 {
		return 0
	}
	delay := rt.baseDelay << attempt
	if delay <= 0 || (rt.maxDelay > 0 && delay > rt.maxDelay) {
		if rt.maxDelay > 0 {
			return rt.maxDelay
		}
		return rt.baseDelay
	}
	return delay
}

// ReadBodyLimit reads at most maxBytes from r, returning ErrBodyTooLarge if exceeded.
func ReadBodyLimit(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes < 0 {
		return nil, errors.New("httpx: maxBytes must be non-negative")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrBodyTooLarge
	}
	return data, nil
}

// DoJSON executes an HTTP request with an optional JSON request body and optional JSON response decoding.
// Non-2xx status codes return the HTTP status code, bounded body bytes, and an error.
func DoJSON(ctx context.Context, client *http.Client, method, rawURL string, reqBody, respDst any, headers map[string]string, maxRespBytes int64) (int, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = http.DefaultClient
	}
	if maxRespBytes <= 0 {
		maxRespBytes = 8 << 20
	}

	var bodyReader io.Reader
	if reqBody != nil {
		encoded, err := json.Marshal(reqBody)
		if err != nil {
			return 0, nil, fmt.Errorf("httpx: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("httpx: build request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if reqBody != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	raw, err := ReadBodyLimit(resp.Body, maxRespBytes)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, raw, fmt.Errorf("httpx: HTTP %d", resp.StatusCode)
	}

	if respDst != nil && resp.StatusCode != http.StatusNoContent && len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, respDst); err != nil {
			return resp.StatusCode, raw, fmt.Errorf("httpx: decode JSON response: %w", err)
		}
	}
	return resp.StatusCode, raw, nil
}
