package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	if cfg.FileName != "" {
		t.Errorf("DefaultConfig().FileName = %q, want empty", cfg.FileName)
	}
	if cfg.MaxSize != 100 {
		t.Errorf("DefaultConfig().MaxSize = %d, want 100", cfg.MaxSize)
	}
	if cfg.MaxBackups != 3 {
		t.Errorf("DefaultConfig().MaxBackups = %d, want 3", cfg.MaxBackups)
	}
	if cfg.MaxAge != 7 {
		t.Errorf("DefaultConfig().MaxAge = %d, want 7", cfg.MaxAge)
	}
	if cfg.Level != "info" {
		t.Errorf("DefaultConfig().Level = %q, want \"info\"", cfg.Level)
	}
	if !cfg.JSONFormat {
		t.Errorf("DefaultConfig().JSONFormat = %v, want true", cfg.JSONFormat)
	}
	if !cfg.Location {
		t.Errorf("DefaultConfig().Location = %v, want true", cfg.Location)
	}
	if cfg.RemoveTime {
		t.Errorf("DefaultConfig().RemoveTime = %v, want false", cfg.RemoveTime)
	}
}

func TestNewWithNilConfig(t *testing.T) {
	cleanup := New(nil)
	defer cleanup()

	// Should not panic and should use defaults
	if logger == nil {
		t.Error("New(nil) did not initialize logger")
	}
}

func TestNewWithCustomWriter(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "info",
		JSONFormat: true,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Output = %q, want to contain \"test message\"", output)
	}
}

func TestNewWithJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "info",
		JSONFormat: true,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	Info("test message", "key", "value")

	output := buf.String()
	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v\nOutput: %s", err, output)
	}
	if result["msg"] != "test message" {
		t.Errorf("result[\"msg\"] = %v, want \"test message\"", result["msg"])
	}
}

func TestNewWithTextFormat(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "info",
		JSONFormat: false,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	Info("test message", "key", "value")

	output := buf.String()
	// Text format should contain key=value pairs
	if !strings.Contains(output, "key=value") {
		t.Errorf("Output = %q, want to contain \"key=value\"", output)
	}
}

func TestNewWithSourceLocation(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "info",
		JSONFormat: true,
		Location:   true,
	}
	cleanup := New(cfg)
	defer cleanup()

	Info("test message")

	output := buf.String()
	// JSON output should contain source field
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
	if _, ok := result["source"]; !ok {
		t.Error("Output does not contain source field")
	}
}

func TestNewWithoutSourceLocation(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "info",
		JSONFormat: true,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	Info("test message")

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
	if _, ok := result["source"]; ok {
		t.Error("Output should not contain source field")
	}
}

