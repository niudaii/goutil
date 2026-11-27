package mitre

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestClientScorePriorityAndCache(t *testing.T) {
	var hits atomic.Int32
	payload := `{
		"cveMetadata": {"cveId": "CVE-2026-10001", "state": "PUBLISHED"},
		"containers": {
			"cna": {
				"providerMetadata": {"shortName": "cna-org"},
				"affected": [{"vendor": "Acme", "product": "AcmeWidget"}],
				"descriptions": [{"lang": "zh", "value": "中文"}, {"lang": "en-US", "value": "English detail"}],
				"metrics": [
					{"cvssV3_1": {"baseScore": 7.5, "baseSeverity": "HIGH"}},
					{"cvssV4_0": {"baseScore": 8.9, "baseSeverity": "HIGH", "vectorString": "CVSS:4.0/..."}}
				]
			},
			"adp": [{
				"providerMetadata": {"shortName": "CISA-ADP"},
				"metrics": [{"cvssV4_0": {"baseScore": 9.9, "baseSeverity": "CRITICAL"}}]
			}]
		}
	}`
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		hits.Add(1)
		if r.URL.String() != "https://cveawg.mitre.org/api/cve/CVE-2026-10001" {
			t.Fatalf("unexpected URL: %s", r.URL.String())
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(payload)),
		}, nil
	})}

	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	c := NewClient(hc, WithNow(func() time.Time { return now }))

	score, err := c.GetScore(context.Background(), " cve-2026-10001 ")
	if err != nil {
		t.Fatalf("GetScore failed: %v", err)
	}
	if score.BaseScore == nil || *score.BaseScore != 8.9 || score.Version != "4.0" || score.Source != "cna-org" {
		t.Fatalf("CNA CVSS 4.0 must take precedence over ADP 4.0: %+v", score)
	}

	products, err := c.GetProducts(context.Background(), "CVE-2026-10001")
	if err != nil || len(products) != 1 || products[0] != "AcmeWidget" {
		t.Fatalf("GetProducts failed: products=%v err=%v", products, err)
	}
	rec, err := c.FetchRecord(context.Background(), "CVE-2026-10001")
	if err != nil || rec.PreferredDescription() != "English detail" {
		t.Fatalf("FetchRecord/PreferredDescription failed: %v", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected cached record after 1 fetch, got %d hits", hits.Load())
	}

	now = now.Add(7 * time.Hour)
	if _, err := c.GetScore(context.Background(), "CVE-2026-10001"); err != nil || hits.Load() != 2 {
		t.Fatalf("expected cache refresh after TTL, hits=%d err=%v", hits.Load(), err)
	}
}

func TestClientValidationAndUnscored(t *testing.T) {
	hc := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/api/cve/CVE-2026-0001":
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
		case "/api/cve/CVE-2026-0002":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"cveMetadata":{"cveId":"CVE-2026-0002","state":"REJECTED"}}`)),
			}, nil
		case "/api/cve/CVE-2026-0003":
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"cveMetadata":{"cveId":"CVE-2026-0003","state":"PUBLISHED"},"containers":{"cna":{"metrics":[{"cvssV3_1":{"baseScore":0.0,"baseSeverity":"NONE"}}]}}}`)),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"cveMetadata":{"cveId":"CVE-2026-0004","state":"PUBLISHED"},"containers":{"cna":{"metrics":[]},"adp":[{"metrics":[{"cvssV2":{"baseScore":5.0,"baseSeverity":"MEDIUM"}}]}]}}`)),
			}, nil
		}
	})}
	c := NewClient(hc)

	if _, err := c.GetScore(context.Background(), "not-a-cve"); !errors.Is(err, ErrInvalidCVEID) {
		t.Fatalf("expected ErrInvalidCVEID, got %v", err)
	}
	if _, err := c.GetScore(context.Background(), "CVE-2026-0001"); !errors.Is(err, ErrCVENotFound) {
		t.Fatalf("expected ErrCVENotFound on 404, got %v", err)
	}
	if _, err := c.GetScore(context.Background(), "CVE-2026-0002"); !errors.Is(err, ErrCVENotFound) {
		t.Fatalf("expected ErrCVENotFound on REJECTED, got %v", err)
	}
	zero, err := c.GetScore(context.Background(), "CVE-2026-0003")
	if err != nil || zero.BaseScore == nil || *zero.BaseScore != 0.0 {
		t.Fatalf("valid 0.0 score must be preserved: %+v err=%v", zero, err)
	}
	adp, err := c.GetScore(context.Background(), "CVE-2026-0004")
	if err != nil || adp.BaseScore == nil || *adp.BaseScore != 5.0 || adp.Version != "2.0" {
		t.Fatalf("ADP cvssV2 fallback failed: %+v err=%v", adp, err)
	}
}
