// Package log provides structured logging for the Firefly framework.
// It uses slog (Go 1.21+ standard library) with lumberjack for log rotation.
package log

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config is the logging configuration.
type Config struct {
	// FileName is the log file name (empty for stdout only)
	FileName string
	// MaxSize is the maximum file size in MB before rotation
	MaxSize int
	// MaxBackups is the maximum number of old log files to retain
	MaxBackups int
	// MaxAge is the maximum number of days to retain old log files
	MaxAge int
	// Level is the log level (debug, info, warn, error)
	Level string
	// JSONFormat enables JSON output format
	JSONFormat bool
	// Location enables source code location in logs
	Location bool
	// RemoveTime removes the time field from logs
	RemoveTime bool
	// Writer is a custom output writer (overrides FileName)
	Writer io.Writer
}

// New creates a new slog.Logger with the given configuration.
// It returns a cleanup function to close the log file.
func New(c *Config) (*slog.Logger, func()) {
	var writers []io.Writer
	var closers []func()

	// Add file writer if FileName is specified
	if c.FileName != "" {
		lj := &lumberjack.Logger{
			Filename:   c.FileName,
			MaxSize:    c.MaxSize,
			MaxBackups: c.MaxBackups,
			MaxAge:     c.MaxAge,
		}
		writers = append(writers, lj)
		closers = append(closers, func() { lj.Close() })
	}

	// Add custom writer or stdout
	if c.Writer != nil {
		writers = append(writers, c.Writer)
	} else {
		writers = append(writers, os.Stdout)
	}

	// Create multi-writer if needed
	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = io.MultiWriter(writers...)
	}

	// Parse log level
	level := parseLevel(c.Level)

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}
	if c.Location {
		opts.AddSource = true
	}

	// Create handler
	var handler slog.Handler
	if c.JSONFormat {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	// Create logger
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Return cleanup function
	cleanup := func() {
		for _, closer := range closers {
			closer()
		}
	}

	return logger, cleanup
}

// parseLevel parses a string log level to slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// DefaultConfig returns the default logging configuration.
func DefaultConfig() *Config {
	return &Config{
		FileName:   "",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Level:      "info",
		JSONFormat: true,
		Location:   true,
	}
}
