package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.InitialWait != 100*time.Millisecond {
		t.Errorf("InitialWait = %v, want 100ms", cfg.InitialWait)
	}
	if cfg.MaxWait != 5*time.Second {
		t.Errorf("MaxWait = %v, want 5s", cfg.MaxWait)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %f, want 2.0", cfg.Multiplier)
	}
}

func TestDo_Success(t *testing.T) {
	attempts := 0
	err := Do(context.Background(), nil, nil, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestDo_RetryThenSuccess(t *testing.T) {
	cfg := &Config{
		MaxAttempts: 3,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	attempts := 0
	err := Do(context.Background(), cfg, nil, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestDo_AllAttemptsFail(t *testing.T) {
	cfg := &Config{
		MaxAttempts: 3,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	attempts := 0
	expectedErr := errors.New("persistent error")
	err := Do(context.Background(), cfg, nil, func() error {
		attempts++
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestDo_NonRetryableError(t *testing.T) {
	cfg := &Config{
		MaxAttempts: 3,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	nonRetryableErr := errors.New("non-retryable error")
	isRetryable := func(err error) bool {
		return err != nonRetryableErr
	}

	attempts := 0
	err := Do(context.Background(), cfg, isRetryable, func() error {
		attempts++
		return nonRetryableErr
	})

	if err != nonRetryableErr {
		t.Errorf("error = %v, want %v", err, nonRetryableErr)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (should not retry)", attempts)
	}
}

func TestDo_ContextCanceled(t *testing.T) {
	cfg := &Config{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     1 * time.Second,
		Multiplier:  2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	attempts := 0
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := Do(ctx, cfg, nil, func() error {
		attempts++
		return errors.New("error")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestDo_ContextCanceledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before starting

	err := Do(ctx, nil, nil, func() error {
		t.Error("operation should not be called")
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestDoWithResult_Success(t *testing.T) {
	cfg := &Config{
		MaxAttempts: 3,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	attempts := 0
	result, err := DoWithResult(context.Background(), cfg, nil, func() (int, error) {
		attempts++
		if attempts < 2 {
			return 0, errors.New("temporary error")
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestDoWithResult_Failure(t *testing.T) {
	cfg := &Config{
		MaxAttempts: 2,
		InitialWait: 1 * time.Millisecond,
		MaxWait:     10 * time.Millisecond,
		Multiplier:  2.0,
	}

	expectedErr := errors.New("persistent error")
	result, err := DoWithResult(context.Background(), cfg, nil, func() (string, error) {
		return "", expectedErr
	})

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
	if result != "" {
		t.Errorf("result = %q, want empty string", result)
	}
}

func TestAlwaysRetry(t *testing.T) {
	if !AlwaysRetry(errors.New("any error")) {
		t.Error("AlwaysRetry should return true for any error")
	}
	if AlwaysRetry(nil) {
		t.Error("AlwaysRetry should return false for nil error")
	}
}
