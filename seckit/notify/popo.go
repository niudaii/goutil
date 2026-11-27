// Package notify provides POPO Webhook and OpenAPI Robot notification clients
// that always log failures and return errors to callers.
package notify

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/niudaii/goutil/seckit/httpx"
)

const (
	// DefaultPopoBaseURL is the base URL for the POPO OpenAPI Robot service.
	DefaultPopoBaseURL = "https://open.popo.netease.com/open-apis/robots/v1"

	// MaxRobotMessageBytes is the conservative UTF-8 byte budget per POPO Robot text message
	// to avoid silent server-side truncation of multi-byte CJK payloads.
	MaxRobotMessageBytes = 2000

	// MaxUploadFileSize caps POPO file uploads at 100 MiB.
	MaxUploadFileSize int64 = 100 * 1024 * 1024

	maxResponseBytes = 64 << 10
)

// ErrEmptyWebhook indicates a WebhookClient has no destination URL configured.
var ErrEmptyWebhook = errors.New("notify: webhook URL is empty")

type options struct {
	httpClient      *http.Client
	logger          *slog.Logger
	baseURL         string
	segmentDelay    time.Duration
	hasSegmentDelay bool
}

// Option customizes WebhookClient or RobotClient behavior.
type Option func(*options)

// WithHTTPClient overrides the HTTP client used for POPO requests.
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) {
		o.httpClient = client
	}
}

// WithLogger overrides the slog.Logger used to record send failures.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithBaseURL overrides DefaultPopoBaseURL for RobotClient.
func WithBaseURL(baseURL string) Option {
	return func(o *options) {
		if strings.TrimSpace(baseURL) != "" {
			o.baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
		}
	}
}

// WithSegmentDelay overrides the pause between multi-segment Robot messages.
func WithSegmentDelay(d time.Duration) Option {
	return func(o *options) {
		o.segmentDelay = d
		o.hasSegmentDelay = true
	}
}

