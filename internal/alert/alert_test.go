package alert

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testAlert() Alert {
	return Alert{
		Title:   "5xx rate high",
		Message: "error rate exceeded 5%",
		Level:   LevelCritical,
		Service: "order-service",
		Time:    time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC),
	}.WithField("error_rate", "7.2%")
}

func TestLarkFormatter(t *testing.T) {
	f := NewLarkFormatter()

	if ct := f.ContentType(); ct != "application/json" {
		t.Errorf("ContentType = %s, want application/json", ct)
	}

	body, err := f.Format(testAlert())
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var payload struct {
		MsgType string `json:"msg_type"`
		Content struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if payload.MsgType != "text" {
		t.Errorf("msg_type = %s, want text", payload.MsgType)
	}
	for _, want := range []string{"CRITICAL", "5xx rate high", "order-service", "error_rate", "7.2%"} {
		if !strings.Contains(payload.Content.Text, want) {
			t.Errorf("text missing %q: %s", want, payload.Content.Text)
		}
	}
}

func TestGotifyFormatter(t *testing.T) {
	f := NewGotifyFormatter("app-token-123")

	headers := f.Headers()
	if headers["X-Gotify-Key"] != "app-token-123" {
		t.Errorf("X-Gotify-Key = %q, want app-token-123", headers["X-Gotify-Key"])
	}

	body, err := f.Format(testAlert())
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	var payload struct {
		Title    string `json:"title"`
		Message  string `json:"message"`
		Priority int    `json:"priority"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !strings.Contains(payload.Title, "CRITICAL") {
		t.Errorf("title = %q, want level marker", payload.Title)
	}
	if payload.Priority != 10 {
		t.Errorf("priority = %d, want 10 for critical", payload.Priority)
	}
}

func TestWebhookNotifier_Feishu(t *testing.T) {
	var gotPath, gotCT, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Feishu custom bot webhook usually lives at this path.
	url := srv.URL + "/open-apis/bot/v2/hook/token"
	notifier := NewWebhookNotifier(url, NewLarkFormatter())

	if err := notifier.Notify(context.Background(), testAlert()); err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	if gotPath != "/open-apis/bot/v2/hook/token" {
		t.Errorf("path = %s, want webhook path", gotPath)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Errorf("Content-Type = %s", gotCT)
	}
	if !strings.Contains(gotBody, `"msg_type":"text"`) {
		t.Errorf("unexpected body: %s", gotBody)
	}
}

func TestWebhookNotifier_Gotify(t *testing.T) {
	var gotKey, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Gotify-Key")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	notifier := NewWebhookNotifier(srv.URL+"/message", NewGotifyFormatter("token-xyz"))

	if err := notifier.Notify(context.Background(), testAlert()); err != nil {
		t.Fatalf("Notify failed: %v", err)
	}

	if gotKey != "token-xyz" {
		t.Errorf("X-Gotify-Key = %q, want token-xyz", gotKey)
	}
	if !strings.Contains(gotBody, `"priority":10`) {
		t.Errorf("unexpected body: %s", gotBody)
	}
}

func TestWebhookNotifier_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	notifier := NewWebhookNotifier(srv.URL, NewLarkFormatter())
	if err := notifier.Notify(context.Background(), testAlert()); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestManager_CooldownDedup(t *testing.T) {
	var sent int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := NewManager(WithCooldown(time.Hour))
	m.AddNotifier(NewWebhookNotifier(srv.URL, NewLarkFormatter()))

	a := testAlert()
	if err := m.SendSync(context.Background(), a); err != nil {
		t.Fatalf("first send failed: %v", err)
	}

	// Same service+title within the cooldown window is deduplicated.
	if err := m.SendSync(context.Background(), a); err != nil {
		t.Fatalf("deduplicated send should not error: %v", err)
	}
	if sent != 1 {
		t.Errorf("expected 1 delivery after dedup, got %d", sent)
	}
}

func TestManager_NoDedupAfterCooldown(t *testing.T) {
	var sent int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Zero cooldown: every alert is delivered.
	m := NewManager(WithCooldown(0))
	m.AddNotifier(NewWebhookNotifier(srv.URL, NewLarkFormatter()))

	a := testAlert()
	m.SendSync(context.Background(), a)
	m.SendSync(context.Background(), a)

	if sent != 2 {
		t.Errorf("expected 2 deliveries with no cooldown, got %d", sent)
	}
}

func TestManager_AsyncSend(t *testing.T) {
	delivered := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delivered <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := NewManager()
	m.AddNotifier(NewWebhookNotifier(srv.URL, NewLarkFormatter()))
	m.Send(context.Background(), testAlert())

	select {
	case <-delivered:
	case <-time.After(2 * time.Second):
		t.Fatal("async delivery did not arrive")
	}
}
