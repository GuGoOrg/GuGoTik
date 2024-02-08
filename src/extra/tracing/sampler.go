package tracing

import (
	"encoding/binary"
	"go.opentelemetry.io/otel/sdk/trace"
	trace2 "go.opentelemetry.io/otel/trace"
)

type GuGoTikSampler struct {
	traceIDUpperBound uint64
}

func GetGuGoTikSampler(fraction float64) trace.Sampler {
	if fraction >= 1 {
		return trace.AlwaysSample()
	}

	if fraction <= 0 {
		fraction = 0
	}

	return &GuGoTikSampler{
		traceIDUpperBound: uint64(fraction * (1 << 63)),
	}
}

func (cs *GuGoTikSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	psc := trace2.SpanContextFromContext(p.ParentContext)

	for _, attribute := range p.Attributes {
		if attribute.Key == "error" {
			return trace.SamplingResult{
				Decision:   trace.RecordAndSample,
				Tracestate: psc.TraceState(),
			}
		}
	}

	x := binary.BigEndian.Uint64(p.TraceID[8:16]) >> 1
	if x < cs.traceIDUpperBound {
		return trace.SamplingResult{
			Decision:   trace.RecordAndSample,
			Tracestate: psc.TraceState(),
		}
	}

	return trace.SamplingResult{
		Decision:   trace.Drop,
		Tracestate: psc.TraceState(),
	}
}

func (cs *GuGoTikSampler) Description() string {
	return "GuGoTikSampler"
}
