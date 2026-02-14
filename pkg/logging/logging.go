// Package logging provides structured logging using Go's slog package.
package logging

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// init initializes the default logger with JSON output for production.
func init() {
	once.Do(func() {
		handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		defaultLogger = slog.New(handler)
	})
}

// Logger returns the default logger.
func Logger() *slog.Logger {
	return defaultLogger
}

// SetLogger allows replacing the default logger (useful for testing).
func SetLogger(l *slog.Logger) {
	defaultLogger = l
}

// WithContext returns a logger that includes context values.
// Can be extended to extract request ID, trace ID, etc.
func WithContext(ctx context.Context) *slog.Logger {
	// Could extract values from context here (e.g., request ID)
	return defaultLogger
}

// Common attribute helpers for consistent logging

// Bucket creates a bucket name attribute.
func Bucket(name string) slog.Attr {
	return slog.String("bucket", name)
}

// Object creates an object/file name attribute.
func Object(name string) slog.Attr {
	return slog.String("object", name)
}

// Path creates a file path attribute.
func Path(path string) slog.Attr {
	return slog.String("path", path)
}

// Operation creates an operation name attribute.
func Operation(op string) slog.Attr {
	return slog.String("op", op)
}

// JobID creates a job ID attribute.
func JobID(id string) slog.Attr {
	return slog.String("job_id", id)
}

// Err creates an error attribute.
func Err(err error) slog.Attr {
	if err == nil {
		return slog.Attr{}
	}
	return slog.Any("error", err)
}

// Duration creates a duration attribute in milliseconds.
func DurationMs(ms int64) slog.Attr {
	return slog.Int64("duration_ms", ms)
}

// Size creates a size attribute in bytes.
func Size(bytes int64) slog.Attr {
	return slog.Int64("size_bytes", bytes)
}

// Status creates an HTTP status code attribute.
func Status(code int) slog.Attr {
	return slog.Int("status", code)
}
