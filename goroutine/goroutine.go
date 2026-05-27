package goroutine

import (
	"context"
	"github.com/zloevil/jet"
	"time"
)

const (
	RetryDelay   = time.Second
	Unrestricted = -1
)

// Goroutine provides a wrapper around native GO goroutine with panic recovery and retry support
type Goroutine interface {
	// Go executes a f func as a goroutine
	Go(ctx context.Context, f func())
	// WithLogger allows to specify prepared logger
	WithLogger(logger jet.CLogger) Goroutine
	// WithLoggerFn allows to specify logger func
	WithLoggerFn(loggerFn jet.CLoggerFunc) Goroutine
	// WithRetry allows to specify retry count
	// if retry less than 0, number of retries isn't restricted
	WithRetry(retry int) Goroutine
	// WithRetryDelay specifies delay before retry
	WithRetryDelay(delay time.Duration) Goroutine
	// Mth allows to specify method to log in case of panic
	// it works only for logger func
	Mth(method string) Goroutine
	// Cmp allows to specify component to log in case of panic
	// it works only for logger func
	Cmp(component string) Goroutine
}

type goroutine struct {
	logger   jet.CLogger
	loggerFn jet.CLoggerFunc
	retry    int
	mth, cmp string
	delay    time.Duration
}

func New() Goroutine {
	return &goroutine{
		delay: RetryDelay,
	}
}

func (g *goroutine) WithLogger(logger jet.CLogger) Goroutine {
	g.logger = logger
	return g
}

func (g *goroutine) WithLoggerFn(loggerFn jet.CLoggerFunc) Goroutine {
	g.loggerFn = loggerFn
	return g
}

func (g *goroutine) WithRetry(retry int) Goroutine {
	g.retry = retry
	return g
}

// WithRetryDelay specifies period between retry
func (g *goroutine) WithRetryDelay(delay time.Duration) Goroutine {
	g.delay = delay
	return g
}

func (g *goroutine) Mth(method string) Goroutine {
	g.mth = method
	return g
}

func (g *goroutine) Cmp(component string) Goroutine {
	g.cmp = component
	return g
}

func (g *goroutine) Go(ctx context.Context, f func()) {

	// copy context to avoid "cancelled context error"
	ctx = jet.Copy(ctx)

	// check if logger passed
	if g.logger == nil && g.loggerFn == nil {
		panic(ErrGoroutineNoLogger(ctx))
	}

	// define logger params
	var logger jet.CLogger
	if g.logger != nil {
		logger = g.logger.C(ctx)
	} else {
		logger = g.loggerFn().Cmp(g.cmp).Mth(g.mth).C(ctx)
	}

	// prepare panic wrapper
	wrapper := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = jet.ErrPanic(ctx, r)
				logger.E(err).St().Err()
			}
		}()
		f()
		return
	}
	retryCounter := 0
	go func() {
		for {
			if err := wrapper(); err != nil && (retryCounter < g.retry || g.retry < 0) {
				logger.Dbg("panic retry")
				// wait for some time before retry to avoid overloading in case of unrecoverable error
				time.Sleep(g.delay)
				// inc retry counter
				retryCounter++
			} else {
				break
			}
		}
	}()
}
