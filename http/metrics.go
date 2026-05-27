package http

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zloevil/jet/monitoring"
	"strconv"
	"strings"
)

const (
	RequestMetricsLatencyMsGaugeName = "request_metrics_latency_ms_gauge"
	RequestMetricsErrorCounterName   = "request_metrics_error_counter"

	RequestMetricsLabelUrl             = "url"
	RequestMetricsLabelIntegrationName = "name"
	RequestMetricsLabelErrorCodeName   = "code"
)

type MetricsProvider interface {
	GetCollector() monitoring.MetricsCollector
	RequestErrorInc(ctx context.Context, metric *RequestError)
	RequestLatencySet(ctx context.Context, metric *RequestLatency)
}

type RequestError struct {
	Url             string
	IntegrationName string
	ErrorCode       int
}
type RequestLatency struct {
	Url             string
	IntegrationName string
	LatencyMs       int64
}

type requestMetricsImpl struct {
	errorCounter *prometheus.CounterVec
	latencyGauge *prometheus.GaugeVec
}

func NewRequestMetrics(config *monitoring.Config) MetricsProvider {
	if !config.Enabled {
		return &requestMetricsMockImpl{}
	}

	errorCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: RequestMetricsErrorCounterName,
		Help: "Counts request error",
	}, []string{
		RequestMetricsLabelUrl,
		RequestMetricsLabelIntegrationName,
		RequestMetricsLabelErrorCodeName,
	})

	latencyGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: RequestMetricsLatencyMsGaugeName,
		Help: "Gauges request latency in milliseconds",
	}, []string{
		RequestMetricsLabelUrl,
		RequestMetricsLabelIntegrationName,
	})

	return &requestMetricsImpl{
		errorCounter: errorCounter,
		latencyGauge: latencyGauge,
	}
}

func (i *requestMetricsImpl) GetCollector() monitoring.MetricsCollector {
	return func() monitoring.MetricsCollection {
		return monitoring.MetricsCollection{
			i.errorCounter,
			i.latencyGauge,
		}
	}
}

func (i *requestMetricsImpl) RequestErrorInc(ctx context.Context, metric *RequestError) {
	i.errorCounter.WithLabelValues(
		strings.ToLower(metric.Url),
		strings.ToLower(metric.IntegrationName),
		strconv.Itoa(metric.ErrorCode),
	).Inc()
}

func (i *requestMetricsImpl) RequestLatencySet(ctx context.Context, metric *RequestLatency) {
	i.latencyGauge.WithLabelValues(
		strings.ToLower(metric.Url),
		strings.ToLower(metric.IntegrationName),
	).Set(float64(metric.LatencyMs))
}
