package cluster_test

import (
	"context"

	"github.com/zloevil/jet/cluster"
)

// exampleConfig is the service's typed configuration.
type exampleConfig struct {
	HTTP struct{ Port string }
}

// exampleApp implements cluster.Bootstrap.
type exampleApp struct{}

func (*exampleApp) Init(ctx context.Context, cfg any) error {
	c := cfg.(*exampleConfig) // cluster passes a *exampleConfig
	_ = c                     // build dependencies from config here
	return nil
}

func (*exampleApp) Start(ctx context.Context) error { return nil } // start servers/consumers
func (*exampleApp) Close(ctx context.Context)       {}             // release resources

// Example shows the minimal entry point of a service: cluster handles config
// loading, the CLI, signal handling and ordered shutdown.
func Example() {
	svc := cluster.New[exampleConfig]("my-service", &exampleApp{})
	_ = svc.Execute()
}
