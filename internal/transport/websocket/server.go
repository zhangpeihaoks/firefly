// Package websocket provides WebSocket transport for the Firefly framework.
// It implements transport.Server and transport.Endpointer, integrating with
// the existing middleware chain for authentication, logging, etc.
//
// Usage:
//
//	ws := websocket.NewServer(
//	    websocket.Address(":8081"),
//	    websocket.Middleware(middleware.Logging()),
//	)
//	ws.OnMessage("/chat", func(ctx context.Context, msg []byte) ([]byte, error) {
//	    return []byte("echo: " + string(msg)), nil
//	})
package websocket

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// Prometheus metrics for WebSocket
var (
	wsConnectionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "firefly_websocket_connections_active",
		Help: "Number of active WebSocket connections",
	})
	wsMessagesReceived = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "firefly_websocket_messages_received_total",
		Help: "Total number of WebSocket messages received",
	})
	wsMessagesSent = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "firefly_websocket_messages_sent_total",
		Help: "Total number of WebSocket messages sent",
	})
)

func init() {
	prometheus.MustRegister(wsConnectionsActive, wsMessagesReceived, wsMessagesSent)
}

// MessageHandler handles a WebSocket message and returns a response.
type MessageHandler func(ctx context.Context, msg []byte) ([]byte, error)

// messageEntry holds a message handler and its per-message middleware.
type messageEntry struct {
	handler    MessageHandler
	middleware []middleware.Middleware
}

type startStatus int32

const (
	statusNotStarted startStatus = iota
	statusStarting
	statusStarted
	statusFailed
)

var ErrServerFailed = errors.New("websocket server previously failed to start")

// Server is the WebSocket transport server.
type Server struct {
	httpServer   *http.Server
	lis          net.Listener
	once         sync.Once
	endpoint     *url.URL
	err          error
	startStatus  atomic.Int32
	network      string
	address      string
	timeout      time.Duration
	ms           []middleware.Middleware
	handlers     map[string]messageEntry
	mu           sync.RWMutex
	log          *slog.Logger
	tlsConf      *tls.Config
	upgrader     websocket.Upgrader
	activeConns  sync.WaitGroup
}

// ServerOption configures the WebSocket server.
type ServerOption func(*Server)

