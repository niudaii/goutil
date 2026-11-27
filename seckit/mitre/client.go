// Package mitre provides a shared client for querying CVE metadata and CVSS scores
// from the MITRE CVE v5 API (cveawg.mitre.org).
package mitre

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/niudaii/goutil/seckit/httpx"
)

// DefaultAPIFormat is the default MITRE CVE v5 API URL format string.
const DefaultAPIFormat = "https://cveawg.mitre.org/api/cve/%s"

const (
	defaultScoredTTL   = 6 * time.Hour
	defaultUnscoredTTL = time.Hour
	maxCacheEntries    = 4096
	maxRecordBytes     = 4 << 20
)

var (
	// ErrInvalidCVEID indicates the supplied string is not a valid CVE-YYYY-NNNN+ identifier.
	ErrInvalidCVEID = errors.New("invalid CVE ID")
	// ErrCVENotFound indicates the CVE does not exist upstream or is not in PUBLISHED state.
	ErrCVENotFound = errors.New("CVE not found or not published")

	cveIDPattern = regexp.MustCompile(`^CVE-[0-9]{4}-[0-9]{4,}$`)
)

// Score summarizes the highest-priority CVSS base score for a CVE while preserving
// the distinction between an unrated CVE (BaseScore == nil) and a valid 0.0 score.
type Score struct {
	ID           string   `json:"id"`
	BaseScore    *float64 `json:"baseScore"`
	Version      string   `json:"version,omitempty"`
	BaseSeverity string   `json:"baseSeverity,omitempty"`
	VectorString string   `json:"vectorString,omitempty"`
	Source       string   `json:"source,omitempty"`
}

// Affected describes an affected vendor/product entry in a CNA container.
type Affected struct {
	Vendor    string   `json:"vendor"`
	Product   string   `json:"product"`
	Platforms []string `json:"platforms"`
	Versions  []struct {
		Version     string `json:"version"`
		LessThan    string `json:"lessThan"`
		VersionType string `json:"versionType"`
		Status      string `json:"status"`
	} `json:"versions"`
}

// Description holds a localized vulnerability description.
type Description struct {
	Value string `json:"value"`
	Lang  string `json:"lang"`
}

// Container represents either the CNA or an ADP container inside a MITRE CVE v5 record.
type Container struct {
	Title            string                        `json:"title"`
	DatePublic       string                        `json:"datePublic"`
	Affected         []Affected                    `json:"affected"`
	Descriptions     []Description                 `json:"descriptions"`
	Metrics          []map[string]json.RawMessage  `json:"metrics"`
	ProviderMetadata struct {
		OrgID       string `json:"orgId"`
		ShortName   string `json:"shortName"`
		DateUpdated string `json:"dateUpdated"`
	} `json:"providerMetadata"`
}

// Record represents a MITRE CVE v5 JSON record.
type Record struct {
	DataType    string `json:"dataType"`
	DataVersion string `json:"dataVersion"`
	CVEMetadata struct {
		CveID             string `json:"cveId"`
		AssignerOrgID     string `json:"assignerOrgId"`
		State             string `json:"state"`
		AssignerShortName string `json:"assignerShortName"`
		DateReserved      string `json:"dateReserved"`
		DatePublished     string `json:"datePublished"`
		DateUpdated       string `json:"dateUpdated"`
	} `json:"cveMetadata"`
	Containers struct {
		CNA Container   `json:"cna"`
		ADP []Container `json:"adp"`
	} `json:"containers"`
}

// Products returns all product names listed under Containers.CNA.Affected.
func (r *Record) Products() []string {
	if r == nil {
		return nil
	}
	products := make([]string, 0, len(r.Containers.CNA.Affected))
	for _, item := range r.Containers.CNA.Affected {
		products = append(products, item.Product)
	}
	return products
}

// PreferredDescription selects the English description (en-US, then en/en-*), falling back to the first entry.
func (r *Record) PreferredDescription() string {
	if r == nil {
		return ""
	}
	descs := r.Containers.CNA.Descriptions
	for _, d := range descs {
		if d.Lang == "en-US" {
			return d.Value
		}
	}
	for _, d := range descs {
		if d.Lang == "en" || strings.HasPrefix(d.Lang, "en-") {
			return d.Value
		}
	}
	if len(descs) > 0 {
		return descs[0].Value
	}
	return ""
}

