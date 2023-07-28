package health

import (
	"GuGoTik/src/rpc/health"
	"context"
)

type ProbeImpl struct {
	health.HealthServer
}

func (h ProbeImpl) Check(context.Context, *health.HealthCheckRequest) (*health.HealthCheckResponse, error) {
	return &health.HealthCheckResponse{Status: health.HealthCheckResponse_SERVING}, nil
}

func (h ProbeImpl) Watch(*health.HealthCheckRequest, health.Health_WatchServer) error {
	return nil
}
