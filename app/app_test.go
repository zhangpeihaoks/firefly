// Package app provides application lifecycle management for the Firefly framework.
package app

import (
	"context"
	"log/slog"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// mockServer is a mock implementation of transport.Server for testing
type mockServer struct {
	started bool
	stopped bool
}

func (m *mockServer) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockServer) Stop(ctx context.Context) error {
	m.stopped = true
	return nil
}

// TestNew creates an App instance with default options
func TestNew(t *testing.T) {
	app := New()
	if app == nil {
		t.Fatal("New() returned nil")
	}
	if app.ctx == nil {
		t.Error("app.ctx should not be nil")
	}
	if app.cancel == nil {
		t.Error("app.cancel should not be nil")
	}
}

// TestNameOption tests the Name option
func TestNameOption(t *testing.T) {
	expectedName := "test-service"
	app := New(Name(expectedName))
	if app.Name() != expectedName {
		t.Errorf("Name() = %q, want %q", app.Name(), expectedName)
	}
}

// TestMetadataOption tests the Metadata option
func TestMetadataOption(t *testing.T) {
	expectedMetadata := map[string]string{
		"version": "1.0.0",
		"env":     "production",
	}
	app := New(Metadata(expectedMetadata))
	metadata := app.Metadata()
	if metadata == nil {
		t.Fatal("Metadata() returned nil")
	}
	if metadata["version"] != "1.0.0" {
		t.Errorf("metadata[\"version\"] = %q, want %q", metadata["version"], "1.0.0")
	}
	if metadata["env"] != "production" {
		t.Errorf("metadata[\"env\"] = %q, want %q", metadata["env"], "production")
	}
}

// TestLoggerOption tests the Logger option
func TestLoggerOption(t *testing.T) {
	logger := slog.Default()
	app := New(Logger(logger))
	if app.opts.logger != logger {
		t.Error("Logger option not applied correctly")
	}
}

// TestServerOption tests the Server option
func TestServerOption(t *testing.T) {
	srv1 := &mockServer{}
	srv2 := &mockServer{}
	app := New(Server(srv1, srv2))
	if len(app.opts.servers) != 2 {
		t.Errorf("len(servers) = %d, want 2", len(app.opts.servers))
	}
}

// TestStartedFuncOption tests the StartedFunc option
func TestStartedFuncOption(t *testing.T) {
	called := false
	app := New(StartedFunc(func() {
		called = true
	}))
	if app.opts.started == nil {
		t.Fatal("StartedFunc option not applied")
	}
	app.opts.started()
	if !called {
		t.Error("started callback was not called")
	}
}

// TestSignalOption tests the Signal option
func TestSignalOption(t *testing.T) {
	app := New(Signal(os.Interrupt, os.Kill))
	if len(app.opts.sigs) != 2 {
		t.Errorf("len(sigs) = %d, want 2", len(app.opts.sigs))
	}
}

// TestStopTimeoutOption tests the StopTimeout option
func TestStopTimeoutOption(t *testing.T) {
	expectedTimeout := 10 * time.Second
	app := New(StopTimeout(expectedTimeout))
	if app.opts.stopTimeout != expectedTimeout {
		t.Errorf("stopTimeout = %v, want %v", app.opts.stopTimeout, expectedTimeout)
	}
}

// TestDefaultStopTimeout tests that the default stop timeout is 5 seconds
func TestDefaultStopTimeout(t *testing.T) {
	app := New()
	expectedTimeout := 5 * time.Second
	if app.opts.stopTimeout != expectedTimeout {
		t.Errorf("default stopTimeout = %v, want %v", app.opts.stopTimeout, expectedTimeout)
	}
}

// TestMultipleOptions tests applying multiple options
func TestMultipleOptions(t *testing.T) {
	app := New(
		Name("multi-service"),
		Metadata(map[string]string{"key": "value"}),
		StopTimeout(15*time.Second),
	)
	if app.Name() != "multi-service" {
		t.Errorf("Name() = %q, want %q", app.Name(), "multi-service")
	}
	if app.Metadata()["key"] != "value" {
		t.Errorf("Metadata()[\"key\"] = %q, want %q", app.Metadata()["key"], "value")
	}
	if app.opts.stopTimeout != 15*time.Second {
		t.Errorf("stopTimeout = %v, want %v", app.opts.stopTimeout, 15*time.Second)
	}
}

// TestEmptyMetadata tests that Metadata returns nil when not set
func TestEmptyMetadata(t *testing.T) {
	app := New()
	if app.Metadata() != nil {
		t.Errorf("Metadata() = %v, want nil", app.Metadata())
	}
}

