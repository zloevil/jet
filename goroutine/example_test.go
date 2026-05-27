package goroutine_test

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
)

func ExampleErrGroup() {
	logger := jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.ErrorLevel}))
	eg := goroutine.NewGroup(context.Background()).WithLogger(logger)

	var n int64
	for i := 0; i < 5; i++ {
		eg.Go(func() error {
			atomic.AddInt64(&n, 1)
			return nil
		})
	}

	err := eg.Wait()
	fmt.Println(atomic.LoadInt64(&n), err)
	// Output: 5 <nil>
}
