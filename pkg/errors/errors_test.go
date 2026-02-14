package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestAppError(t *testing.T) {
	t.Run("Error() returns message", func(t *testing.T) {
		appErr := &AppError{
			Err:        errors.New("internal error"),
			Message:    "user-safe message",
			StatusCode: 500,
		}

		if appErr.Error() != "user-safe message" {
			t.Errorf("got %q, want %q", appErr.Error(), "user-safe message")
		}
	})

	t.Run("Unwrap() returns internal error", func(t *testing.T) {
		internalErr := errors.New("internal error")
		appErr := &AppError{
			Err:        internalErr,
			Message:    "user-safe message",
			StatusCode: 500,
		}

		if appErr.Unwrap() != internalErr {
			t.Errorf("Unwrap() did not return the internal error")
		}
	})
}

func TestNew(t *testing.T) {
	internalErr := errors.New("internal error")
	appErr := New(internalErr, "user message", 400)

	if appErr.Err != internalErr {
		t.Error("Err field not set correctly")
	}
	if appErr.Message != "user message" {
		t.Error("Message field not set correctly")
	}
	if appErr.StatusCode != 400 {
		t.Error("StatusCode field not set correctly")
	}
}

func TestWrap(t *testing.T) {
	t.Run("wraps non-nil error", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := Wrap(original, "context")

		if wrapped == nil {
			t.Fatal("Wrap returned nil")
		}

		expected := "context: original error"
		if wrapped.Error() != expected {
			t.Errorf("got %q, want %q", wrapped.Error(), expected)
		}

		// Verify unwrapping works
		if !errors.Is(wrapped, original) {
			t.Error("errors.Is failed to match original error")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		if Wrap(nil, "context") != nil {
			t.Error("Wrap should return nil for nil error")
		}
	})
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, ""},
		{"AppError", &AppError{Err: errors.New("internal"), Message: "safe message", StatusCode: 400}, "safe message"},
		{"ErrNotFound", ErrNotFound, "Resource not found"},
		{"ErrUnauthorized", ErrUnauthorized, "Unauthorized"},
		{"ErrBadRequest", ErrBadRequest, "Invalid request"},
		{"ErrBucketNotFound", ErrBucketNotFound, "Bucket not found"},
		{"ErrObjectNotFound", ErrObjectNotFound, "Object not found"},
		{"ErrPathTraversal", ErrPathTraversal, "Invalid path"},
		{"wrapped ErrNotFound", fmt.Errorf("context: %w", ErrNotFound), "Resource not found"},
		{"bucket not found pattern", errors.New("bucket 'test' not found"), "Bucket not found"},
		{"object not found pattern", errors.New("object not found in bucket"), "File not found"},
		{"file not found pattern", errors.New("file not found"), "File not found"},
		{"credential error", errors.New("invalid credentials"), "Authentication failed"},
		{"unauthorized pattern", errors.New("unauthorized access"), "Authentication failed"},
		{"authentication pattern", errors.New("authentication failed"), "Authentication failed"},
		{"client creation failure", errors.New("failed to create b2 client"), "Failed to connect to storage"},
		{"connection refused", errors.New("connection refused"), "Connection error"},
		{"no such host", errors.New("no such host"), "Connection error"},
		{"timeout", errors.New("operation timeout"), "Connection error"},
		{"permission denied", errors.New("permission denied"), "Access denied"},
		{"access denied", errors.New("access denied"), "Access denied"},
		{"generic error", errors.New("something unexpected"), "An error occurred"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sanitize(tt.err)
			if result != tt.expected {
				t.Errorf("Sanitize(%v) = %q, want %q", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"ErrNotFound", ErrNotFound, true},
		{"ErrBucketNotFound", ErrBucketNotFound, true},
		{"ErrObjectNotFound", ErrObjectNotFound, true},
		{"wrapped ErrNotFound", fmt.Errorf("context: %w", ErrNotFound), true},
		{"not found in message", errors.New("resource not found"), true},
		{"no such in message", errors.New("no such file"), true},
		{"unrelated error", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"ErrUnauthorized", ErrUnauthorized, true},
		{"wrapped ErrUnauthorized", fmt.Errorf("context: %w", ErrUnauthorized), true},
		{"unauthorized in message", errors.New("unauthorized access"), true},
		{"forbidden in message", errors.New("access forbidden"), true},
		{"credential in message", errors.New("bad credentials"), true},
		{"authentication in message", errors.New("authentication failed"), true},
		{"unrelated error", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUnauthorized(tt.err)
			if result != tt.expected {
				t.Errorf("IsUnauthorized(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestContainsAll(t *testing.T) {
	tests := []struct {
		s        string
		substrs  []string
		expected bool
	}{
		{"bucket not found", []string{"bucket", "not found"}, true},
		{"bucket not found", []string{"bucket", "missing"}, false},
		{"test string", []string{"test"}, true},
		{"test string", []string{}, true},
	}

	for _, tt := range tests {
		result := containsAll(tt.s, tt.substrs...)
		if result != tt.expected {
			t.Errorf("containsAll(%q, %v) = %v, want %v", tt.s, tt.substrs, result, tt.expected)
		}
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s        string
		substrs  []string
		expected bool
	}{
		{"bucket not found", []string{"found", "missing"}, true},
		{"bucket not found", []string{"error", "missing"}, false},
		{"test string", []string{"test"}, true},
		{"test string", []string{}, false},
	}

	for _, tt := range tests {
		result := containsAny(tt.s, tt.substrs...)
		if result != tt.expected {
			t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrs, result, tt.expected)
		}
	}
}
