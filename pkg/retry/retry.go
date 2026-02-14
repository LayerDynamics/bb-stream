// Package retry provides exponential backoff retry logic.
package retry

import (
	"context"
	"math/rand"
	"time"
)

// Config configures retry behavior.
type Config struct {
	MaxAttempts int           // Maximum number of attempts (default 3)
	InitialWait time.Duration // Initial wait time before first retry (default 100ms)
	MaxWait     time.Duration // Maximum wait time between retries (default 5s)
	Multiplier  float64       // Multiplier for each successive retry (default 2.0)
}

// DefaultConfig returns sensible defaults for retry behavior.
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Multiplier:  2.0,
	}
}

// IsRetryable is a function that determines if an error is retryable.
type IsRetryable func(error) bool

// AlwaysRetry returns true for any non-nil error.
func AlwaysRetry(err error) bool {
	return err != nil
}

// Do executes the operation with exponential backoff retry.
// It retries on errors that pass the isRetryable check.
// Returns the last error if all attempts fail.
func Do(ctx context.Context, cfg *Config, isRetryable IsRetryable, operation func() error) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if isRetryable == nil {
		isRetryable = AlwaysRetry
	}

	var lastErr error
	wait := cfg.InitialWait

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// Check context before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if !isRetryable(err) {
			return err
		}

		// Don't wait after the last attempt
		if attempt == cfg.MaxAttempts {
			break
		}

		// Add jitter (±25% of wait time)
		jitter := time.Duration(rand.Int63n(int64(wait/4))) - wait/8
		sleepTime := wait + jitter

		// Cap at MaxWait
		if sleepTime > cfg.MaxWait {
			sleepTime = cfg.MaxWait
		}

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepTime):
		}

		// Increase wait time for next attempt
		wait = time.Duration(float64(wait) * cfg.Multiplier)
	}

	return lastErr
}

// DoWithResult executes an operation that returns a result with retry.
func DoWithResult[T any](ctx context.Context, cfg *Config, isRetryable IsRetryable, operation func() (T, error)) (T, error) {
	var result T
	err := Do(ctx, cfg, isRetryable, func() error {
		var opErr error
		result, opErr = operation()
		return opErr
	})
	return result, err
}