// NewServer creates a new WebSocket server.
func NewServer(opts ...ServerOption) *Server {
	s := &Server{
		network:  "tcp",
		address:  ":8081",
		timeout:  30 * time.Second,
		handlers: make(map[string]messageEntry),
		log:      log.L(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Address sets the listen address.
func Address(addr string) ServerOption {
	return func(s *Server) { s.address = addr }
}

// Timeout sets the server timeout.
func Timeout(d time.Duration) ServerOption {
	return func(s *Server) { s.timeout = d }
}

// Logger sets the logger.
func Logger(logger *slog.Logger) ServerOption {
	return func(s *Server) { s.log = logger }
}

// Middleware sets the middleware chain applied to upgrade requests.
func Middleware(ms ...middleware.Middleware) ServerOption {
	return func(s *Server) { s.ms = ms }
}

// TLS sets the TLS configuration.
func TLS(conf *tls.Config) ServerOption {
	return func(s *Server) { s.tlsConf = conf }
}

// CheckOrigin sets the origin check function for the upgrader.
func CheckOrigin(fn func(r *http.Request) bool) ServerOption {
	return func(s *Server) { s.upgrader.CheckOrigin = fn }
}

// OnMessage registers a message handler for a URL path.
// Optional per-message middleware is applied to each message before the handler.
func (s *Server) OnMessage(path string, handler MessageHandler, ms ...middleware.Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[path] = messageEntry{handler: handler, middleware: ms}
}

// Start starts the WebSocket server.
func (s *Server) Start(ctx context.Context) error {
	if !s.startStatus.CompareAndSwap(int32(statusNotStarted), int32(statusStarting)) {
		return errors.New("websocket server already started or failed")
	}

	// Build HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleUpgrade)

	s.httpServer = &http.Server{
		Addr:         s.address,
		Handler:      mux,
		ReadTimeout:  s.timeout,
		WriteTimeout: s.timeout,
		TLSConfig:    s.tlsConf,
	}

	lis, err := net.Listen(s.network, s.address)
	if err != nil {
		s.err = err
		s.startStatus.Store(int32(statusFailed))
		return err
	}
	s.lis = lis

	s.endpoint = &url.URL{Scheme: "ws", Host: lis.Addr().String()}
	if s.tlsConf != nil {
		s.endpoint.Scheme = "wss"
	}

	s.log.Info("websocket server starting", "address", s.address)

	go func() {
		var serveErr error
		if s.tlsConf != nil {
			serveErr = s.httpServer.ServeTLS(lis, "", "")
		} else {
			serveErr = s.httpServer.Serve(lis)
		}
		if serveErr != nil && serveErr != http.ErrServerClosed {
			s.log.Error("websocket server error", "error", serveErr)
		}
	}()

	s.startStatus.Store(int32(statusStarted))
	return nil
}

// Stop stops the WebSocket server gracefully.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("websocket server stopping")

	done := make(chan struct{})
	go func() {
		s.httpServer.Shutdown(ctx)
		s.activeConns.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("websocket server stopped")
		return nil
	case <-ctx.Done():
		s.httpServer.Close()
		return ctx.Err()
	}
}

// Endpoint returns the server endpoint URL.
func (s *Server) Endpoint() (*url.URL, error) {
	if s.endpoint == nil {
		return nil, fmt.Errorf("server not started")
	}
	return s.endpoint, nil
}

// handleUpgrade handles HTTP-to-WebSocket upgrade requests.
func (s *Server) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	entry, exists := s.handlers[r.URL.Path]
	s.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	// Build and apply middleware chain
	chain := middleware.Chain(s.ms...)
	next := chain(func(ctx context.Context, req any) (any, error) {
		// Upgrade to WebSocket
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.log.Error("websocket upgrade failed", "error", err)
			return nil, err
		}
		defer conn.Close()

		s.activeConns.Add(1)
		defer s.activeConns.Done()

		// Inject WebSocket transport context
		ctx = newWSContext(ctx, r.URL.Path)

		// Build per-message middleware chain
		msgChain := middleware.Chain(entry.middleware...)

		// Message loop
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					s.log.Debug("websocket connection closed", "error", err)
				}
				break
			}

			// Apply per-message middleware then call handler
			h := msgChain(func(c context.Context, _ any) (any, error) {
				return entry.handler(c, msg)
			})
			resp, err := h(ctx, msg)
			if err != nil {
				s.log.Error("websocket handler error", "error", err)
				conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
				break
			}

			if resp != nil {
				if writeErr := conn.WriteMessage(websocket.TextMessage, resp.([]byte)); writeErr != nil {
					s.log.Debug("websocket write error", "error", writeErr)
					break
				}
			}
		}
		return nil, nil
	})

	ctx := r.Context()
	next(ctx, r)
}

// wsTransporter implements transport.Transporter for WebSocket connections.
type wsTransporter struct {
	path string
}

func (t *wsTransporter) Kind() transport.Kind             { return transport.KindHTTP }
func (t *wsTransporter) Endpoint() string                 { return t.path }
func (t *wsTransporter) Operation() string                { return t.path }
func (t *wsTransporter) RequestHeader() transport.Header  { return &wsHeader{} }
func (t *wsTransporter) ReplyHeader() transport.Header    { return &wsHeader{} }
func (t *wsTransporter) PathParams() map[string]string    { return nil }
func (t *wsTransporter) QueryParams() map[string][]string { return nil }

// wsHeader is a minimal no-op header implementation for WebSocket transport.
type wsHeader struct{}

func (h *wsHeader) Get(string) string  { return "" }
func (h *wsHeader) Set(string, string) {}
func (h *wsHeader) Keys() []string     { return nil }

// newWSContext creates a context with WebSocket transport info injected.
func newWSContext(ctx context.Context, path string) context.Context {
	return transport.NewContext(ctx, &wsTransporter{path: path})
}