// Score extracts the CVSS score for r using its metadata CVE ID.
func (r *Record) Score() *Score {
	if r == nil {
		return &Score{}
	}
	return r.ScoreFor(r.CVEMetadata.CveID)
}

var cvssVersionOrder = []struct {
	keys  []string
	label string
}{
	{keys: []string{"cvssV4_0"}, label: "4.0"},
	{keys: []string{"cvssV3_1"}, label: "3.1"},
	{keys: []string{"cvssV3_0"}, label: "3.0"},
	{keys: []string{"cvssV2_0", "cvssV2"}, label: "2.0"},
}

type rawCVSSMetric struct {
	BaseScore    *float64 `json:"baseScore"`
	BaseSeverity string   `json:"baseSeverity"`
	VectorString string   `json:"vectorString"`
}

// ScoreFor prefers CNA over ADP and newer CVSS versions (4.0 > 3.1 > 3.0 > 2.0),
// selecting the highest valid BaseScore within the same version.
func (r *Record) ScoreFor(id string) *Score {
	result := &Score{ID: id}
	if r == nil {
		return result
	}
	for _, containers := range [][]Container{{r.Containers.CNA}, r.Containers.ADP} {
		for _, ver := range cvssVersionOrder {
			for _, container := range containers {
				for _, metric := range container.Metrics {
					for _, key := range ver.keys {
						raw, ok := metric[key]
						if !ok {
							continue
						}
						var value rawCVSSMetric
						if json.Unmarshal(raw, &value) != nil || value.BaseScore == nil || *value.BaseScore < 0 || *value.BaseScore > 10 {
							continue
						}
						if result.BaseScore == nil || *value.BaseScore > *result.BaseScore {
							scoreCopy := *value.BaseScore
							result.BaseScore = &scoreCopy
							result.Version = ver.label
							result.BaseSeverity = value.BaseSeverity
							result.VectorString = value.VectorString
							result.Source = container.ProviderMetadata.ShortName
						}
					}
				}
			}
			if result.BaseScore != nil {
				return result
			}
		}
	}
	return result
}

// ExtractSeverityAndScore returns severity and numeric score, falling back from CNA to ADP
// and returning ("UNKNOWN", 0) when no CVSS metrics are present.
func (r *Record) ExtractSeverityAndScore() (severity string, score float64) {
	if r == nil {
		return "UNKNOWN", 0
	}
	if s := r.Score(); s.BaseScore != nil {
		return s.BaseSeverity, *s.BaseScore
	}
	for _, containers := range [][]Container{{r.Containers.CNA}, r.Containers.ADP} {
		for _, ver := range cvssVersionOrder {
			for _, container := range containers {
				for _, metric := range container.Metrics {
					for _, key := range ver.keys {
						raw, ok := metric[key]
						if !ok {
							continue
						}
						var value rawCVSSMetric
						if json.Unmarshal(raw, &value) == nil && value.BaseSeverity != "" {
							return value.BaseSeverity, 0
						}
					}
				}
			}
		}
	}
	return "UNKNOWN", 0
}

type cachedEntry struct {
	record    *Record
	score     *Score
	expiresAt time.Time
}

// Client queries and caches MITRE CVE v5 records.
type Client struct {
	httpClient  *http.Client
	apiFormat   string
	scoredTTL   time.Duration
	unscoredTTL time.Duration
	now         func() time.Time

	mu    sync.Mutex
	cache map[string]cachedEntry
}

// Option customizes a Client created by NewClient.
type Option func(*Client)

// WithAPIFormat overrides the URL format string (defaults to DefaultAPIFormat).
func WithAPIFormat(format string) Option {
	return func(c *Client) {
		if strings.TrimSpace(format) != "" {
			c.apiFormat = format
		}
	}
}

