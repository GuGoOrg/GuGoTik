package logging

import (
	"GuGoTik/src/constant/config"
	"fmt"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"io"
	"os"
	"path"
)

var hostname string

func init() {
	hostname, _ = os.Hostname()

	switch config.EnvCfg.LoggerLevel {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN", "WARNING":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "FATAL":
		log.SetLevel(log.FatalLevel)
	case "TRACE":
		log.SetLevel(log.TraceLevel)
	}

	filePath := path.Join("/var", "log", "gugotik", "gugotik.log")
	dir := path.Dir(filePath)
	if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
		panic(err)
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}

	log.SetFormatter(&log.JSONFormatter{})
	log.AddHook(logTraceHook{})
	log.SetOutput(io.MultiWriter(f, os.Stdout))

	Logger = log.WithFields(log.Fields{
		"Tied":     config.EnvCfg.TiedLogging,
		"Hostname": hostname,
		"PodIP":    config.EnvCfg.PodIpAddr,
	})
}

type logTraceHook struct{}

func (t logTraceHook) Levels() []log.Level { return log.AllLevels }

func (t logTraceHook) Fire(entry *log.Entry) error {
	ctx := entry.Context
	if ctx == nil {
		return nil
	}

	span := trace.SpanFromContext(ctx)
	//if !span.IsRecording() {
	//	return nil
	//}

	sCtx := span.SpanContext()
	if sCtx.HasTraceID() {
		entry.Data["trace_id"] = sCtx.TraceID().String()
	}
	if sCtx.HasSpanID() {
		entry.Data["span_id"] = sCtx.SpanID().String()
	}

	if config.EnvCfg.LoggerWithTraceState == "enable" {
		attrs := make([]attribute.KeyValue, 0)
		logSeverityKey := attribute.Key("log.severity")
		logMessageKey := attribute.Key("log.message")
		attrs = append(attrs, logSeverityKey.String(entry.Level.String()))
		attrs = append(attrs, logMessageKey.String(entry.Message))
		for key, value := range entry.Data {
			fields := attribute.Key(fmt.Sprintf("log.fields.%s", key))
			attrs = append(attrs, fields.String(fmt.Sprintf("%v", value)))
		}
		span.AddEvent("log", trace.WithAttributes(attrs...))
		if entry.Level <= log.ErrorLevel {
			span.SetStatus(codes.Error, entry.Message)
		}
	}
	return nil
}

var Logger *log.Entry

func LogService(name string) *log.Entry {
	return Logger.WithFields(log.Fields{
		"Service": name,
	})
}

func SetSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, "Internal Error")
}

func SetSpanErrorWithDesc(span trace.Span, err error, desc string) {
	span.RecordError(err)
	span.SetStatus(codes.Error, desc)
}

func SetSpanWithHostname(span trace.Span) {
	span.SetAttributes(attribute.String("hostname", hostname))
	span.SetAttributes(attribute.String("podIP", config.EnvCfg.PodIpAddr))
}
