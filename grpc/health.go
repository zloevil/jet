package grpc

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"sync"
)

// HealthServer implements `service Health`.
type HealthServer struct {
	mu sync.Mutex
	// statusMap stores the serving status of the services this Server monitors.
	statusMap map[string]health.HealthCheckResponse_ServingStatus
}

// NewHealthServer creates a new health server
func NewHealthServer() *HealthServer {
	return &HealthServer{
		statusMap: make(map[string]health.HealthCheckResponse_ServingStatus),
	}
}

// Check implements `service Health`.
func (s *HealthServer) Check(ctx context.Context, in *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.Service == "" {
		// check the server overall health status.
		return &health.HealthCheckResponse{
			Status: health.HealthCheckResponse_SERVING,
		}, nil
	}
	if st, ok := s.statusMap[in.Service]; ok {
		return &health.HealthCheckResponse{
			Status: st,
		}, nil
	}
	return nil, status.Error(codes.NotFound, "unknown service")
}

func (s *HealthServer) Watch(*health.HealthCheckRequest, health.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "unimplemented")
}

// SetServingStatus is called when need to reset the serving status of a service
// or insert a new service entry into the statusMap.
func (s *HealthServer) SetServingStatus(service string, status health.HealthCheckResponse_ServingStatus) {
	s.mu.Lock()
	s.statusMap[service] = status
	s.mu.Unlock()
}

func (s *HealthServer) List(context.Context, *health.HealthListRequest) (*health.HealthListResponse, error) {
	r := &health.HealthListResponse{
		Statuses: make(map[string]*health.HealthCheckResponse),
	}
	for k, v := range s.statusMap {
		r.Statuses[k] = &health.HealthCheckResponse{
			Status: v,
		}
	}
	return r, nil
}
