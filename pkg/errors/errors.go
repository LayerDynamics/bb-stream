// Package errors provides error handling utilities for bb-stream.
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for common conditions.
var (
	ErrNotFound       = errors.New("resource not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrBadRequest     = errors.New("invalid request")
	ErrInternalError  = errors.New("internal server error")
	ErrBucketNotFound = errors.New("bucket not found")
	ErrObjectNotFound = errors.New("object not found")
	ErrPathTraversal  = errors.New("path traversal not allowed")
)

// AppError wraps an error with a user-safe message.
// The internal error is logged but not exposed to clients.
type AppError struct {
	Err        error  // Internal error (logged, not exposed)
	Message    string // User-safe message
	StatusCode int    // HTTP status code
}

// Error implements the error interface.
func (e *AppError) Error() string {
	return e.Message
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates an AppError with the given internal error, user message, and status code.
func New(err error, message string, statusCode int) *AppError {
	return &AppError{
		Err:        err,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap wraps an error with additional context.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Sanitize returns a user-safe error message.
// This prevents leaking internal details like file paths, credentials, etc.
func Sanitize(err error) string {
	if err == nil {
		return ""
	}

	// Check if it's already an AppError with a safe message
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Message
	}

	// Check for sentinel errors
	switch {
	case errors.Is(err, ErrNotFound):
		return "Resource not found"
	case errors.Is(err, ErrUnauthorized):
		return "Unauthorized"
	case errors.Is(err, ErrBadRequest):
		return "Invalid request"
	case errors.Is(err, ErrBucketNotFound):
		return "Bucket not found"
	case errors.Is(err, ErrObjectNotFound):
		return "Object not found"
	case errors.Is(err, ErrPathTraversal):
		return "Invalid path"
	}

	// Map known error patterns to safe messages
	errStr := strings.ToLower(err.Error())

	// B2/bucket specific errors - check object/file first as they're more specific
	if containsAll(errStr, "object", "not found") || containsAll(errStr, "file", "not found") {
		return "File not found"
	}
	if containsAll(errStr, "bucket", "not found") {
		return "Bucket not found"
	}

	// Authentication errors
	if containsAny(errStr, "credential", "unauthorized", "authentication") {
		return "Authentication failed"
	}
	if containsAll(errStr, "failed to create", "client") {
		return "Failed to connect to storage"
	}

	// Network errors
	if containsAny(errStr, "connection refused", "no such host", "timeout") {
		return "Connection error"
	}

	// Permission errors
	if containsAny(errStr, "permission denied", "access denied") {
		return "Access denied"
	}

	// Generic fallback - never expose raw error
	return "An error occurred"
}

// containsAll checks if s contains all of the given substrings.
func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// containsAny checks if s contains any of the given substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// IsNotFound returns true if the error indicates a resource was not found.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrBucketNotFound) || errors.Is(err, ErrObjectNotFound) {
		return true
	}
	errStr := strings.ToLower(err.Error())
	return containsAny(errStr, "not found", "no such")
}

// IsUnauthorized returns true if the error indicates an authentication failure.
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrUnauthorized) {
		return true
	}
	errStr := strings.ToLower(err.Error())
	return containsAny(errStr, "unauthorized", "forbidden", "credential", "authentication")
}
