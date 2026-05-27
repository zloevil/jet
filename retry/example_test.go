package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zloevil/jet/retry"
)

func ExampleDo() {
	attempts := 0
	err := retry.Do(context.Background(),
		retry.Config{MaxAttempts: 3, InitialDelay: time.Millisecond, Multiplier: 1},
		func(ctx context.Context) error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary failure")
			}
			return nil
		},
	)

	fmt.Println(attempts, err)
	// Output: 3 <nil>
}

func ExampleNonRetryable() {
	attempts := 0
	err := retry.Do(context.Background(), retry.DefaultConfig(), func(ctx context.Context) error {
		attempts++
		return retry.NonRetryable(errors.New("fatal"))
	})

	fmt.Println(attempts)
	fmt.Println(err)
	// Output:
	// 1
	// fatal
}
