package monitoring

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
	"net/http"
	"regexp"
)

type prometheusMetricsSrv struct {
	logger     jet.CLoggerFunc
	registerer *prometheus.Registry
	router     *mux.Router
	httpSrv    *http.Server
}

func NewMetricsServer(logger jet.CLoggerFunc) MetricsServer {
	srv := &prometheusMetricsSrv{
		logger: logger,
	}
	return srv
}

func (s *prometheusMetricsSrv) l() jet.CLogger {
	return s.logger().Pr("http").Cmp("prometheus")
}

func (s *prometheusMetricsSrv) Init(config *Config, metricProviders ...MetricsProvider) error {

	if match, _ := regexp.MatchString("^\\d{1,6}$", config.Port); !match {
		return ErrPrometheusInvalidPort(config.Port)
	}

	url := config.UrlPath
	if url == "" {
		url = "/metrics"
	}

	s.registerer = prometheus.NewRegistry()

	for _, pr := range metricProviders {
		for _, m := range pr.GetCollector()() {
			if err := s.registerer.Register(m); err != nil {
				return ErrPrometheusRegisterAppMetrics(err)
			}
		}
	}

	if config.GoMetrics {
		if err := s.registerer.Register(collectors.NewGoCollector()); err != nil {
			return ErrPrometheusRegisterGoMetrics(err)
		}
		if err := s.registerer.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
			return ErrPrometheusRegisterProcessMetrics(err)
		}
	}

	s.router = mux.NewRouter()
	s.router.Path(url).Handler(promhttp.HandlerFor(s.registerer, promhttp.HandlerOpts{}))

	s.httpSrv = &http.Server{
		Addr:    fmt.Sprintf(":%s", config.Port),
		Handler: s.router,
	}

	return nil

}

func (s *prometheusMetricsSrv) Listen() {
	goroutine.New().
		WithLoggerFn(s.logger).
		WithRetry(goroutine.Unrestricted).
		Go(context.Background(),
			func() {
				l := s.l().Mth("listen").F(jet.KV{"url": s.httpSrv.Addr}).Inf("listening")
				if err := s.httpSrv.ListenAndServe(); err != nil {
					if !errors.Is(err, http.ErrServerClosed) {
						l.E(ErrPrometheusHttpServer(err)).St().Err()
					} else {
						l.Dbg("server closed")
					}
				}
			},
		)
}

func (s *prometheusMetricsSrv) Close() {
	_ = s.httpSrv.Close()
	s.l().Inf("closed")
}
