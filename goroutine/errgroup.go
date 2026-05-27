package goroutine

import (
	"context"
	"github.com/zloevil/jet"
	"sync"
)

// ErrGroup is a replica of a standard errgroup with panic handling and custom logging
// it allows to run multiple goroutines and wait unless all finished either one of them failed
// Once one fail, all other are cancelled at once
type ErrGroup interface {
	// Go calls the given function in a new goroutine.
	//
	// The first call to return a non-nil error cancels the group; its error will be
	// returned by Wait.
	Go(f func() error)
	// Wait blocks until all function calls from the Go method have returned, then
	// returns the first non-nil error (if any) from them.
	Wait() error
	// Cancel cancels all goroutines running
	Cancel()
	// CancelFunc returns cancel function defined by cancelled context
	CancelFunc() func()
	// WithLogger allows to specify prepared logger
	WithLogger(logger jet.CLogger) ErrGroup
	// WithLoggerFn allows to specify logger func
	WithLoggerFn(loggerFn jet.CLoggerFunc) ErrGroup
	// Mth allows to specify method to log in case of panic
	// it works only for logger func
	Mth(method string) ErrGroup
	// Cmp allows to specify component to log in case of panic
	// it works only for logger func
	Cmp(component string) ErrGroup
}

type errGroup struct {
	cancel   func()
	wg       sync.WaitGroup
	errOnce  sync.Once
	err      error
	logger   jet.CLogger
	loggerFn jet.CLoggerFunc
	ctx      context.Context
	mth, cmp string
}

// NewGroup returns a new Group and an associated Context derived from ctx.
//
// The derived Context is canceled the first time a function passed to Go
// returns a non-nil error or the first time Wait returns, whichever occurs
// first.
func NewGroup(ctx context.Context) ErrGroup {
	_, cancel := context.WithCancel(ctx)
	return &errGroup{cancel: cancel, ctx: ctx}
}

func (g *errGroup) WithLogger(logger jet.CLogger) ErrGroup {
	g.logger = logger
	return g
}

func (g *errGroup) WithLoggerFn(loggerFn jet.CLoggerFunc) ErrGroup {
	g.loggerFn = loggerFn
	return g
}

func (g *errGroup) Mth(method string) ErrGroup {
	g.mth = method
	return g
}

func (g *errGroup) Cmp(component string) ErrGroup {
	g.cmp = component
	return g
}

func (g *errGroup) Cancel() {
	if g.cancel != nil {
		g.cancel()
	}
}

func (g *errGroup) CancelFunc() func() {
	return g.cancel
}

func (g *errGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return g.err
}

func (g *errGroup) Go(f func() error) {

	// check if logger passed
	if g.logger == nil && g.loggerFn == nil {
		panic(ErrGoroutineNoLogger(g.ctx))
	}

	// define logger params
	var logger jet.CLogger
	if g.logger != nil {
		logger = g.logger.C(g.ctx)
	} else {
		logger = g.loggerFn().Cmp(g.cmp).Mth(g.mth).C(g.ctx)
	}

	// prepare panic wrapper
	wrapper := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = jet.ErrPanic(g.ctx, r)
				logger.E(err).St().Err()
			}
		}()
		err = f()
		return
	}

	g.wg.Add(1)

	go func() {
		defer g.wg.Done()

		if err := wrapper(); err != nil {
			g.errOnce.Do(func() {
				g.err = err
				if g.cancel != nil {
					g.cancel()
				}
			})
		}
	}()
}
