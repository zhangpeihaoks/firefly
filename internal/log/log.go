// Package log provides structured logging for the Firefly framework.
// It uses Go's standard library slog as the core and integrates lumberjack for log rotation.
package log

import (
	"context"
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config is the log configuration.
type Config struct {
	// FileName is the log file name.
	// If empty, logs will only output to console.
	FileName string
	// MaxSize is the maximum file size in megabytes before rotation.
	MaxSize int
	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int
	// MaxAge is the maximum number of days to retain old log files.
	MaxAge int
	// Level is the log level: debug, info, warn, error.
	Level string
	// JSONFormat indicates whether to use JSON format for logs.
	JSONFormat bool
	// Location indicates whether to show source code location in logs.
	Location bool
	// RemoveTime indicates whether to remove the time field from logs.
	RemoveTime bool
	// Writer is a custom output writer.
	// If set, it will be used instead of file and console output.
	Writer io.Writer
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		FileName:   "",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Level:      "info",
		JSONFormat: true,
		Location:   true,
		RemoveTime: false,
	}
}

// logger is the global logger instance.
var logger *slog.Logger

// New creates a new logger with the given configuration.
// It returns a cleanup function that should be called to close the log file.
//
// The logger supports:
//   - JSON and Text formats
//   - Log levels: Debug, Info, Warn, Error
//   - Log rotation via lumberjack
//   - Output to file, console, or both
//   - Source code location display
//
// Example:
//
//	cleanup := log.New(&log.Config{
//	    FileName:   "app.log",
//	    MaxSize:    100,
//	    MaxBackups: 3,
//	    MaxAge:     7,
//	    Level:      "info",
//	    JSONFormat: true,
//	    Location:   true,
//	})
//	defer cleanup()
func New(c *Config) (cleanup func()) {
	if c == nil {
		c = DefaultConfig()
	}

	// Apply defaults for zero values
	if c.MaxSize <= 0 {
		c.MaxSize = 100
	}
	if c.MaxBackups <= 0 {
		c.MaxBackups = 3
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 7
	}
	if c.Level == "" {
		c.Level = "info"
	}

	// Parse log level
	level := parseLevel(c.Level)

	// Create writer
	var writers []io.Writer
	var lumberjackWriter *lumberjack.Logger

	// Use custom writer if provided
	if c.Writer != nil {
		writers = append(writers, c.Writer)
	} else {
		// Add file writer if FileName is specified
		if c.FileName != "" {
			lumberjackWriter = &lumberjack.Logger{
				Filename:   c.FileName,
				MaxSize:    c.MaxSize,
				MaxBackups: c.MaxBackups,
				MaxAge:     c.MaxAge,
				Compress:   true,
			}
			writers = append(writers, lumberjackWriter)
		}
		// Always add console output
		writers = append(writers, os.Stdout)
	}

	// Create multi-writer
	var writer io.Writer
	if len(writers) == 1 {
		writer = writers[0]
	} else {
		writer = io.MultiWriter(writers...)
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}
	if c.Location {
		opts.AddSource = true
	}

	// Remove time field if configured
	if c.RemoveTime {
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		}
	}

	// Create handler
	var handler slog.Handler
	if c.JSONFormat {
		handler = slog.NewJSONHandler(writer, opts)
		// Wrap with context handler for automatic request_id propagation
		handler = NewContextHandler(handler)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	// Create logger
	logger = slog.New(handler)
	slog.SetDefault(logger)

	// Return cleanup function
	return func() {
		if lumberjackWriter != nil {
			_ = lumberjackWriter.Close()
		}
	}
}

// parseLevel parses a string level to slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// L returns the global logger instance.
// If no logger has been initialized, it returns the default slog logger.
func L() *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}

// SetLogger sets the global logger instance.
func SetLogger(l *slog.Logger) {
	logger = l
	slog.SetDefault(l)
}

// Debug logs a message at Debug level.
func Debug(msg string, args ...any) {
	L().Debug(msg, args...)
}

// DebugCtx logs a message at Debug level with context.
func DebugCtx(ctx context.Context, msg string, args ...any) {
	L().DebugContext(ctx, msg, args...)
}

// Info logs a message at Info level.
func Info(msg string, args ...any) {
	L().Info(msg, args...)
}

// InfoCtx logs a message at Info level with context.
func InfoCtx(ctx context.Context, msg string, args ...any) {
	L().InfoContext(ctx, msg, args...)
}

// Warn logs a message at Warn level.
func Warn(msg string, args ...any) {
	L().Warn(msg, args...)
}

// WarnCtx logs a message at Warn level with context.
func WarnCtx(ctx context.Context, msg string, args ...any) {
	L().WarnContext(ctx, msg, args...)
}

// Error logs a message at Error level.
func Error(msg string, args ...any) {
	L().Error(msg, args...)
}

// ErrorCtx logs a message at Error level with context.
func ErrorCtx(ctx context.Context, msg string, args ...any) {
	L().ErrorContext(ctx, msg, args...)
}

// With returns a logger with the given key-value pairs.
func With(args ...any) *slog.Logger {
	return L().With(args...)
}

// WithGroup returns a logger with the given group.
func WithGroup(name string) *slog.Logger {
	return L().WithGroup(name)
}
