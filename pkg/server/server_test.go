package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	ferrors "github.com/zhangpeihaoks/firefly/pkg/errors"
	"github.com/zhangpeihaoks/firefly/pkg/middleware"
)

// startServer starts a server on an ephemeral loopback port and returns it
// with its base URL. The server is stopped when the test ends.
func startServer(t *testing.T, opts ...Option) (*Server, string) {
	t.Helper()

	base := []Option{
		Address("127.0.0.1:0"),
		Timeout(5 * time.Second),
		// An explicit chain keeps the default observability chain (metrics,
		// tracing) out of the way in unit tests.
		Middleware(middleware.Recovery()),
	}
	srv := New(append(base, opts...)...)
	if err := srv.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := srv.Stop(context.Background()); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	})

	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("Endpoint() error = %v", err)
	}
	return srv, "http://" + endpoint.Host
}

// TestRouteReturnsJSON verifies a registered route answers with the handler's
// return value as a 200 JSON response.
func TestRouteReturnsJSON(t *testing.T) {
	srv, base := startServer(t)
	srv.Route(http.MethodGet, "/healthz", func(_ context.Context, _ any) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	resp, err := http.Get(base + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body error = %v", err)
	}
	if !strings.Contains(string(body), `"status":"ok"`) {
		t.Fatalf("body = %s, want it to contain the handler value", body)
	}
}

// TestRouteErrorMapsToStatus verifies a returned *errors.Error becomes the
// HTTP status it carries, while a plain error is masked as 500.
func TestRouteErrorMapsToStatus(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  int
		wantBody  []string
		wantLeaks string
	}{
		{
			name:     "typed error keeps its code",
			err:      ferrors.New(ferrors.CodeServiceUnavailable, "NOT_READY", "webhook url missing"),
			wantCode: http.StatusServiceUnavailable,
			// The transport answers with the unified envelope
			// {"code":<status>,"msg":<message>,"data":null}; the reason is not
			// rendered, so the test pins the code and the message.
			wantBody: []string{`"code":503`, "webhook url missing"},
		},
		{
			name:      "plain error is masked",
			err:       errors.New("boom"),
			wantCode:  http.StatusInternalServerError,
			wantLeaks: "boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, base := startServer(t)
			srv.Route(http.MethodGet, "/x", func(_ context.Context, _ any) (any, error) {
				return nil, tt.err
			})

			resp, err := http.Get(base + "/x")
			if err != nil {
				t.Fatalf("GET /x error = %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantCode {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body error = %v", err)
			}
			for _, want := range tt.wantBody {
				if !strings.Contains(string(body), want) {
					t.Fatalf("body = %s, want it to contain %q", body, want)
				}
			}
			if tt.wantLeaks != "" && strings.Contains(string(body), tt.wantLeaks) {
				t.Fatalf("body = %s, plain error text must not leak", body)
			}
		})
	}
}

// TestMiddlewareOrder verifies global middleware runs before route
// middleware, in the order they were supplied.
func TestMiddlewareOrder(t *testing.T) {
	var order []string
	recorder := func(name string) middleware.Middleware {
		return func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (any, error) {
				order = append(order, name)
				return next(ctx, req)
			}
		}
	}

	srv, base := startServer(t, Middleware(recorder("global")))
	srv.Route(http.MethodGet, "/chain", func(_ context.Context, _ any) (any, error) {
		order = append(order, "handler")
		return "ok", nil
	}, recorder("route"))

	resp, err := http.Get(base + "/chain")
	if err != nil {
		t.Fatalf("GET /chain error = %v", err)
	}
	defer resp.Body.Close()

	want := []string{"global", "route", "handler"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
}

// TestGroupPrefix verifies routes registered on a group are served under the
// group prefix, including nested groups.
func TestGroupPrefix(t *testing.T) {
	srv, base := startServer(t)
	v1 := srv.Group("/v1")
	v1.Route(http.MethodGet, "/ping", func(_ context.Context, _ any) (any, error) {
		return "pong", nil
	})
	nested := v1.Group("/admin")
	nested.Route(http.MethodGet, "/ping", func(_ context.Context, _ any) (any, error) {
		return "nested", nil
	})

	for path, want := range map[string]string{"/v1/ping": `"pong"`, "/v1/admin/ping": `"nested"`} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s error = %v", path, err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			t.Fatalf("read body error = %v", readErr)
		}
		if !strings.Contains(string(body), want) {
			t.Fatalf("GET %s body = %s, want it to contain %s", path, body, want)
		}
	}
}

// TestHandlerReceivesHTTPRequest verifies the req argument is the original
// *http.Request, which is how a handler reaches the raw body and query.
func TestHandlerReceivesHTTPRequest(t *testing.T) {
	srv, base := startServer(t)
	srv.Route(http.MethodGet, "/inspect", func(_ context.Context, req any) (any, error) {
		r, ok := req.(*http.Request)
		if !ok {
			return nil, ferrors.New(ferrors.CodeInternal, "WRONG_TYPE", "req is not *http.Request")
		}
		return map[string]string{"mention": r.URL.Query().Get("mention")}, nil
	})

	resp, err := http.Get(base + "/inspect?mention=oncall")
	if err != nil {
		t.Fatalf("GET /inspect error = %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body error = %v", err)
	}
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "oncall") {
		t.Fatalf("status = %d body = %s, want 200 carrying the query value", resp.StatusCode, body)
	}
}

// TestStopClosesListener verifies the server stops accepting requests after
// Stop returns.
func TestStopClosesListener(t *testing.T) {
	srv, base := startServer(t)
	srv.Route(http.MethodGet, "/healthz", func(_ context.Context, _ any) (any, error) {
		return "ok", nil
	})

	if err := srv.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(base + "/healthz")
	if err == nil {
		resp.Body.Close()
		t.Fatal("request succeeded after Stop, want a connection error")
	}
}