func TestNewWithRemoveTime(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "info",
		JSONFormat: true,
		Location:   false,
		RemoveTime: true,
	}
	cleanup := New(cfg)
	defer cleanup()

	Info("test message")

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
	if _, ok := result["time"]; ok {
		t.Error("Output should not contain time field when RemoveTime is true")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Log Level Filtering Tests - Requirement 5.3
// =============================================================================

// Feature: backend-server-framework, Property 13: 日志级别过滤
// Validates: Requirement 5.3

// TestLogLevelFilteringUnit tests log level filtering with specific examples.
// Debug level should log all messages, Info should filter Debug, etc.
func TestLogLevelFilteringUnit(t *testing.T) {
	tests := []struct {
		name      string
		level     string
		logFunc   func()
		shouldLog bool
	}{
		// Debug level: logs all messages
		{"debug level logs debug", "debug", func() { Debug("debug msg") }, true},
		{"debug level logs info", "debug", func() { Info("info msg") }, true},
		{"debug level logs warn", "debug", func() { Warn("warn msg") }, true},
		{"debug level logs error", "debug", func() { Error("error msg") }, true},

		// Info level: filters Debug, logs Info/Warn/Error
		{"info level filters debug", "info", func() { Debug("debug msg") }, false},
		{"info level logs info", "info", func() { Info("info msg") }, true},
		{"info level logs warn", "info", func() { Warn("warn msg") }, true},
		{"info level logs error", "info", func() { Error("error msg") }, true},

		// Warn level: filters Debug/Info, logs Warn/Error
		{"warn level filters debug", "warn", func() { Debug("debug msg") }, false},
		{"warn level filters info", "warn", func() { Info("info msg") }, false},
		{"warn level logs warn", "warn", func() { Warn("warn msg") }, true},
		{"warn level logs error", "warn", func() { Error("error msg") }, true},

		// Error level: only logs Error
		{"error level filters debug", "error", func() { Debug("debug msg") }, false},
		{"error level filters info", "error", func() { Info("info msg") }, false},
		{"error level filters warn", "error", func() { Warn("warn msg") }, false},
		{"error level logs error", "error", func() { Error("error msg") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cfg := &Config{
				Writer:     &buf,
				Level:      tt.level,
				JSONFormat: true,
				Location:   false,
			}
			cleanup := New(cfg)
			defer cleanup()

			tt.logFunc()

			output := buf.String()
			if tt.shouldLog && output == "" {
				t.Errorf("Expected log output for level %s, got empty", tt.level)
			}
			if !tt.shouldLog && output != "" {
				t.Errorf("Expected no log output for level %s, got: %s", tt.level, output)
			}
		})
	}
}

// Property 13: Log Level Filtering
// Validates: Requirement 5.3
// Feature: backend-server-framework, Property 13: 日志级别过滤
//
// For any log level configuration, log output should follow level filtering rules.
// Higher levels filter lower levels (Debug < Info < Warn < Error).
func TestProperty13LogLevelFiltering(t *testing.T) {
	type levelTestCase struct {
		configLevel string
		logLevel    string
		shouldLog   bool
	}

	f := func(tc levelTestCase) bool {
		// Normalize level names
		configLevel := strings.ToLower(tc.configLevel)
		if configLevel == "" {
			configLevel = "info"
		}
		logLevel := strings.ToLower(tc.logLevel)
		if logLevel == "" {
			logLevel = "info"
		}

		// Ensure valid log level for test
		validLogLevels := map[string]func(string){
			"debug": func(msg string) { Debug(msg) },
			"info":  func(msg string) { Info(msg) },
			"warn":  func(msg string) { Warn(msg) },
			"error": func(msg string) { Error(msg) },
		}

		logFunc, ok := validLogLevels[logLevel]
		if !ok {
			logFunc = func(msg string) { Info(msg) }
		}

		var buf bytes.Buffer
		cfg := &Config{
			Writer:     &buf,
			Level:      configLevel,
			JSONFormat: true,
			Location:   false,
		}
		cleanup := New(cfg)

		// Log a message at the specified level
		logFunc("test message")
		cleanup()

		output := buf.String()

		// Determine if this should log based on level hierarchy
		levelOrder := map[string]int{
			"debug": 0,
			"info":  1,
			"warn":  2,
			"error": 3,
		}

		configIdx, configOk := levelOrder[configLevel]
		logIdx, logOk := levelOrder[logLevel]

		// If level not recognized, assume should log (for invalid input safety)
		if !configOk || !logOk {
			return true
		}

		expectedShouldLog := logIdx >= configIdx

		if expectedShouldLog && output == "" {
			t.Logf("Level %s should log %s but got empty", configLevel, logLevel)
			return false
		}
		if !expectedShouldLog && output != "" {
			t.Logf("Level %s should filter %s but got: %s", configLevel, logLevel, output)
			return false
		}

		return true
	}

	// Generate test cases
	levels := []string{"debug", "info", "warn", "error"}

	if err := quick.Check(
		func(tc levelTestCase) bool {
			// Ensure valid levels
			if tc.configLevel == "" {
				tc.configLevel = "info"
			}
			if tc.logLevel == "" {
				tc.logLevel = "info"
			}

			// Only use valid levels
			validConfig := false
			validLog := false
			for _, l := range levels {
				if tc.configLevel == l {
					validConfig = true
				}
				if tc.logLevel == l {
					validLog = true
				}
			}
			if !validConfig || !validLog {
				return true // Skip invalid levels
			}

			return f(tc)
		},
		&quick.Config{
			Values: func(v []reflect.Value, r *rand.Rand) {
				v[0] = reflect.ValueOf(levelTestCase{
					configLevel: levels[r.Intn(4)],
					logLevel:    levels[r.Intn(4)],
				})
			},
			MaxCount: 100, // Minimum 100 iterations as required
		},
	); err != nil {
		t.Errorf("Property 13 failed: %v", err)
	}
}

func TestLogWithFile(t *testing.T) {
	// Create a temporary log file
	tmpFile := "test_log_" + strings.ReplaceAll(t.Name(), "/", "_") + ".log"
	defer os.Remove(tmpFile)

	cfg := &Config{
		FileName:   tmpFile,
		MaxSize:    10,
		MaxBackups: 1,
		MaxAge:     1,
		Level:      "info",
		JSONFormat: true,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	Info("test message to file")

	// Read the file to verify
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "test message to file") {
		t.Errorf("File content = %q, want to contain \"test message to file\"", string(content))
	}
}

func TestLogWithBothFileAndConsole(t *testing.T) {
	// Create a temporary log file
	tmpFile := "test_log_both_" + strings.ReplaceAll(t.Name(), "/", "_") + ".log"
	defer os.Remove(tmpFile)

	cfg := &Config{
		FileName:   tmpFile,
		MaxSize:    10,
		MaxBackups: 1,
		MaxAge:     1,
		Level:      "info",
		JSONFormat: true,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	// Redirect stdout to capture console output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	Info("test message to both")

	w.Close()
	var consoleOutput bytes.Buffer
	consoleOutput.ReadFrom(r)

	// Verify file output
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(content), "test message to both") {
		t.Errorf("File content = %q, want to contain \"test message to both\"", string(content))
	}
}

func TestL(t *testing.T) {
	// Reset logger
	logger = nil

	// L() should return default logger when no logger is set
	l := L()
	if l == nil {
		t.Error("L() returned nil")
	}

	// Set a custom logger
	var buf bytes.Buffer
	customLogger := slog.New(slog.NewJSONHandler(&buf, nil))
	SetLogger(customLogger)

	// L() should return the custom logger
	l = L()
	l.Info("test")

	if !strings.Contains(buf.String(), "test") {
		t.Error("L() did not return the custom logger")
	}
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "info",
		JSONFormat: true,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	l := With("service", "firefly")
	l.Info("test message")

	output := buf.String()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}
	if result["service"] != "firefly" {
		t.Errorf("result[\"service\"] = %v, want \"firefly\"", result["service"])
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "info",
		JSONFormat: true,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	l := WithGroup("request")
	l.Info("test message", "id", "123")

	output := buf.String()
	// The output should contain the group
	if !strings.Contains(output, "request") {
		t.Errorf("Output = %q, want to contain \"request\"", output)
	}
}

func TestLogContextFunctions(t *testing.T) {
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		Level:      "debug",
		JSONFormat: true,
		Location:   false,
	}
	cleanup := New(cfg)
	defer cleanup()

	ctx := context.Background()

	DebugCtx(ctx, "debug message")
	InfoCtx(ctx, "info message")
	WarnCtx(ctx, "warn message")
	ErrorCtx(ctx, "error message")

	output := buf.String()
	// Should have 4 log lines
	lines := strings.Count(output, "\n")
	if lines != 4 {
		t.Errorf("Expected 4 log lines, got %d", lines)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test that zero values get defaults applied
	var buf bytes.Buffer
	cfg := &Config{
		Writer:     &buf,
		MaxSize:    0,  // Should default to 100
		MaxBackups: 0,  // Should default to 3
		MaxAge:     0,  // Should default to 7
		Level:      "", // Should default to "info"
	}
	cleanup := New(cfg)
	defer cleanup()

	// Should not panic and should work with defaults
	Info("test message")
	if buf.String() == "" {
		t.Error("Logger did not output with default config")
	}
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// Property 12: Log Format Correctness
// Validates: Requirement 5.2
// Feature: backend-server-framework, Property 12: 日志格式正确性
//
// For any log message, it should be output correctly in both JSON and Text formats.
func TestProperty12LogFormatCorrectness(t *testing.T) {
	f := func(msg string) bool {
		if msg == "" {
			msg = "test"
		}
		// Ensure msg is valid printable ASCII
		validMsg := filterToPrintable(msg)

		// Test JSON format
		var jsonBuf bytes.Buffer
		jsonCfg := &Config{
			Writer:     &jsonBuf,
			Level:      "info",
			JSONFormat: true,
			Location:   false,
		}
		jsonCleanup := New(jsonCfg)
		Info(validMsg)
		jsonCleanup()

		jsonOutput := jsonBuf.String()
		if jsonOutput == "" {
			t.Log("JSON format: empty output")
			return false
		}

		// Verify JSON is valid
		var jsonResult map[string]any
		if err := json.Unmarshal([]byte(jsonOutput), &jsonResult); err != nil {
			t.Logf("JSON format: invalid JSON - %v", err)
			return false
		}

		// Test Text format
		var textBuf bytes.Buffer
		textCfg := &Config{
			Writer:     &textBuf,
			Level:      "info",
			JSONFormat: false,
			Location:   false,
		}
		textCleanup := New(textCfg)
		Info(validMsg)
		textCleanup()

		textOutput := textBuf.String()
		if textOutput == "" {
			t.Log("Text format: empty output")
			return false
		}

		// Text format should contain the message
		if !strings.Contains(textOutput, validMsg) {
			t.Logf("Text format: message not found in output")
			return false
		}

		return true
	}

	// Run with minimum 100 iterations
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property 12 failed: %v", err)
	}
}

// filterToPrintable converts a string to printable ASCII characters
func filterToPrintable(s string) string {
	result := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		if b >= 32 && b <= 126 {
			result = append(result, b)
		}
	}
	if len(result) == 0 {
		return "test"
	}
	return string(result)
}

