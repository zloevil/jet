// Package monitoring exposes Prometheus metrics over HTTP.
//
// MetricsServer serves a /metrics endpoint from a private registry populated by
// the MetricsProvider implementations passed to Init. The package also ships
// ready-made collectors: ErrorMonitoring (classifies AppError into
// business/system/panic counters) and RegexpMonitoring (counts text matches).
package monitoring
