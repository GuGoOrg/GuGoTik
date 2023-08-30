package prom

import (
	"GuGoTik/src/constant/config"
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

func ExtractContext(ctx context.Context) prometheus.Labels {
	if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
		return prometheus.Labels{
			"traceID": span.TraceID().String(),
			"spanID":  span.SpanID().String(),
			"podId":   config.EnvCfg.PodIpAddr,
		}
	}
	return nil
}
