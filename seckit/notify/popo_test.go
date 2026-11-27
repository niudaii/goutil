package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWebhookSendLogsAndReturnsError(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	empty := NewWebhook("", "", WithLogger(logger))
	if err := empty.Send(context.Background(), "hello"); !errors.Is(err, ErrEmptyWebhook) {
		t.Fatalf("expected ErrEmptyWebhook, got %v", err)
	}
	if !strings.Contains(logs.String(), "empty webhook URL") {
		t.Fatalf("expected logged error on empty webhook, got %s", logs.String())
	}

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["message"] != "alert" || body["signData"] != "sec" {
			t.Errorf("unexpected webhook payload: %v", body)
		}
		if calls.Load() == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
			return
		}
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	logs.Reset()
	okClient := NewWebhook(srv.URL, "sec", WithHTTPClient(srv.Client()), WithLogger(logger))
	if err := okClient.Send(context.Background(), "alert"); err == nil {
		t.Fatal("expected error without retry client when upstream returns 500")
	}
	if !strings.Contains(logs.String(), "POPO webhook send failed") {
		t.Fatalf("expected error log, got %s", logs.String())
	}
	if err := okClient.Send(context.Background(), "alert"); err != nil {
		t.Fatalf("expected success on 200, got %v", err)
	}

	badBizSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":4001,"errmsg":"invalid"}`))
	}))
	defer badBizSrv.Close()
	if err := NewWebhook(badBizSrv.URL, "", WithHTTPClient(badBizSrv.Client()), WithLogger(logger)).Send(context.Background(), "x"); err == nil {
		t.Fatal("expected error for non-zero POPO errcode")
	}
}

func TestWebhookSendChunkedAndSplits(t *testing.T) {
	var sent []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		sent = append(sent, body.Message)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewWebhook(srv.URL, "", WithHTTPClient(srv.Client()))
	err := c.SendChunked(context.Background(), "新增 %d 条", "\n", []string{"条目一", "条目二", "条目三"}, 10, 4)
	if err != nil {
		t.Fatalf("SendChunked failed: %v", err)
	}
	if len(sent) != 3 || !strings.Contains(sent[0], "新增 3 条（1/3）") || !strings.Contains(sent[2], "（3/3）") {
		t.Fatalf("unexpected chunked messages: %v", sent)
	}

	bytesSegs := SplitByBytes(strings.Repeat("中", 10), 10)
	if len(bytesSegs) != 4 {
		t.Fatalf("expected 4 UTF-8 segments of <= 10 bytes (3 Chinese chars = 9 bytes), got %d: %v", len(bytesSegs), bytesSegs)
	}
	for _, s := range bytesSegs {
		if len(s) > 10 {
			t.Fatalf("segment exceeds byte budget: %d", len(s))
		}
	}
}

func TestRobotSendMessageAndFile(t *testing.T) {
	var tokenHits atomic.Int32
	var sentPayloads []map[string]any
	var uploadedFile atomic.Bool

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenHits.Add(1)
			_, _ = w.Write([]byte(`{"code":0,"data":{"accessToken":"tok-1","expiresIn":3600}}`))
		case "/im/send-msg":
			if r.Header.Get("Open-Access-Token") != "tok-1" {
				t.Errorf("missing token header")
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			sentPayloads = append(sentPayloads, body)
			_, _ = w.Write([]byte(`{"code":0,"errcode":0}`))
		case "/im/file":
			_, _ = io.WriteString(w, `{"errcode":0,"data":{"fileKey":"fk-1","uploadUrl":"`+srv.URL+`/upload"}}`)
		case "/upload":
			uploadedFile.Store(true)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	robot := NewRobot("key", "secret", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()), WithSegmentDelay(0))
	if err := robot.SendMessage(context.Background(), "7103665-alice@example.com", "hello"); err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if len(sentPayloads) != 1 || sentPayloads[0]["receiver"] != "7103665" {
		t.Fatalf("unexpected group-user receiver resolution: %+v", sentPayloads)
	}
	msgBody := sentPayloads[0]["message"].(map[string]any)
	if !strings.HasPrefix(msgBody["content"].(string), "@alice@example.com ") {
		t.Fatalf("expected @mention prefix, got %q", msgBody["content"])
	}

	tmpFile := filepath.Join(t.TempDir(), "report.csv")
	if err := os.WriteFile(tmpFile, []byte("a,b\n1,2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := robot.SendFile(context.Background(), "alice@example.com", tmpFile, "csv"); err != nil {
		t.Fatalf("SendFile failed: %v", err)
	}
	if !uploadedFile.Load() || len(sentPayloads) != 2 || sentPayloads[1]["msgType"] != "file" {
		t.Fatalf("unexpected file send state: uploaded=%v payloads=%+v", uploadedFile.Load(), sentPayloads)
	}
	if tokenHits.Load() != 1 {
		t.Fatalf("expected cached token reuse, got %d token calls", tokenHits.Load())
	}
}
