package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestDefaultTransportHookPreserved(t *testing.T) {
	client := New(WithTimeout(5*time.Second), WithNoRedirect())
	if client.Transport != nil {
		t.Fatalf("expected nil Transport when no transport/retry/proxy/header options are set, got %T", client.Transport)
	}

	prev := http.DefaultTransport
	var intercepted atomic.Int32
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		intercepted.Add(1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		}, nil
	})
	defer func() { http.DefaultTransport = prev }()

	var out struct {
		OK bool `json:"ok"`
	}
	status, _, err := DoJSON(context.Background(), client, http.MethodGet, "https://example.invalid/test", nil, &out, nil, 1024)
	if err != nil || status != http.StatusOK || !out.OK || intercepted.Load() != 1 {
		t.Fatalf("unexpected DoJSON result: status=%d out=%+v err=%v intercepted=%d", status, out, err, intercepted.Load())
	}
}

func TestNoRedirectAndMaxRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/next", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	noRedir := New(WithNoRedirect())
	resp, err := noRedir.Get(srv.URL + "/start")
	if err != nil {
		t.Fatalf("Get with NoRedirect failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 without following redirect, got %d", resp.StatusCode)
	}

	oneRedir := New(WithMaxRedirects(1))
	resp2, err := oneRedir.Get(srv.URL + "/start")
	if err != nil {
		t.Fatalf("Get with MaxRedirects(1) failed: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after 1 redirect, got %d", resp2.StatusCode)
	}
}

func TestRetryAndCancel(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"ping":1}` {
			t.Errorf("attempt %d lost request body: %q", n, string(body))
		}
		if r.Header.Get("User-Agent") != "httpx-test/1.0" || r.Header.Get("X-Extra") != "yes" {
			t.Errorf("missing configured headers: UA=%q X-Extra=%q", r.Header.Get("User-Agent"), r.Header.Get("X-Extra"))
		}
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("busy"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	client := New(
		WithRetry(2, time.Millisecond, 5*time.Millisecond),
		WithUserAgent("httpx-test/1.0"),
		WithHeaders(map[string]string{"X-Extra": "yes"}),
	)
	var res struct {
		Status string `json:"status"`
	}
	status, _, err := DoJSON(context.Background(), client, http.MethodPost, srv.URL, map[string]int{"ping": 1}, &res, nil, 4096)
	if err != nil || status != http.StatusOK || res.Status != "ok" || calls.Load() != 3 {
		t.Fatalf("retry failed: status=%d res=%+v err=%v calls=%d", status, res, err, calls.Load())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	slowRetry := New(WithRetry(3, time.Second, time.Second))
	srv429 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv429.Close()

	start := time.Now()
	_, _, err = DoJSON(ctx, slowRetry, http.MethodGet, srv429.URL, nil, nil, nil, 1024)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded during retry sleep, got %v", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("retry sleep did not cancel promptly on context expiry: %v", time.Since(start))
	}
}

func TestProxyValidationAndBodyLimit(t *testing.T) {
	if _, err := NewWithError(WithProxyURL("://bad-proxy")); err == nil {
		t.Fatal("expected error for invalid proxy URL")
	}
	if _, err := NewWithError(WithProxyURL("http://127.0.0.1:8080")); err != nil {
		t.Fatalf("valid proxy rejected: %v", err)
	}

	if _, err := ReadBodyLimit(strings.NewReader("123456"), 5); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("expected ErrBodyTooLarge, got %v", err)
	}
	got, err := ReadBodyLimit(strings.NewReader("12345"), 5)
	if err != nil || string(got) != "12345" {
		t.Fatalf("exact limit failed: got=%q err=%v", string(got), err)
	}
}
