package grpc

import (
	"context"
	"google.golang.org/grpc"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"time"
)

type readinessAwaiter struct {
	health.HealthClient
}

func newReadinessAwaiter(c *grpc.ClientConn) readinessAwaiter {
	s := readinessAwaiter{}
	s.HealthClient = health.NewHealthClient(c)
	return s
}

func (r *readinessAwaiter) AwaitReadiness(timeout time.Duration) bool {
	timeoutElapsed := time.Now().Add(timeout)
	ctx, cancel := context.WithDeadline(context.Background(), timeoutElapsed)
	defer cancel()
	for {
		rs, err := r.HealthClient.Check(ctx, new(health.HealthCheckRequest))
		if err == nil && rs.GetStatus() == health.HealthCheckResponse_SERVING {
			return true
		}
		if time.Now().After(timeoutElapsed) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}
