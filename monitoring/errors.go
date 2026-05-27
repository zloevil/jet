package monitoring

import "github.com/zloevil/jet"

const (
	ErrCodePrometheusRegisterGoMetrics      = "MON-001"
	ErrCodePrometheusRegisterProcessMetrics = "MON-002"
	ErrCodePrometheusHttpServer             = "MON-003"
	ErrCodePrometheusInvalidPort            = "MON-004"
	ErrCodePrometheusRegisterAppMetrics     = "MON-005"
)

var (
	ErrPrometheusRegisterGoMetrics = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodePrometheusRegisterGoMetrics, "").Wrap(cause).Err()
	}
	ErrPrometheusRegisterProcessMetrics = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodePrometheusRegisterProcessMetrics, "").Wrap(cause).Err()
	}
	ErrPrometheusHttpServer = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodePrometheusHttpServer, "").Wrap(cause).Err()
	}
	ErrPrometheusInvalidPort = func(port string) error {
		return jet.NewAppErrBuilder(ErrCodePrometheusInvalidPort, "invalid port").F(jet.KV{"port": port}).Err()
	}
	ErrPrometheusRegisterAppMetrics = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodePrometheusRegisterAppMetrics, "").Wrap(cause).Err()
	}
)
