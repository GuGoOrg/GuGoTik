package logging

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
)

type JaegerLogger struct {
	entry *logrus.Entry
}

func (l *JaegerLogger) Error(msg string) {
	l.entry.Error(msg)
}

func (l *JaegerLogger) Infof(msg string, args ...interface{}) {
	l.entry.Infof(msg, args...)
}

func GetJaegerLogger() *JaegerLogger {
	return &JaegerLogger{entry: Logger.WithFields(logrus.Fields{
		"component": "jaeger",
	})}
}

func SetSpanError(span opentracing.Span, err error) {
	span.LogFields(
		log.String("event", "error"),
		log.String("message", err.Error()),
	)
	span.SetTag("error", true)
}

func GetSpanLogger(span opentracing.Span, method string) *logrus.Entry {
	return logrus.WithFields(logrus.Fields{
		"operation": method,
		"trace_id":  span.Context().(jaeger.SpanContext).TraceID().String(),
		"span_id":   span.Context().(jaeger.SpanContext).SpanID().String(),
	})
}
