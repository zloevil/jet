package monitoring

import "github.com/prometheus/client_golang/prometheus"

type MetricsCollection []prometheus.Collector

type MetricsCollector func() MetricsCollection

type MetricsProvider interface {
	GetCollector() MetricsCollector
}

type Config struct {
	// Enabled indicates if monitoring enabled
	Enabled bool
	// Port indicates on which port to listen
	Port string
	// UrlPath (default /metrics)
	UrlPath string
	// GoMetrics if true, then built-in metrics are exposed
	GoMetrics bool `mapstructure:"go_metrics"`
}

// MetricsServer exposes metrics via HTTP server
type MetricsServer interface {
	// Init initializes server with given opts
	Init(config *Config, MetricProviders ...MetricsProvider) error
	// Listen starts async listening
	Listen()
	// Close closes connection
	Close()
}