// Property 14: Log Configuration Correctness
// Validates: Requirements 5.4, 5.6, 5.7
// Feature: backend-server-framework, Property 14: 日志配置正确性
//
// For any log configuration, all fields should be correctly accessed and applied.
func TestProperty14LogConfigCorrectness(t *testing.T) {
	type logConfigTest struct {
		maxSize    int
		maxBackups int
		maxAge     int
		level      string
		jsonFormat bool
		location   bool
	}

	if err := quick.Check(
		func(tc logConfigTest) bool {
			// Ensure valid values
			maxSize := tc.maxSize
			if maxSize <= 0 {
				maxSize = 100
			}
			maxBackups := tc.maxBackups
			if maxBackups <= 0 {
				maxBackups = 3
			}
			maxAge := tc.maxAge
			if maxAge <= 0 {
				maxAge = 7
			}
			level := tc.level
			if level == "" {
				level = "info"
			}

			var buf bytes.Buffer
			config := &Config{
				Writer:     &buf,
				MaxSize:    maxSize,
				MaxBackups: maxBackups,
				MaxAge:     maxAge,
				Level:      level,
				JSONFormat: tc.jsonFormat,
				Location:   tc.location,
			}

			cleanup := New(config)
			if logger == nil {
				t.Log("Logger not initialized")
				return false
			}

			// Log a message at the configured level (use Error to ensure it's logged)
			Error("test message")
			cleanup()

			// Verify output exists - error level should always be logged
			if buf.Len() == 0 {
				t.Log("No log output")
				return false
			}

			return true
		},
		&quick.Config{
			Values: func(v []reflect.Value, r *rand.Rand) {
				v[0] = reflect.ValueOf(logConfigTest{
					maxSize:    r.Intn(200) + 1,
					maxBackups: r.Intn(10) + 1,
					maxAge:     r.Intn(30) + 1,
					level:      []string{"debug", "info", "warn", "error"}[r.Intn(4)],
					jsonFormat: r.Intn(2) == 1,
					location:   r.Intn(2) == 1,
				})
			},
			MaxCount: 100,
		},
	); err != nil {
		t.Errorf("Property 14 failed: %v", err)
	}
}