func resolveOptions(defaultClient *http.Client, opts []Option) options {
	cfg := options{
		httpClient:   defaultClient,
		logger:       slog.Default(),
		baseURL:      DefaultPopoBaseURL,
		segmentDelay: time.Second,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.logger == nil {
		cfg.logger = slog.Default()
	}
	if cfg.httpClient == nil {
		cfg.httpClient = defaultClient
	}
	return cfg
}

// WebhookClient sends messages to a POPO group webhook endpoint.
type WebhookClient struct {
	webhookURL string
	sign       string
	client     *http.Client
	logger     *slog.Logger
}

// NewWebhook creates a POPO WebhookClient.
func NewWebhook(webhookURL, sign string, opts ...Option) *WebhookClient {
	defaultClient := httpx.New(
		httpx.WithTimeout(15*time.Second),
		httpx.WithNoRedirect(),
		httpx.WithRetry(2, time.Second, 5*time.Second),
	)
	cfg := resolveOptions(defaultClient, opts)
	return &WebhookClient{
		webhookURL: strings.TrimSpace(webhookURL),
		sign:       sign,
		client:     cfg.httpClient,
		logger:     cfg.logger,
	}
}

// Enabled reports whether a non-empty webhook URL is configured.
func (c *WebhookClient) Enabled() bool {
	return c != nil && c.webhookURL != ""
}

// Send posts a single message to the configured POPO webhook, logging and returning any error.
func (c *WebhookClient) Send(ctx context.Context, msg string) error {
	if !c.Enabled() {
		if c != nil && c.logger != nil {
			c.logger.Error("POPO webhook send skipped: empty webhook URL")
		}
		return ErrEmptyWebhook
	}
	if ctx == nil {
		ctx = context.Background()
	}

	payload := map[string]any{"message": msg}
	if c.sign != "" {
		payload["signData"] = c.sign
		payload["timestamp"] = time.Now().Unix()
	}

	status, body, err := httpx.DoJSON(ctx, c.client, http.MethodPost, c.webhookURL, payload, nil, nil, maxResponseBytes)
	if err != nil {
		if len(body) > 0 {
			err = fmt.Errorf("notify: webhook HTTP %d: %s", status, strings.TrimSpace(string(body)))
		} else {
			err = fmt.Errorf("notify: webhook send failed: %w", err)
		}
		c.logger.Error("POPO webhook send failed", "err", err)
		return err
	}
	if apiErr := parsePopoAPIError(body, true); apiErr != nil {
		c.logger.Error("POPO webhook rejected message", "err", apiErr)
		return apiErr
	}
	return nil
}

// SendChunked splits items at rune boundaries and sends each chunk with headerFormat and sep.
func (c *WebhookClient) SendChunked(ctx context.Context, headerFormat, sep string, items []string, maxRunes, headerReserve int) error {
	if len(items) == 0 {
		return nil
	}
	limit := maxRunes - headerReserve
	if limit <= 0 {
		limit = maxRunes
	}
	chunks := SplitByRunes(items, sep, limit)
	var errs []error
	for i, chunk := range chunks {
		header := headerFormat
		if strings.Contains(headerFormat, "%") {
			header = fmt.Sprintf(headerFormat, len(items))
		}
		if len(chunks) > 1 {
			header += fmt.Sprintf("（%d/%d）", i+1, len(chunks))
		}
		if err := c.Send(ctx, header+sep+strings.Join(chunk, sep)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// SplitByRunes groups items along boundaries so joining each chunk with sep does not exceed limit runes.
// A single item exceeding limit runes is placed in its own chunk.
func SplitByRunes(items []string, sep string, limit int) [][]string {
	if len(items) == 0 {
		return nil
	}
	sepLen := utf8.RuneCountInString(sep)
	chunks := make([][]string, 0, 1)
	chunk := make([]string, 0, len(items))
	size := 0

	for _, item := range items {
		itemLen := utf8.RuneCountInString(item)
		cost := itemLen
		if len(chunk) > 0 {
			cost += sepLen
		}
		if len(chunk) > 0 && size+cost > limit {
			chunks = append(chunks, chunk)
			chunk = make([]string, 0, len(items))
			size = 0
			cost = itemLen
		}
		chunk = append(chunk, item)
		size += cost
	}
	if len(chunk) > 0 {
		chunks = append(chunks, chunk)
	}
	return chunks
}

// SplitByBytes splits content on rune boundaries into segments of at most maxBytes UTF-8 bytes,
// preferring newline, sentence, clause, or whitespace break points.
func SplitByBytes(content string, maxBytes int) []string {
	if maxBytes <= 0 || len(content) <= maxBytes {
		return []string{content}
	}

	var segments []string
	runes := []rune(content)
	for len(runes) > 0 {
		end := len(runes)
		for end > 0 && len(string(runes[:end])) > maxBytes {
			end--
		}
		if end == 0 {
			end = 1
		}
		if end < len(runes) {
			end = chooseSplitPoint(runes, end)
		}
		segments = append(segments, string(runes[:end]))
		runes = runes[end:]
	}
	return segments
}

func chooseSplitPoint(runes []rune, originalEnd int) int {
	searchBack := 100
	if searchBack > originalEnd {
		searchBack = originalEnd
	}
	for _, set := range []string{"\n", "。!？.", "，；,;"} {
		for i := originalEnd - 1; i >= originalEnd-searchBack && i >= 0; i-- {
			if strings.ContainsRune(set, runes[i]) {
				return i + 1
			}
		}
	}
	searchBackSmall := 50
	if searchBackSmall > originalEnd {
		searchBackSmall = originalEnd
	}
	for i := originalEnd - 1; i >= originalEnd-searchBackSmall && i >= 0; i-- {
		if runes[i] == ' ' || runes[i] == '\t' {
			return i + 1
		}
	}
	return originalEnd
}

// RobotClient sends text and file messages through the POPO OpenAPI Robot endpoints.
type RobotClient struct {
	appKey       string
	appSecret    string
	baseURL      string
	client       *http.Client
	logger       *slog.Logger
	segmentDelay time.Duration

	mu          sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

// NewRobot creates a POPO RobotClient.
func NewRobot(appKey, appSecret string, opts ...Option) *RobotClient {
	defaultClient := httpx.New(
		httpx.WithTimeout(30*time.Second),
		httpx.WithNoRedirect(),
	)
	cfg := resolveOptions(defaultClient, opts)
	return &RobotClient{
		appKey:       appKey,
		appSecret:    appSecret,
		baseURL:      cfg.baseURL,
		client:       cfg.httpClient,
		logger:       cfg.logger,
		segmentDelay: cfg.segmentDelay,
	}
}

// GetAccessToken returns a cached access token or fetches a new one from POPO.
func (r *RobotClient) GetAccessToken(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.RLock()
	if r.accessToken != "" && time.Now().Before(r.tokenExpiry) {
		tok := r.accessToken
		r.mu.RUnlock()
		return tok, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.accessToken != "" && time.Now().Before(r.tokenExpiry) {
		return r.accessToken, nil
	}

	if strings.TrimSpace(r.appKey) == "" || strings.TrimSpace(r.appSecret) == "" {
		err := errors.New("notify: POPO appKey and appSecret are required")
		r.logger.Error("POPO robot credentials missing")
		return "", err
	}

	var tokenResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken string `json:"accessToken"`
			ExpiresIn   int64  `json:"expiresIn"`
		} `json:"data"`
	}
	status, body, err := httpx.DoJSON(ctx, r.client, http.MethodPost, r.baseURL+"/token", map[string]string{
		"appKey":    r.appKey,
		"appSecret": r.appSecret,
	}, &tokenResp, nil, maxResponseBytes)
	if err != nil {
		wrapped := fmt.Errorf("notify: get POPO token failed (HTTP %d): %w", status, err)
		if len(body) > 0 {
			wrapped = fmt.Errorf("notify: get POPO token failed (HTTP %d): %s", status, strings.TrimSpace(string(body)))
		}
		r.logger.Error("POPO token request failed", "err", wrapped)
		return "", wrapped
	}
	if tokenResp.Code != 0 || tokenResp.Data.AccessToken == "" {
		err := fmt.Errorf("notify: get POPO token rejected (code=%d, message=%s)", tokenResp.Code, tokenResp.Message)
		r.logger.Error("POPO token rejected", "code", tokenResp.Code, "message", tokenResp.Message)
		return "", err
	}

	ttl := time.Duration(tokenResp.Data.ExpiresIn)*time.Second - 5*time.Minute
	if ttl <= 0 {
		ttl = time.Minute
	}
	r.accessToken = tokenResp.Data.AccessToken
	r.tokenExpiry = time.Now().Add(ttl)
	return r.accessToken, nil
}

// SendMessage sends a text message to a POPO user email or group ID (supports "groupID-user@domain").
func (r *RobotClient) SendMessage(ctx context.Context, receiver, content string) error {
	return r.SendMessageWithAt(ctx, receiver, content, nil)
}

// SendMessageWithAt sends a text message with optional @mentions, splitting long messages into UTF-8 byte segments.
func (r *RobotClient) SendMessageWithAt(ctx context.Context, receiver, content string, atUids []string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	resolvedReceiver, resolvedAt := resolveReceiverAndAt(receiver, atUids)
	atPrefixBytes := 0
	for _, uid := range resolvedAt {
		atPrefixBytes += len("@"+uid) + 1
	}
	maxContentBytes := MaxRobotMessageBytes - atPrefixBytes
	if maxContentBytes < 500 {
		maxContentBytes = 500
	}

	segments := SplitByBytes(content, maxContentBytes)
	for i, seg := range segments {
		if err := r.sendSingleText(ctx, resolvedReceiver, seg, resolvedAt); err != nil {
			if len(segments) > 1 {
				return fmt.Errorf("notify: send segment %d/%d: %w", i+1, len(segments), err)
			}
			return err
		}
		if i < len(segments)-1 && r.segmentDelay > 0 {
			timer := time.NewTimer(r.segmentDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return nil
}

func (r *RobotClient) sendSingleText(ctx context.Context, receiver, content string, atUids []string) error {
	token, err := r.GetAccessToken(ctx)
	if err != nil {
		return err
	}
	finalContent := content
	if len(atUids) > 0 {
		mentions := make([]string, len(atUids))
		for i, uid := range atUids {
			mentions[i] = "@" + uid
		}
		finalContent = strings.Join(mentions, " ") + " " + content
	}
	payload := map[string]any{
		"receiver": receiver,
		"msgType":  "text",
		"message": map[string]any{
			"content": finalContent,
			"atUids":  atUids,
		},
	}
	return r.sendRawMessage(ctx, token, payload)
}

func (r *RobotClient) sendRawMessage(ctx context.Context, token string, payload any) error {
	status, body, err := httpx.DoJSON(
		ctx,
		r.client,
		http.MethodPost,
		r.baseURL+"/im/send-msg",
		payload,
		nil,
		map[string]string{"Open-Access-Token": token},
		maxResponseBytes,
	)
	if err != nil {
		wrapped := fmt.Errorf("notify: POPO send-msg failed (HTTP %d): %w", status, err)
		if len(body) > 0 {
			wrapped = fmt.Errorf("notify: POPO send-msg failed (HTTP %d): %s", status, strings.TrimSpace(string(body)))
		}
		r.logger.Error("POPO send-msg failed", "err", wrapped)
		return wrapped
	}
	if apiErr := parsePopoAPIError(body, false); apiErr != nil {
		r.logger.Error("POPO send-msg rejected", "err", apiErr)
		return apiErr
	}
	return nil
}

// SendFile uploads filePath and sends a file message to receiver.
func (r *RobotClient) SendFile(ctx context.Context, receiver, filePath, fileType string) error {
	return r.SendFileWithAt(ctx, receiver, filePath, fileType, nil)
}

// SendFileWithAt uploads filePath and sends a file message with optional @mentions.
func (r *RobotClient) SendFileWithAt(ctx context.Context, receiver, filePath, fileType string, atUids []string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	fi, err := os.Stat(filePath)
	if err != nil {
		r.logger.Error("POPO stat upload file failed", "err", err)
		return fmt.Errorf("notify: stat file: %w", err)
	}
	if fi.Size() > MaxUploadFileSize {
		err := fmt.Errorf("notify: file too large (%d > %d)", fi.Size(), MaxUploadFileSize)
		r.logger.Error("POPO upload file too large", "size", fi.Size(), "limit", MaxUploadFileSize)
		return err
	}

	md5Hex, err := fileMD5(filePath)
	if err != nil {
		r.logger.Error("POPO compute file md5 failed", "err", err)
		return err
	}

	uploadReq := map[string]any{
		"fileType": fileType,
		"fileName": filepath.Base(filePath),
		"fileMd5":  md5Hex,
	}

	const maxAttempts = 3
	var fileKey string
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		token, tokErr := r.GetAccessToken(ctx)
		if tokErr != nil {
			return tokErr
		}
		var uploadResp struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
			Data    struct {
				FileKey   string `json:"fileKey"`
				UploadUrl string `json:"uploadUrl"`
			} `json:"data"`
		}
		status, body, gerr := httpx.DoJSON(
			ctx,
			r.client,
			http.MethodPost,
			r.baseURL+"/im/file",
			uploadReq,
			&uploadResp,
			map[string]string{"Open-Access-Token": token},
			maxResponseBytes,
		)
		if gerr != nil {
			lastErr = fmt.Errorf("notify: get upload url (attempt %d/%d, HTTP %d): %w", attempt, maxAttempts, status, gerr)
			if len(body) > 0 {
				lastErr = fmt.Errorf("notify: get upload url (attempt %d/%d, HTTP %d): %s", attempt, maxAttempts, status, strings.TrimSpace(string(body)))
			}
		} else if uploadResp.ErrCode != 0 {
			lastErr = fmt.Errorf("notify: get upload url errcode=%d errmsg=%s", uploadResp.ErrCode, uploadResp.ErrMsg)
		} else if uerr := r.uploadFile(ctx, token, uploadResp.Data.UploadUrl, filePath); uerr != nil {
			lastErr = fmt.Errorf("notify: upload file (attempt %d/%d): %w", attempt, maxAttempts, uerr)
			if !strings.Contains(uerr.Error(), "HTTP 5") && !strings.Contains(uerr.Error(), "upload request") {
				r.logger.Error("POPO file upload failed", "err", lastErr)
				return lastErr
			}
		} else {
			fileKey = uploadResp.Data.FileKey
			lastErr = nil
			break
		}

		if attempt < maxAttempts {
			delay := time.Duration(attempt) * r.segmentDelay
			if delay > 0 {
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return errors.Join(lastErr, ctx.Err())
				case <-timer.C:
				}
			}
		}
	}
	if lastErr != nil {
		r.logger.Error("POPO file upload failed", "err", lastErr)
		return lastErr
	}

	token, err := r.GetAccessToken(ctx)
	if err != nil {
		return err
	}
	resolvedReceiver, resolvedAt := resolveReceiverAndAt(receiver, atUids)
	payload := map[string]any{
		"receiver": resolvedReceiver,
		"msgType":  "file",
		"message": map[string]any{
			"fileKey": fileKey,
			"atUids":  resolvedAt,
		},
	}
	return r.sendRawMessage(ctx, token, payload)
}

func (r *RobotClient) uploadFile(ctx context.Context, token, uploadURL, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create multipart field: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copy file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &body)
	if err != nil {
		return fmt.Errorf("build upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Open-Access-Token", token)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := httpx.ReadBodyLimit(resp.Body, maxResponseBytes)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func fileMD5(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("notify: open file: %w", err)
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("notify: compute md5: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func resolveReceiverAndAt(receiver string, atUids []string) (string, []string) {
	outAt := make([]string, 0, len(atUids)+1)
	seen := make(map[string]bool, len(atUids)+1)
	for _, uid := range atUids {
		if uid != "" && !seen[uid] {
			seen[uid] = true
			outAt = append(outAt, uid)
		}
	}
	if strings.Contains(receiver, "-") {
		groupID, userEmail, ok := strings.Cut(receiver, "-")
		if ok {
			receiver = groupID
			if userEmail != "" && !seen[userEmail] {
				outAt = append(outAt, userEmail)
			}
		}
	}
	return receiver, outAt
}

func parsePopoAPIError(body []byte, allowHTTP200Code bool) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil
	}
	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if json.Unmarshal(trimmed, &parsed) != nil {
		return nil
	}
	if parsed.ErrCode != 0 {
		return fmt.Errorf("notify: POPO errcode=%d errmsg=%s", parsed.ErrCode, parsed.ErrMsg)
	}
	if parsed.Code != 0 && !(allowHTTP200Code && parsed.Code == http.StatusOK) {
		return fmt.Errorf("notify: POPO code=%d message=%s", parsed.Code, parsed.Message)
	}
	return nil
}
