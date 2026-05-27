package batch

import (
	"context"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
	"time"
)

// Worker allows writing data in a batch manner
// events to write fire either by time interval or by number of items waiting to be written
type Worker[T any] interface {
	// Start starts worker
	Start(ctx context.Context)
	// Write calls to pass a single item to be written
	Write(ctx context.Context, item *T)
	// Close closes worker
	Close(ctx context.Context)
}

// Writer interface must be implemented and passed by calling side
type Writer[T any] interface {
	// Write writes batch items somewhere
	Write(ctx context.Context, items []*T) error
}

type Options struct {
	Interval    time.Duration // Interval after which writing happens
	MaxItems    int           // MaxItems specifies number of items in a queue to be written
	MaxCapacity int           // MaxCapacity capacity of channel before locking
}

type batchWorker[T any] struct {
	itemChan  chan *T
	cancelCtx context.Context
	cancelFn  context.CancelFunc
	opt       *Options
	logger    jet.CLoggerFunc
	writer    Writer[T]
}

func NewBatchWorker[T any](writer Writer[T], opt *Options, logger jet.CLoggerFunc) Worker[T] {
	r := &batchWorker[T]{
		opt:    opt,
		logger: logger,
		writer: writer,
	}
	return r
}

func (f *batchWorker[T]) l() jet.CLogger {
	return f.logger().Cmp("batch-worker")
}

func (f *batchWorker[T]) Start(ctx context.Context) {
	f.l().C(ctx).Mth("start").Dbg()

	// cancel running forcibly
	if f.cancelFn != nil {
		f.cancelFn()
	}
	f.itemChan = make(chan *T, f.opt.MaxCapacity)

	// init cancellation context
	f.cancelCtx, f.cancelFn = context.WithCancel(ctx)

	// run writer in a separate goroutine
	goroutine.New().WithLogger(f.l().C(ctx).Mth("writer")).WithRetry(goroutine.Unrestricted).Go(ctx, func() { f.worker(ctx) })
}

func (f *batchWorker[T]) Write(ctx context.Context, item *T) {
	f.itemChan <- item
}

func (f *batchWorker[T]) Close(ctx context.Context) {
	f.l().C(ctx).Mth("close").Dbg()
	if f.cancelFn != nil {
		f.cancelFn()
	}
	f.cancelFn = nil
	close(f.itemChan)
}

func (f *batchWorker[T]) worker(ctx context.Context) {
	l := f.l().C(ctx).Mth("worker").Dbg()

	for keepGoing := true; keepGoing; {

		var batch []*T

		expire := time.After(f.opt.Interval)

		for {
			select {

			case ev, ok := <-f.itemChan:
				if !ok {
					keepGoing = false
					goto flush
				}
				batch = append(batch, ev)
				if len(batch) >= f.opt.MaxItems {
					goto flush
				}

			// flush when timeout expires
			case <-expire:
				goto flush

			// leave when context cancelled
			case <-ctx.Done():
				keepGoing = false
				l.Inf("close")
				goto flush
			}
		}
	flush:
		if len(batch) > 0 {
			// write
			if err := f.writer.Write(ctx, batch); err != nil {
				f.l().C(ctx).Mth("writer").E(err).Err()
			}
		}
	}
}
