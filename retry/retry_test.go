package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
)

// RetrySuite is the test suite for the retry mechanism.
type RetrySuite struct {
	jet.Suite
}

// SetupSuite initializes the test suite.
func (s *RetrySuite) SetupSuite() {
	s.Suite.Init(nil)
}

// TestDoSuccess verifies a successful run on the first attempt.
func (s *RetrySuite) TestDoSuccess() {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Do(s.Ctx, cfg, func(ctx context.Context) error {
		attempts++
		return nil
	})

	s.NoError(err)
	s.Equal(1, attempts)
}

// TestDoRetryThenSuccess verifies success after several failed attempts.
func (s *RetrySuite) TestDoRetryThenSuccess() {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Do(s.Ctx, cfg, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	s.NoError(err)
	s.Equal(3, attempts)
}

// TestDoMaxAttemptsExceeded verifies reaching the maximum number of attempts.
func (s *RetrySuite) TestDoMaxAttemptsExceeded() {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	expectedErr := errors.New("persistent error")
	err := Do(s.Ctx, cfg, func(ctx context.Context) error {
		attempts++
		return expectedErr
	})

	s.Error(err)
	s.Equal(expectedErr, err)
	s.Equal(3, attempts)
}

// TestDoNonRetryableError verifies an immediate return for non-retryable errors.
func (s *RetrySuite) TestDoNonRetryableError() {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Do(s.Ctx, cfg, func(ctx context.Context) error {
		attempts++
		return NonRetryable(errors.New("non-retryable error"))
	})

	s.Error(err)
	s.Equal(1, attempts)
}

// TestDoContextCancelled verifies interruption on context cancellation.
func (s *RetrySuite) TestDoContextCancelled() {
	cfg := Config{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
	}

	ctx, cancel := context.WithTimeout(s.Ctx, 50*time.Millisecond)
	defer cancel()

	attempts := 0
	err := Do(ctx, cfg, func(ctx context.Context) error {
		attempts++
		return errors.New("error")
	})

	s.Error(err)
	s.Less(attempts, 5) // must stop before 5 attempts
}

// TestDoWithResultSuccess verifies a successful run returning a result.
func (s *RetrySuite) TestDoWithResultSuccess() {
	cfg := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	result, err := DoWithResult(s.Ctx, cfg, func(ctx context.Context) (int, error) {
		attempts++
		if attempts < 2 {
			return 0, errors.New("temporary error")
		}
		return 42, nil
	})

	s.NoError(err)
	s.Equal(42, result)
	s.Equal(2, attempts)
}

// TestDoWithResultMaxAttempts verifies the zero value is returned when attempts are exhausted.
func (s *RetrySuite) TestDoWithResultMaxAttempts() {
	cfg := Config{
		MaxAttempts:  2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	result, err := DoWithResult(s.Ctx, cfg, func(ctx context.Context) (string, error) {
		return "", errors.New("always fails")
	})

	s.Error(err)
	s.Empty(result)
}

// TestCalculateDelay verifies delay computation with exponential backoff.
func (s *RetrySuite) TestCalculateDelay() {
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       0, // no jitter for predictability
	}

	delay1 := cfg.calculateDelay(1)
	delay2 := cfg.calculateDelay(2)
	delay3 := cfg.calculateDelay(3)
	delay4 := cfg.calculateDelay(4)

	s.Equal(100*time.Millisecond, delay1)
	s.Equal(200*time.Millisecond, delay2)
	s.Equal(400*time.Millisecond, delay3)
	s.Equal(800*time.Millisecond, delay4)
}

// TestCalculateDelayMaxCap verifies the maximum delay cap.
func (s *RetrySuite) TestCalculateDelayMaxCap() {
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       0,
	}

	delay5 := cfg.calculateDelay(5) // 100 * 2^4 = 1600ms, but capped at 500ms

	s.Equal(500*time.Millisecond, delay5)
}

// TestCalculateDelayWithJitter verifies that jitter is added to the delay.
func (s *RetrySuite) TestCalculateDelayWithJitter() {
	cfg := Config{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.5, // 50% jitter
	}

	// verify jitter works - run several computations
	delays := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		delay := cfg.calculateDelay(1)
		delays[delay] = true
		// delay must be in range 50ms - 150ms (100ms ± 50%)
		s.GreaterOrEqual(delay, 50*time.Millisecond)
		s.LessOrEqual(delay, 150*time.Millisecond)
	}

	// values must differ due to jitter
	s.Greater(len(delays), 1)
}

// TestDefaultConfig verifies the default values.
func (s *RetrySuite) TestDefaultConfig() {
	cfg := DefaultConfig()

	s.Equal(3, cfg.MaxAttempts)
	s.Equal(500*time.Millisecond, cfg.InitialDelay)
	s.Equal(5*time.Second, cfg.MaxDelay)
	s.Equal(2.0, cfg.Multiplier)
	s.Equal(0.1, cfg.Jitter)
}

// TestRPCConfig verifies the RPC call configuration.
func (s *RetrySuite) TestRPCConfig() {
	cfg := RPCConfig()

	s.Equal(3, cfg.MaxAttempts)
	s.Equal(200*time.Millisecond, cfg.InitialDelay)
	s.Equal(2*time.Second, cfg.MaxDelay)
	s.Equal(2.0, cfg.Multiplier)
	s.Equal(0.2, cfg.Jitter)
}

// TestIsRetryable verifies retryability detection.
func (s *RetrySuite) TestIsRetryable() {
	s.False(IsRetryable(nil))
	s.True(IsRetryable(errors.New("regular error")))
	s.False(IsRetryable(NonRetryable(errors.New("non-retryable"))))
}

// TestNonRetryableNil verifies handling of a nil error.
func (s *RetrySuite) TestNonRetryableNil() {
	s.Nil(NonRetryable(nil))
}

// TestRetryableErrorUnwrap verifies Unwrap for RetryableError.
func (s *RetrySuite) TestRetryableErrorUnwrap() {
	original := errors.New("original error")
	wrapped := NonRetryable(original)

	s.Equal("original error", wrapped.Error())
	s.Equal(original, errors.Unwrap(wrapped))
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetrySuite))
}
