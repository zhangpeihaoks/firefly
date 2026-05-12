package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewServerDefaults(t *testing.T) {
	srv := NewServer()
	if srv.address != ":8081" {
		t.Errorf("expected :8081, got %s", srv.address)
	}
	if srv.handlers == nil {
		t.Error("handlers map should be initialized")
	}
}

func TestOnMessage(t *testing.T) {
	srv := NewServer()
	srv.OnMessage("/chat", func(ctx context.Context, msg []byte) ([]byte, error) {
		return nil, nil
	})
	entry, ok := srv.handlers["/chat"]
	if !ok {
		t.Error("handler not registered")
	}
	if entry.handler == nil {
		t.Error("handler function should not be nil")
	}
}

func TestOptions(t *testing.T) {
	srv := NewServer(Address(":9000"), Timeout(60*time.Second))
	if srv.address != ":9000" {
		t.Errorf("expected :9000, got %s", srv.address)
	}
}

func TestUpgrade_NoHandler(t *testing.T) {
	srv := NewServer()
	req := httptest.NewRequest("GET", "/unknown", nil)
	w := httptest.NewRecorder()
	srv.handleUpgrade(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpgrade_NotWebSocket(t *testing.T) {
	srv := NewServer()
	srv.OnMessage("/chat", func(ctx context.Context, msg []byte) ([]byte, error) {
		return nil, nil
	})
	req := httptest.NewRequest("GET", "/chat", nil)
	w := httptest.NewRecorder()
	srv.handleUpgrade(w, req)
	if w.Code == http.StatusOK {
		t.Error("non-upgrade should not return 200")
	}
}

func TestEndpoint_BeforeStart(t *testing.T) {
	srv := NewServer()
	_, err := srv.Endpoint()
	if err == nil {
		t.Error("expected error before start")
	}
}
