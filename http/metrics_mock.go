package http

import (
	"context"
	"github.com/zloevil/jet/monitoring"
)

type requestMetricsMockImpl struct {
}

func (i *requestMetricsMockImpl) GetCollector() monitoring.MetricsCollector {
	return func() monitoring.MetricsCollection {
		return nil
	}
}

func (i *requestMetricsMockImpl) RequestErrorInc(_ context.Context, _ *RequestError) {
	// nothing to do
}

func (i *requestMetricsMockImpl) RequestLatencySet(_ context.Context, _ *RequestLatency) {
	// nothing to do
}
