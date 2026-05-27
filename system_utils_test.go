package jet

import (
	"context"
	"errors"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type RetryTestSuite struct {
	Suite
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetryTestSuite))
}

func (s *RetryTestSuite) SetupSuite() {
	s.Suite.Init(func() CLogger { return L(InitLogger(&LogConfig{Level: TraceLevel})) })
}

func (s *RetryTestSuite) Test_SuccessOnFirstAttempt() {
	fn := func(ctx context.Context, in int) (string, error) {
		return "ok", nil
	}

	cfg := RetryCfg{
		MaxAttempts:   3,
		NextAttemptFn: dummyBackoff,
	}

	out, err := Retry(s.Ctx, fn, 42, cfg)
	s.NoError(err)
	s.Equal("ok", out)
}

func (s *RetryTestSuite) Test_SuccessAfterRetries() {
	attempts := 0
	fn := func(ctx context.Context, in int) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("fail")
		}
		return "success", nil
	}

	cfg := RetryCfg{
		MaxAttempts:   5,
		NextAttemptFn: fixedDelay(10 * time.Millisecond),
	}

	start := time.Now()
	out, err := Retry(s.Ctx, fn, 42, cfg)
	elapsed := time.Since(start)

	s.NoError(err)
	s.Equal("success", out)
	s.GreaterOrEqual(elapsed, 20*time.Millisecond)
	s.Equal(3, attempts)
}

func (s *RetryTestSuite) Test_AllAttemptsFail() {
	fn := func(ctx context.Context, in int) (string, error) {
		return "", errors.New("still failing")
	}

	cfg := RetryCfg{
		MaxAttempts:   3,
		NextAttemptFn: fixedDelay(5 * time.Millisecond),
	}

	_, err := Retry(s.Ctx, fn, 42, cfg)
	s.Error(err)
	s.AssertAppErr(err, ErrCodeSysUtilsRetryMaxAttempts)
}

func (s *RetryTestSuite) Test_NilFunction() {
	cfg := RetryCfg{
		MaxAttempts:   3,
		NextAttemptFn: fixedDelay(1 * time.Millisecond),
	}

	out, err := Retry[string, string](s.Ctx, nil, "input", cfg)
	s.Error(err)
	s.AssertAppErr(err, ErrCodeSysUtilsRetryFnEmpty)
	s.Empty(out)
}

// dummyBackoff returns a retry time 1ms in the future
func dummyBackoff(now time.Time, _ int) time.Time {
	return now.Add(1 * time.Millisecond)
}

// fixedDelay returns a retry time after a fixed duration
func fixedDelay(d time.Duration) func(time.Time, int) time.Time {
	return func(now time.Time, _ int) time.Time {
		return now.Add(d)
	}
}
