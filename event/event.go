package event

import (
	"context"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
	"sync"
)

type Event[T any] interface {
	Register(handler func(ctx context.Context, data T) error)
	ExecuteAsync(ctx context.Context, data T)
	Execute(ctx context.Context, data T) error
	Wait()
}

type eventHandler[T any] struct {
	handlers []func(ctx context.Context, data T) error
	mu       sync.RWMutex
	wg       sync.WaitGroup
	logger   jet.CLoggerFunc
}

func (e *eventHandler[T]) l() jet.CLogger {
	return e.logger().Cmp("event-handler")
}

func (e *eventHandler[T]) Register(handler func(ctx context.Context, data T) error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.handlers = append(e.handlers, handler)
}

func (e *eventHandler[T]) Wait() {
	e.wg.Wait()
}

func (e *eventHandler[T]) Execute(ctx context.Context, data T) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, handler := range e.handlers {
		if err := handler(ctx, data); err != nil {
			return err
		}
	}
	return nil
}

func (e *eventHandler[T]) ExecuteAsync(ctx context.Context, data T) {
	e.wg.Add(1)
	goroutine.New().WithLogger(e.l().C(ctx).Mth("execute-async")).Go(ctx, func() {
		defer e.wg.Done()

		e.mu.RLock()
		defer e.mu.RUnlock()

		for _, handler := range e.handlers {
			if err := handler(ctx, data); err != nil {
				e.l().E(err).Err()
			}
		}
	})
}

func NewEventHandler[T any](logger jet.CLoggerFunc) Event[T] {
	return &eventHandler[T]{
		logger: logger,
	}
}
