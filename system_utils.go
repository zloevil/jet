package jet

import (
	"context"
	"fmt"
	"go.uber.org/atomic"
	"time"
)

const (
	ErrCodeSysUtilsRetryMaxAttempts = "SU-001"
	ErrCodeSysUtilsRetryFnEmpty     = "SU-002"
)

var (
	ErrSysUtilsRetryMaxAttempts = func(ctx context.Context) error {
		return NewAppErrBuilder(ErrCodeSysUtilsRetryMaxAttempts, "max attempts reached").C(ctx).Err()
	}
	ErrSysUtilsRetryFnEmpty = func(ctx context.Context) error {
		return NewAppErrBuilder(ErrCodeSysUtilsRetryFnEmpty, "function empty").C(ctx).Err()
	}
)

// Await allows awaiting some state by periodically hitting fn unless either it returns true or error or timeout
// It returns nil when fn results true
func Await(fn func() (bool, error), tick, timeout time.Duration) chan error {
	c := make(chan error)
	go func() {
		// first try without ticker
		res, err := fn()
		if err != nil {
			c <- err
			return
		}
		if res {
			c <- nil
			return
		}
		// if first try fails, run ticker
		ticker := time.NewTicker(tick)
		defer ticker.Stop()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		for {
			select {
			case <-ticker.C:
				res, err := fn()
				if err != nil {
					c <- err
					return
				}
				if res {
					c <- nil
					return
				}
			case <-ctx.Done():
				c <- fmt.Errorf("timeout")
				return
			}
		}
	}()
	return c
}

// WaitGroup wait group with duration timeout
type WaitGroup struct {
	i *atomic.Int64
}

func NewWG() *WaitGroup {
	return &WaitGroup{
		i: atomic.NewInt64(0),
	}
}

func (w *WaitGroup) Add(delta int) {
	w.i.Add(int64(delta))
}

func (w *WaitGroup) Done() {
	w.i.Dec()
}

func (w *WaitGroup) Wait(to time.Duration) bool {
	for {
		select {
		case <-time.After(to):
			return false
		default:
			if w.i.Load() <= 0 {
				return true
			}
		}
	}
}

type RetryCfg struct {
	NextAttemptFn func(curAttemptTime time.Time, curAttempt int) time.Time
	MaxAttempts   int
}

// Retry takes a generic function with parameters and run it with retry wrapper
// retry params are configured with RetryCfg
func Retry[TIn, TOut any](ctx context.Context, fn func(ctx context.Context, in TIn) (TOut, error), in TIn, cfg RetryCfg) (TOut, error) {

	if fn == nil {
		return *new(TOut), ErrSysUtilsRetryFnEmpty(ctx)
	}

	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.NextAttemptFn == nil {
		cfg.NextAttemptFn = func(curAttemptTime time.Time, attempt int) time.Time {
			return curAttemptTime.Add(time.Duration(60*attempt) * time.Second)
		}
	}

	for i := 0; i < cfg.MaxAttempts; i++ {
		r, err := fn(ctx, in)
		if err == nil {
			return r, nil
		}
		if i < cfg.MaxAttempts-1 {
			next := cfg.NextAttemptFn(time.Now(), i)
			time.Sleep(time.Until(next))
		}
	}

	return *new(TOut), ErrSysUtilsRetryMaxAttempts(ctx)
}