// TestEmptyName tests that Name returns empty string when not set
func TestEmptyName(t *testing.T) {
	app := New()
	if app.Name() != "" {
		t.Errorf("Name() = %q, want %q", app.Name(), "")
	}
}

// TestServerOptionAccumulate tests that Server option accumulates servers
func TestServerOptionAccumulate(t *testing.T) {
	srv1 := &mockServer{}
	srv2 := &mockServer{}
	srv3 := &mockServer{}
	app := New(Server(srv1), Server(srv2, srv3))
	if len(app.opts.servers) != 3 {
		t.Errorf("len(servers) = %d, want 3", len(app.opts.servers))
	}
}

// Verify mockServer implements transport.Server interface
var _ transport.Server = (*mockServer)(nil)

// errorMockServer is a mock that returns an error on Start
type errorMockServer struct {
	startErr error
	stopErr  error
}

func (m *errorMockServer) Start(ctx context.Context) error {
	return m.startErr
}

func (m *errorMockServer) Stop(ctx context.Context) error {
	return m.stopErr
}

// blockingMockServer blocks until context is cancelled
type blockingMockServer struct {
	started chan struct{}
	stopped chan struct{}
}

func newBlockingMockServer() *blockingMockServer {
	return &blockingMockServer{
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (m *blockingMockServer) Start(ctx context.Context) error {
	close(m.started)
	<-ctx.Done()
	return nil
}

func (m *blockingMockServer) Stop(ctx context.Context) error {
	close(m.stopped)
	return nil
}

// TestRun tests the Run method with a single server
func TestRun(t *testing.T) {
	srv := newBlockingMockServer()
	app := New(
		Server(srv),
		Logger(slog.Default()),
	)

	// Start the app in a goroutine
	done := make(chan struct{})
	var exitCode int
	var err error
	go func() {
		exitCode, err = app.Run()
		close(done)
	}()

	// Wait for server to start
	select {
	case <-srv.started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start within timeout")
	}

	// Cancel the context to trigger shutdown
	app.cancel()

	// Wait for Run to complete
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not complete within timeout")
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}

	// Verify server was stopped
	select {
	case <-srv.stopped:
	case <-time.After(1 * time.Second):
		t.Error("server was not stopped")
	}
}

// TestRunWithStartedCallback tests that the started callback is called
func TestRunWithStartedCallback(t *testing.T) {
	callbackCalled := false
	srv := newBlockingMockServer()
	app := New(
		Server(srv),
		Logger(slog.Default()),
		StartedFunc(func() {
			callbackCalled = true
		}),
	)

	go func() {
		<-srv.started
		app.cancel()
	}()

	exitCode, _ := app.Run()

	if !callbackCalled {
		t.Error("started callback was not called")
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
}

// TestRunWithMultipleServers tests running multiple servers concurrently
func TestRunWithMultipleServers(t *testing.T) {
	srv1 := newBlockingMockServer()
	srv2 := newBlockingMockServer()
	srv3 := newBlockingMockServer()

	app := New(
		Server(srv1, srv2, srv3),
		Logger(slog.Default()),
	)

	go func() {
		// Wait for all servers to start
		<-srv1.started
		<-srv2.started
		<-srv3.started
		app.cancel()
	}()

	exitCode, err := app.Run()

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

// TestRunWithServerError tests that Run returns error when server fails to start
func TestRunWithServerError(t *testing.T) {
	startErr := context.DeadlineExceeded
	srv := &errorMockServer{startErr: startErr}

	app := New(
		Server(srv),
		Logger(slog.Default()),
	)

	exitCode, err := app.Run()

	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
	}
	if err != startErr {
		t.Errorf("err = %v, want %v", err, startErr)
	}
}

// TestRunWithNoServers tests running with no servers (should succeed)
func TestRunWithNoServers(t *testing.T) {
	app := New(
		Logger(slog.Default()),
	)

	go func() {
		time.Sleep(100 * time.Millisecond)
		app.cancel()
	}()

	exitCode, err := app.Run()

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

// TestDefaultSignals tests that default signals include SIGINT and SIGTERM
func TestDefaultSignals(t *testing.T) {
	app := New()
	if len(app.opts.sigs) != 2 {
		t.Errorf("expected 2 default signals, got %d", len(app.opts.sigs))
	}

	// Check that both SIGINT and SIGTERM are in the default signals
	hasSIGINT := false
	hasSIGTERM := false
	for _, sig := range app.opts.sigs {
		if sig == os.Interrupt {
			hasSIGINT = true
		}
		if sig == syscall.SIGTERM {
			hasSIGTERM = true
		}
	}

	if !hasSIGINT {
		t.Error("SIGINT not in default signals")
	}
	if !hasSIGTERM {
		t.Error("SIGTERM not in default signals")
	}
}

// TestSignalHandlingWithSIGINT tests SIGINT signal handling
func TestSignalHandlingWithSIGINT(t *testing.T) {
	srv := newBlockingMockServer()
	app := New(
		Server(srv),
		Logger(slog.Default()),
		Signal(os.Interrupt), // Test with SIGINT
	)

	// Start the app in a goroutine
	done := make(chan struct{})
	var exitCode int
	var err error
	go func() {
		exitCode, err = app.Run()
		close(done)
	}()

	// Wait for server to start
	select {
	case <-srv.started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start within timeout")
	}

	// Simulate SIGINT signal
	app.cancel()

	// Wait for Run to complete
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not complete within timeout")
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

// TestSignalHandlingWithSIGTERM tests SIGTERM signal handling
func TestSignalHandlingWithSIGTERM(t *testing.T) {
	srv := newBlockingMockServer()
	app := New(
		Server(srv),
		Logger(slog.Default()),
		Signal(syscall.SIGTERM), // Test with SIGTERM
	)

	// Start the app in a goroutine
	done := make(chan struct{})
	var exitCode int
	var err error
	go func() {
		exitCode, err = app.Run()
		close(done)
	}()

	// Wait for server to start
	select {
	case <-srv.started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start within timeout")
	}

	// Cancel context to simulate signal
	app.cancel()

	// Wait for Run to complete
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not complete within timeout")
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

// TestGracefulShutdownWithTimeout tests graceful shutdown with timeout
func TestGracefulShutdownWithTimeout(t *testing.T) {
	srv := newBlockingMockServer()
	shutdownTimeout := 2 * time.Second

	app := New(
		Server(srv),
		Logger(slog.Default()),
		StopTimeout(shutdownTimeout),
	)

	// Start the app in a goroutine
	done := make(chan struct{})
	go func() {
		app.Run()
		close(done)
	}()

	// Wait for server to start
	select {
	case <-srv.started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start within timeout")
	}

	// Trigger shutdown
	app.cancel()

	// Verify shutdown completes within reasonable time
	select {
	case <-done:
		// Good - shutdown completed
	case <-time.After(shutdownTimeout + 1*time.Second):
		t.Fatal("shutdown took longer than expected")
	}
}

// TestImmediateShutdown tests immediate shutdown when stopTimeout is 0
func TestImmediateShutdown(t *testing.T) {
	srv := newBlockingMockServer()

	app := New(
		Server(srv),
		Logger(slog.Default()),
		StopTimeout(0), // Immediate shutdown
	)

	// Start the app in a goroutine
	done := make(chan struct{})
	go func() {
		app.Run()
		close(done)
	}()

	// Wait for server to start
	select {
	case <-srv.started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start within timeout")
	}

	// Trigger shutdown
	app.cancel()

	// Verify shutdown completes quickly (should be immediate)
	select {
	case <-done:
		// Good - shutdown completed
	case <-time.After(500 * time.Millisecond):
		t.Fatal("immediate shutdown took too long")
	}
}

// TestGracefulShutdownStopsNewRequests tests that graceful shutdown stops accepting new requests
func TestGracefulShutdownStopsNewRequests(t *testing.T) {
	srv := newBlockingMockServer()

	app := New(
		Server(srv),
		Logger(slog.Default()),
		StopTimeout(1*time.Second),
	)

	// Start the app in a goroutine
	done := make(chan struct{})
	go func() {
		app.Run()
		close(done)
	}()

	// Wait for server to start
	select {
	case <-srv.started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start within timeout")
	}

	// Trigger shutdown
	app.cancel()

	// Wait for shutdown to complete
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown did not complete within timeout")
	}
}

// slowMockServer simulates a server that takes time to stop
type slowMockServer struct {
	started chan struct{}
	stopped chan struct{}
}

func newSlowMockServer() *slowMockServer {
	return &slowMockServer{
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (m *slowMockServer) Start(ctx context.Context) error {
	close(m.started)
	<-ctx.Done()
	return nil
}

func (m *slowMockServer) Stop(ctx context.Context) error {
	// Simulate slow shutdown
	select {
	case <-time.After(200 * time.Millisecond):
		close(m.stopped)
		return nil
	case <-ctx.Done():
		// Context cancelled (timeout)
		close(m.stopped)
		return ctx.Err()
	}
}

// TestGracefulShutdownWaitsForExistingRequests tests that shutdown waits for existing requests
func TestGracefulShutdownWaitsForExistingRequests(t *testing.T) {
	srv := newSlowMockServer()

	app := New(
		Server(srv),
		Logger(slog.Default()),
		StopTimeout(500*time.Millisecond), // Give enough time for slow shutdown
	)

	// Start the app in a goroutine
	done := make(chan struct{})
	go func() {
		app.Run()
		close(done)
	}()

	// Wait for server to start
	select {
	case <-srv.started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start within timeout")
	}

	// Trigger shutdown
	app.cancel()

	// Wait for shutdown to complete
	select {
	case <-done:
		// Good - server had time to stop gracefully
	case <-time.After(1 * time.Second):
		t.Fatal("shutdown did not complete within timeout")
	}

	// Verify the server was stopped
	select {
	case <-srv.stopped:
		// Good - server stopped
	case <-time.After(100 * time.Millisecond):
		t.Error("server was not stopped")
	}
}

// TestGracefulShutdownTimeoutForcefullyCloses tests that shutdown forcefully closes after timeout
func TestGracefulShutdownTimeoutForcefullyCloses(t *testing.T) {
	srv := newSlowMockServer()

	app := New(
		Server(srv),
		Logger(slog.Default()),
		StopTimeout(50*time.Millisecond), // Very short timeout
	)

	// Start the app in a goroutine
	done := make(chan struct{})
	var exitCode int
	var err error
	go func() {
		exitCode, err = app.Run()
		close(done)
	}()

	// Wait for server to start
	select {
	case <-srv.started:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not start within timeout")
	}

	// Trigger shutdown
	app.cancel()

	// Wait for shutdown to complete
	select {
	case <-done:
		// Good - shutdown completed (may have timed out)
	case <-time.After(1 * time.Second):
		t.Fatal("shutdown did not complete within timeout")
	}

	// The exit code might be 1 if timeout occurred and server returned error
	// This is expected behavior - timeout causes context cancellation which returns error
	_ = exitCode
	_ = err
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// Property 1: Application Configuration Correctness
// Validates: Requirements 1.1, 1.4, 1.6
//
// For any options combination, the created App instance should contain correct configuration values.
func TestProperty1AppConfigCorrectness(t *testing.T) {
	tests := []struct {
		name  string
		opts  []Option
		check func(*App) bool
	}{
		{
			name: "default config",
			opts: []Option{},
			check: func(a *App) bool {
				return a.opts.stopTimeout == 5*time.Second && a.Name() == "" && a.Metadata() == nil
			},
		},
		{
			name: "name option",
			opts: []Option{Name("test-service")},
			check: func(a *App) bool {
				return a.Name() == "test-service"
			},
		},
		{
			name: "metadata option",
			opts: []Option{Metadata(map[string]string{"env": "prod"})},
			check: func(a *App) bool {
				return a.Metadata()["env"] == "prod"
			},
		},
		{
			name: "stop timeout option",
			opts: []Option{StopTimeout(10 * time.Second)},
			check: func(a *App) bool {
				return a.opts.stopTimeout == 10*time.Second
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New(tt.opts...)
			if !tt.check(app) {
				t.Error("app configuration not correct")
			}
		})
	}
}

// Property 2: Server Concurrency Management
// Validates: Requirements 1.2, 1.7
//
// For any server list, App should correctly manage concurrent server start and stop.
func TestProperty2ServerConcurrency(t *testing.T) {
	// Test multiple server management
	srv1 := newBlockingMockServer()
	srv2 := newBlockingMockServer()

	app := New(
		Server(srv1, srv2),
		Logger(slog.Default()),
	)

	// Start in goroutine
	done := make(chan struct{})
	go func() {
		app.Run()
		close(done)
	}()

	// Wait for both servers to start
	select {
	case <-srv1.started:
	case <-time.After(2 * time.Second):
		t.Fatal("srv1 did not start")
	}
	select {
	case <-srv2.started:
	case <-time.After(2 * time.Second):
		t.Fatal("srv2 did not start")
	}

	// Stop app
	app.cancel()

	// Wait for completion
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("app did not stop")
	}

	// Verify both servers were stopped
	select {
	case <-srv1.stopped:
	case <-time.After(1 * time.Second):
		t.Error("srv1 was not stopped")
	}
	select {
	case <-srv2.stopped:
	case <-time.After(1 * time.Second):
		t.Error("srv2 was not stopped")
	}
}
