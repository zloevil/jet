package jet

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/heptiolabs/healthcheck"
)

// HealthcheckConfig configuration for healthcheck server.
type HealthcheckConfig struct {
	Port string
}

// Check is a health check function that returns nil if healthy, or an error if unhealthy.
type Check = healthcheck.Check

// Healthcheck provides HTTP endpoints for Kubernetes liveness and readiness probes.
// Endpoints:
//   - GET /live - liveness probe (indicates if app should be restarted)
//   - GET /ready - readiness probe (indicates if app can serve traffic)
type Healthcheck struct {
	hcHandler healthcheck.Handler
	port      string
	srv       *http.Server
}

// NewHealthCheck creates a new Healthcheck instance.
func NewHealthCheck(cfg *HealthcheckConfig) *Healthcheck {
	return &Healthcheck{
		hcHandler: healthcheck.NewHandler(),
		port:      cfg.Port,
	}
}

// AddLivenessCheck adds a check that indicates whether this instance of the
// application should be destroyed or restarted.
//
// A failed liveness check indicates that this instance is unhealthy and should
// be killed and restarted by Kubernetes. Use for detecting internal failures
// like deadlocks, infinite loops, or stuck workers.
//
// Note: All liveness checks are automatically included as readiness checks.
//
// Example:
//
//	hc.AddLivenessCheck("worker", func() error {
//	    if time.Since(workerLastActivity) > 5*time.Minute {
//	        return errors.New("worker is stuck")
//	    }
//	    return nil
//	})
func (h *Healthcheck) AddLivenessCheck(name string, check Check) {
	h.hcHandler.AddLivenessCheck(name, check)
}

// AddReadinessCheck adds a check that indicates whether this instance of the
// application is currently ready to serve requests.
//
// A failed readiness check indicates that this instance should temporarily
// not receive traffic (e.g., during startup, when dependencies are unavailable).
// Kubernetes will stop routing traffic but won't restart the container.
//
// Example:
//
//	hc.AddReadinessCheck("database", func() error {
//	    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//	    defer cancel()
//	    return db.PingContext(ctx)
//	})
func (h *Healthcheck) AddReadinessCheck(name string, check Check) {
	h.hcHandler.AddReadinessCheck(name, check)
}

// Start starts the healthcheck HTTP server in a background goroutine.
// The server will automatically retry on failure with 5 second delay.
func (h *Healthcheck) Start() {
	h.srv = &http.Server{
		Addr:    fmt.Sprintf(":%s", h.port),
		Handler: h.hcHandler,
	}

	go func() {
	start:
		if err := h.srv.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				time.Sleep(time.Second * 5)
				goto start
			}
			return
		}
	}()
}

// Stop gracefully stops the healthcheck HTTP server.
func (h *Healthcheck) Stop() {
	if h.srv != nil {
		_ = h.srv.Close()
		h.srv = nil
	}
}
