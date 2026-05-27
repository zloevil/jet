package goroutine

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/zloevil/jet"
	"sync"
	"testing"
	"time"
)

var (
	logger = jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})
	logf   = func() jet.CLogger {
		return jet.L(logger)
	}
)

func Test_Goroutine_WhenPanic_WithRetry(t *testing.T) {
	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(4)
	New().
		WithLoggerFn(logf).
		WithRetry(3).
		WithRetryDelay(time.Millisecond*100).
		Mth("test-method").
		Cmp("test-component").
		Go(ctx, func() {
			time.Sleep(time.Millisecond * 300)
			fmt.Println("running")
			wg.Done()
			panic("panic")
		})
	wg.Wait()
}

func Test_Goroutine_WhenPanic_WithoutRetry(t *testing.T) {
	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(1)
	New().
		WithLoggerFn(logf).
		Mth("test-method").
		Cmp("test-component").
		WithRetryDelay(time.Millisecond*100).
		Go(ctx, func() {
			time.Sleep(time.Millisecond * 300)
			fmt.Println("running")
			wg.Done()
			panic("panic")
		})
	wg.Wait()
	time.Sleep(time.Millisecond * 300)
}

func Test_Goroutine_WhenPanic_WithEndlessRetry(t *testing.T) {
	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(3)
	New().
		WithLoggerFn(logf).
		Mth("test-method").
		Cmp("test-component").
		WithRetry(-1).
		WithRetryDelay(time.Millisecond*100).
		Go(ctx, func() {
			time.Sleep(time.Millisecond * 300)
			fmt.Println("running")
			wg.Done()
			panic("panic")
		})
	wg.Wait()
}

func Test_Goroutine_WhenNoPanic(t *testing.T) {
	ctx := context.Background()
	wg := sync.WaitGroup{}
	wg.Add(1)
	New().
		WithLoggerFn(logf).
		Mth("test-method").
		Cmp("test-component").
		WithRetryDelay(time.Millisecond*100).
		WithRetry(5).
		Go(ctx, func() {
			time.Sleep(time.Millisecond * 300)
			fmt.Println("running")
			wg.Done()
		})
	wg.Wait()
}

func Test_ErrGroup_WhenNoErrorsOrPanic(t *testing.T) {

	eg := NewGroup(context.Background()).
		WithLoggerFn(logf).
		Mth("test-method").
		Cmp("test-component")

	eg.Go(func() error {
		time.Sleep(time.Millisecond * 300)
		fmt.Println("running 1")
		return nil
	})

	eg.Go(func() error {
		time.Sleep(time.Millisecond * 300)
		fmt.Println("running 2")
		return nil
	})

	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}

func Test_ErrGroup_WhenFirstReturnsError(t *testing.T) {

	eg := NewGroup(context.Background()).
		WithLoggerFn(logf).
		Mth("test-method").
		Cmp("test-component")

	eg.Go(func() error {
		defer fmt.Println("finished 1")
		time.Sleep(time.Millisecond * 300)
		fmt.Println("running 1")
		return fmt.Errorf("error")
	})

	eg.Go(func() error {
		defer fmt.Println("finished 2")
		time.Sleep(time.Millisecond * 500)
		fmt.Println("running 2")
		return nil
	})

	err := eg.Wait()
	assert.Error(t, err)
}

func Test_ErrGroup_WhenFirstPanics(t *testing.T) {

	eg := NewGroup(context.Background()).
		WithLoggerFn(logf).
		Mth("test-method").
		Cmp("test-component")

	eg.Go(func() error {
		defer fmt.Println("finished 1")
		time.Sleep(time.Millisecond * 300)
		fmt.Println("running 1")
		panic(fmt.Errorf("error"))
	})

	eg.Go(func() error {
		defer fmt.Println("finished 2")
		time.Sleep(time.Millisecond * 500)
		fmt.Println("running 2")
		return nil
	})

	err := eg.Wait()
	assert.Error(t, err)
}