// WithCacheTTL overrides cache TTLs for scored and unscored CVE records.
func WithCacheTTL(scored, unscored time.Duration) Option {
	return func(c *Client) {
		if scored > 0 {
			c.scoredTTL = scored
		}
		if unscored > 0 {
			c.unscoredTTL = unscored
		}
	}
}

// WithNow overrides the clock function used for cache expiration.
func WithNow(fn func() time.Time) Option {
	return func(c *Client) {
		if fn != nil {
			c.now = fn
		}
	}
}

// NewClient constructs a MITRE CVE API client. If httpClient is nil, a default
// httpx client with timeout, no redirects, and retry is used.
func NewClient(httpClient *http.Client, opts ...Option) *Client {
	if httpClient == nil {
		httpClient = httpx.New(
			httpx.WithTimeout(15*time.Second),
			httpx.WithNoRedirect(),
			httpx.WithRetry(2, 500*time.Millisecond, 3*time.Second),
		)
	}
	c := &Client{
		httpClient:  httpClient,
		apiFormat:   DefaultAPIFormat,
		scoredTTL:   defaultScoredTTL,
		unscoredTTL: defaultUnscoredTTL,
		now:         time.Now,
		cache:       make(map[string]cachedEntry),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// FetchRecord fetches, validates, and caches a published MITRE CVE v5 record.
func (c *Client) FetchRecord(ctx context.Context, id string) (*Record, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := strings.ToUpper(strings.TrimSpace(id))
	if !cveIDPattern.MatchString(normalized) {
		return nil, ErrInvalidCVEID
	}

	now := c.now()
	c.mu.Lock()
	if entry, ok := c.cache[normalized]; ok && now.Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.record, nil
	}
	c.mu.Unlock()

	var record Record
	status, _, err := httpx.DoJSON(ctx, c.httpClient, http.MethodGet, fmt.Sprintf(c.apiFormat, normalized), nil, &record, nil, maxRecordBytes)
	if status == http.StatusNotFound {
		return nil, ErrCVENotFound
	}
	if err != nil {
		if status != 0 && status != http.StatusOK {
			return nil, fmt.Errorf("MITRE CVE API: HTTP %d", status)
		}
		return nil, err
	}
	if record.CVEMetadata.CveID != normalized || record.CVEMetadata.State == "" {
		return nil, fmt.Errorf("MITRE CVE API returned an invalid record for %s", normalized)
	}
	if record.CVEMetadata.State != "PUBLISHED" {
		return nil, ErrCVENotFound
	}

	score := record.ScoreFor(normalized)
	ttl := c.scoredTTL
	if score.BaseScore == nil {
		ttl = c.unscoredTTL
	}

	c.mu.Lock()
	c.pruneLocked(now)
	c.cache[normalized] = cachedEntry{
		record:    &record,
		score:     score,
		expiresAt: now.Add(ttl),
	}
	c.mu.Unlock()

	return &record, nil
}

// GetScore returns the parsed CVSS score for id, serving from cache when fresh.
func (c *Client) GetScore(ctx context.Context, id string) (*Score, error) {
	normalized := strings.ToUpper(strings.TrimSpace(id))
	if !cveIDPattern.MatchString(normalized) {
		return nil, ErrInvalidCVEID
	}
	now := c.now()
	c.mu.Lock()
	if entry, ok := c.cache[normalized]; ok && now.Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.score, nil
	}
	c.mu.Unlock()

	record, err := c.FetchRecord(ctx, normalized)
	if err != nil {
		return nil, err
	}
	return record.ScoreFor(normalized), nil
}

// GetProducts returns the affected product list from the CVE's CNA container.
func (c *Client) GetProducts(ctx context.Context, id string) ([]string, error) {
	record, err := c.FetchRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	return record.Products(), nil
}

func (c *Client) pruneLocked(now time.Time) {
	for k, v := range c.cache {
		if !now.Before(v.expiresAt) {
			delete(c.cache, k)
		}
	}
	for len(c.cache) >= maxCacheEntries {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.cache {
			if oldestKey == "" || v.expiresAt.Before(oldestTime) {
				oldestKey, oldestTime = k, v.expiresAt
			}
		}
		if oldestKey == "" {
			break
		}
		delete(c.cache, oldestKey)
	}
}
