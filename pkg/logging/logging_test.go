package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
)

func TestLogger(t *testing.T) {
	// Ensure default logger is not nil
	logger := Logger()
	if logger == nil {
		t.Error("Logger() returned nil")
	}
}

func TestSetLogger(t *testing.T) {
	// Save original logger
	original := Logger()
	defer SetLogger(original)

	// Create and set a new logger
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	newLogger := slog.New(handler)

	SetLogger(newLogger)

	if Logger() != newLogger {
		t.Error("SetLogger did not update the default logger")
	}

	// Log something and verify it went to our buffer
	Logger().Info("test message")
	if buf.Len() == 0 {
		t.Error("Expected log output in buffer")
	}
}

func TestAttributeHelpers(t *testing.T) {
	tests := []struct {
		name     string
		attr     slog.Attr
		wantKey  string
		wantVal  interface{}
	}{
		{"Bucket", Bucket("my-bucket"), "bucket", "my-bucket"},
		{"Object", Object("path/to/file"), "object", "path/to/file"},
		{"Path", Path("/local/path"), "path", "/local/path"},
		{"Operation", Operation("upload"), "op", "upload"},
		{"JobID", JobID("job-123"), "job_id", "job-123"},
		{"DurationMs", DurationMs(150), "duration_ms", int64(150)},
		{"Size", Size(1024), "size_bytes", int64(1024)},
		{"Status", Status(200), "status", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.wantKey {
				t.Errorf("got key %q, want %q", tt.attr.Key, tt.wantKey)
			}
			// For Value comparison, check the underlying value
			got := tt.attr.Value.Any()
			switch want := tt.wantVal.(type) {
			case int64:
				if gotInt, ok := got.(int64); !ok || gotInt != want {
					t.Errorf("got value %v, want %v", got, want)
				}
			case int:
				if gotInt, ok := got.(int64); !ok || gotInt != int64(want) {
					t.Errorf("got value %v, want %v", got, want)
				}
			case string:
				if gotStr, ok := got.(string); !ok || gotStr != want {
					t.Errorf("got value %v, want %v", got, want)
				}
			}
		})
	}
}

func TestErrAttribute(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		attr := Err(nil)
		// Empty attr for nil error
		if attr.Key != "" {
			t.Errorf("expected empty key for nil error, got %q", attr.Key)
		}
	})

	t.Run("non-nil error", func(t *testing.T) {
		err := errors.New("test error")
		attr := Err(err)
		if attr.Key != "error" {
			t.Errorf("got key %q, want %q", attr.Key, "error")
		}
	})
}

func TestWithContext(t *testing.T) {
	// WithContext currently just returns the default logger
	// This test ensures it doesn't panic and returns a valid logger
	logger := WithContext(nil)
	if logger == nil {
		t.Error("WithContext returned nil")
	}
}

func TestLoggerOutputFormat(t *testing.T) {
	// Create a logger that writes to a buffer
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, nil)
	logger := slog.New(handler)

	// Log a message with attributes
	logger.Info("test operation",
		Bucket("test-bucket"),
		Object("test-object"),
		Status(200),
	)

	// Parse the JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Verify expected fields
	if logEntry["msg"] != "test operation" {
		t.Errorf("got msg %q, want %q", logEntry["msg"], "test operation")
	}
	if logEntry["bucket"] != "test-bucket" {
		t.Errorf("got bucket %q, want %q", logEntry["bucket"], "test-bucket")
	}
	if logEntry["object"] != "test-object" {
		t.Errorf("got object %q, want %q", logEntry["object"], "test-object")
	}
	// Status is logged as a number in JSON
	if status, ok := logEntry["status"].(float64); !ok || status != 200 {
		t.Errorf("got status %v, want 200", logEntry["status"])
	}
}
