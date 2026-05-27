package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zloevil/jet"
)

const (
	BusinessErrorCounter = "business_error_counter"
	SystemErrorCounter   = "system_error_counter"
	PanicCounter         = "panic_counter"
)

// ErrorMonitoring manages error metrics
type ErrorMonitoring interface {
	// BusinessErrorInc increases business error counter
	BusinessErrorInc(errCode string)
	// SystemErrorInc increases system error counter
	SystemErrorInc(errCode string)
	// PanicInc increases panic counter
	PanicInc()
	// Error analyze passed error and incs proper metric
	Error(err error)
	// GetCollector returns metric collector
	GetCollector() MetricsCollector
}

type errorMonitoring struct {
	businessErrorCounter *prometheus.CounterVec
	systemErrorCounter   *prometheus.CounterVec
	panicCounter         *prometheus.CounterVec
}

func NewErrorMonitoring() ErrorMonitoring {
	m := &errorMonitoring{}

	m.businessErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: BusinessErrorCounter,
		Help: "Counts business errors",
	}, []string{"errorCode"})

	m.systemErrorCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: SystemErrorCounter,
		Help: "Counts system errors",
	}, []string{"errorCode"})

	m.panicCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: PanicCounter,
		Help: "Counts panics",
	}, []string{})

	return m
}

func (m *errorMonitoring) Error(err error) {
	// monitoring errors
	if appError, ok := jet.IsAppErr(err); ok {
		switch appError.Type() {
		case jet.ErrTypeBusiness:
			m.BusinessErrorInc(appError.Code())
		case jet.ErrTypePanic:
			m.PanicInc()
		default:
			m.SystemErrorInc(appError.Code())
		}
	} else {
		m.SystemErrorInc("unhandled")
	}
}

func (m *errorMonitoring) BusinessErrorInc(errCode string) {
	m.businessErrorCounter.WithLabelValues(errCode).Inc()
}

func (m *errorMonitoring) SystemErrorInc(errCode string) {
	m.systemErrorCounter.WithLabelValues(errCode).Inc()
}

func (m *errorMonitoring) PanicInc() {
	m.panicCounter.WithLabelValues().Inc()
}

func (m *errorMonitoring) GetCollector() MetricsCollector {
	return func() MetricsCollection {
		return MetricsCollection{
			m.businessErrorCounter,
			m.systemErrorCounter,
			m.panicCounter,
		}
	}
}
