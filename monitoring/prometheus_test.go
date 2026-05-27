//go:build example

package monitoring

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"math/rand"
	"net/http"
	"testing"
	"time"
)

func Test(t *testing.T) {

	var metric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test",
			Help: "test metric",
		}, []string{"path"},
	)
	err := prometheus.Register(metric)
	if err != nil {
		t.Fatal(err)
	}

	var gauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   "",
		Subsystem:   "",
		Name:        "",
		Help:        "",
		ConstLabels: nil,
	})
	gauge.Set(0)

	h := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace:   "",
		Subsystem:   "",
		Name:        "",
		Help:        "",
		ConstLabels: nil,
		Buckets:     nil,
	})
	h.Observe(0)

	lb := []string{"abc", "bcd", "edf"}

	go func() {
		for {
			i := rand.Int31n(3)
			metric.WithLabelValues(lb[i]).Inc()
			fmt.Println(lb[i])
			time.Sleep(time.Second)
		}
	}()

	router := mux.NewRouter()
	router.Path("/metrics").Handler(promhttp.Handler())
	err = http.ListenAndServe(":9000", router)
	if err != nil {
		t.Fatal(err)
	}
}
