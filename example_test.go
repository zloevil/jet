package jet_test

import (
	"context"
	"fmt"

	"github.com/zloevil/jet"
)

func ExampleNewAppErrBuilder() {
	err := jet.NewAppErrBuilder("ORD-001", "order not found: %s", "42").Business().Err()

	appErr, _ := jet.IsAppErr(err)
	fmt.Println(appErr.Code())
	fmt.Println(appErr.Type())
	fmt.Println(appErr.Message())
	// Output:
	// ORD-001
	// business
	// order not found: 42
}

func ExampleMap() {
	doubled := jet.Map([]int{1, 2, 3}, func(i int) int { return i * 2 })
	fmt.Println(doubled)
	// Output: [2 4 6]
}

func ExampleFilter() {
	even := jet.Filter([]int{1, 2, 3, 4}, func(i int) bool { return i%2 == 0 })
	fmt.Println(even)
	// Output: [2 4]
}

func ExampleRequestContext() {
	ctx := jet.NewRequestCtx().WithRequestId("req-1").WithUser("u-1", "alice").ToContext(context.Background())

	rc, _ := jet.Request(ctx)
	fmt.Println(rc.GetRequestId(), rc.GetUserId(), rc.GetUsername())
	// Output: req-1 u-1 alice
}

// ExampleInitLogger shows logger setup and the chainable, context-aware API.
func ExampleInitLogger() {
	logger := jet.InitLogger(&jet.LogConfig{Level: jet.InfoLevel, Format: jet.FormatterJson})

	log := jet.L(logger)
	log.Cmp("orders").Mth("Create").F(jet.KV{"id": "42"}).Inf("order created")
	// log lines are timestamped and written to stdout, so they are not asserted here.
}

// ExampleNewConfigLoader loads a typed config from YAML with env overrides.
func ExampleNewConfigLoader() {
	type Config struct {
		HTTP struct{ Port string }
	}

	cfg, err := jet.NewConfigLoader[Config]().WithPath("./config.yml").WithPrefix("MYSVC").Load()
	if err != nil {
		return
	}
	_ = cfg
}
