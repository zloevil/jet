package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/zloevil/jet"
)

// Config holds the settings for the retry mechanism.
type Config struct {
	// MaxAttempts is the maximum number of attempts (including the first one).
	MaxAttempts int
	// InitialDelay is the initial delay before the first retry.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between attempts.
	MaxDelay time.Duration
	// Multiplier is the factor for exponential delay growth.
	Multiplier float64
	// Jitter adds randomness to the delay (0.0 - 1.0).
	Jitter float64
}

// DefaultConfig returns the default configuration.
// Suitable for most operations with moderate delays.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,                      // 3 attempts (1 initial + 2 retries)
		InitialDelay: 500 * time.Millisecond, // initial delay 500ms
		MaxDelay:     5 * time.Second,        // max delay 5s
		Multiplier:   2.0,                    // delay doubling: 500ms -> 1s -> 2s -> ...
		Jitter:       0.1,                    // ±10% randomness to avoid thundering herd
	}
}

// RPCConfig returns a configuration optimized for RPC calls.
// More aggressive retries with smaller delays for fast recovery.
func RPCConfig() Config {
	return Config{
		MaxAttempts:  3,                      // 3 attempts (1 initial + 2 retries)
		InitialDelay: 200 * time.Millisecond, // initial delay 200ms (faster than default)
		MaxDelay:     2 * time.Second,        // max delay 2s (smaller than default)
		Multiplier:   2.0,                    // delay doubling: 200ms -> 400ms -> 800ms -> ...
		Jitter:       0.2,                    // ±20% randomness (more, to spread load)
	}
}

// RetryableError represents an error and whether it may be retried.
type RetryableError struct {
	Err       error
	Retryable bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable reports whether the operation may be retried for the given error.
// By default all errors are considered retryable, except those explicitly marked.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var re *RetryableError
	if errors.As(err, &re) {
		return re.Retryable
	}
	return true // by default all errors are retryable
}

// NonRetryable wraps an error, marking it as non-retryable.
func NonRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err, Retryable: false}
}

// Do runs the function with retry logic.
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// check the context before the attempt
		if ctx.Err() != nil {
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// check whether it can be retried
		if !IsRetryable(err) {
			return err
		}

		// last attempt - do not wait
		if attempt == cfg.MaxAttempts {
			break
		}

		// compute the delay with exponential backoff
		delay := cfg.calculateDelay(attempt)

		// wait, respecting the context
		select {
		case <-time.After(delay):
			// proceed to the next attempt
		case <-ctx.Done():
			return lastErr
		}
	}

	return lastErr
}

// DoWithResult runs the function with retry logic and returns a result.
func DoWithResult[T any](ctx context.Context, cfg Config, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// check the context before the attempt
		if ctx.Err() != nil {
			if lastErr != nil {
				return result, lastErr
			}
			return result, ctx.Err()
		}

		res, err := fn(ctx)
		if err == nil {
			return res, nil
		}

		lastErr = err

		// check whether it can be retried
		if !IsRetryable(err) {
			return result, err
		}

		// last attempt - do not wait
		if attempt == cfg.MaxAttempts {
			break
		}

		// compute the delay with exponential backoff
		delay := cfg.calculateDelay(attempt)

		// wait, respecting the context
		select {
		case <-time.After(delay):
			// proceed to the next attempt
		case <-ctx.Done():
			return result, lastErr
		}
	}

	return result, lastErr
}

// DoWithLogger runs the function with retry logic and logging.
func DoWithLogger(ctx context.Context, cfg Config, logger jet.CLogger, operation string, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// check the context before the attempt
		if ctx.Err() != nil {
			if lastErr != nil {
				return lastErr
			}
			return ctx.Err()
		}

		err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				logger.F(jet.KV{
					"operation": operation,
					"attempt":   attempt,
				}).Dbg("operation succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// check whether it can be retried
		if !IsRetryable(err) {
			logger.F(jet.KV{
				"operation": operation,
				"attempt":   attempt,
			}).E(err).Dbg("non-retryable error")
			return err
		}

		// last attempt - do not wait
		if attempt == cfg.MaxAttempts {
			logger.F(jet.KV{
				"operation":    operation,
				"max_attempts": cfg.MaxAttempts,
			}).E(err).Warn("max retry attempts reached")
			break
		}

		// compute the delay with exponential backoff
		delay := cfg.calculateDelay(attempt)

		logger.F(jet.KV{
			"operation": operation,
			"attempt":   attempt,
			"delay_ms":  delay.Milliseconds(),
		}).E(err).Dbg("retrying after error")

		// wait, respecting the context
		select {
		case <-time.After(delay):
			// proceed to the next attempt
		case <-ctx.Done():
			return lastErr
		}
	}

	return lastErr
}

// DoWithResultAndLogger runs the function with retry logic and logging, and returns a result.
func DoWithResultAndLogger[T any](ctx context.Context, cfg Config, logger jet.CLogger, operation string, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		// check the context before the attempt
		if ctx.Err() != nil {
			if lastErr != nil {
				return result, lastErr
			}
			return result, ctx.Err()
		}

		res, err := fn(ctx)
		if err == nil {
			if attempt > 1 {
				logger.F(jet.KV{
					"operation": operation,
					"attempt":   attempt,
				}).Dbg("operation succeeded after retry")
			}
			return res, nil
		}

		lastErr = err

		// check whether it can be retried
		if !IsRetryable(err) {
			logger.F(jet.KV{
				"operation": operation,
				"attempt":   attempt,
			}).E(err).Dbg("non-retryable error")
			return result, err
		}

		// last attempt - do not wait
		if attempt == cfg.MaxAttempts {
			logger.F(jet.KV{
				"operation":    operation,
				"max_attempts": cfg.MaxAttempts,
			}).E(err).Warn("max retry attempts reached")
			break
		}

		// compute the delay with exponential backoff
		delay := cfg.calculateDelay(attempt)

		logger.F(jet.KV{
			"operation": operation,
			"attempt":   attempt,
			"delay_ms":  delay.Milliseconds(),
		}).E(err).Dbg("retrying after error")

		// wait, respecting the context
		select {
		case <-time.After(delay):
			// proceed to the next attempt
		case <-ctx.Done():
			return result, lastErr
		}
	}

	return result, lastErr
}

// calculateDelay computes the delay for the given attempt.
func (c Config) calculateDelay(attempt int) time.Duration {
	// exponential backoff: delay = initialDelay * multiplier^(attempt-1)
	delay := float64(c.InitialDelay) * math.Pow(c.Multiplier, float64(attempt-1))

	// apply the maximum cap
	if delay > float64(c.MaxDelay) {
		delay = float64(c.MaxDelay)
	}

	// add jitter
	if c.Jitter > 0 {
		jitterRange := delay * c.Jitter
		delay = delay + (rand.Float64()*2-1)*jitterRange
	}

	return time.Duration(delay)
}
