package logging

import (
	"context"
	"errors"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"time"
)

var errRecordNotFound = errors.New("record not found")

type GormLogger struct {
	entry *logrus.Entry
}

func (g GormLogger) LogMode(_ logger.LogLevel) logger.Interface {
	// We do not use this because Gorm will print different log according to log set.
	// However, we just print to TRACE.
	return g
}

func (g GormLogger) Info(ctx context.Context, s string, i ...interface{}) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		g.entry.WithFields(logrus.Fields{
			"trace_id": span.Context().(jaeger.SpanContext).TraceID().String(),
			"span_id":  span.Context().(jaeger.SpanContext).SpanID().String(),
		}).Infof(s, i...)
	} else {
		g.entry.Infof(s, i...)
	}
}

func (g GormLogger) Warn(ctx context.Context, s string, i ...interface{}) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		g.entry.WithFields(logrus.Fields{
			"trace_id": span.Context().(jaeger.SpanContext).TraceID().String(),
			"span_id":  span.Context().(jaeger.SpanContext).SpanID().String(),
		}).Infof(s, i...)
	} else {
		g.entry.Warnf(s, i...)
	}
}

func (g GormLogger) Error(ctx context.Context, s string, i ...interface{}) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		g.entry.WithFields(logrus.Fields{
			"trace_id": span.Context().(jaeger.SpanContext).TraceID().String(),
			"span_id":  span.Context().(jaeger.SpanContext).SpanID().String(),
		}).Infof(s, i...)
	} else {
		g.entry.Errorf(s, i...)
	}
}

func (g GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	const traceStr = "File: %s, Cost: %v, Rows: %v, SQL: %s"
	elapsed := time.Since(begin)
	sql, rows := fc()
	span := opentracing.SpanFromContext(ctx)
	localLog := g.entry
	if err != nil && !errors.Is(err, errRecordNotFound) {
		localLog = localLog.WithFields(logrus.Fields{
			"err": err,
		})
		SetSpanError(span, err)
	}

	if span != nil {
		localLog = localLog.WithFields(logrus.Fields{
			"trace_id": span.Context().(jaeger.SpanContext).TraceID().String(),
			"span_id":  span.Context().(jaeger.SpanContext).SpanID().String(),
		})
	}

	if rows == -1 {
		localLog.Tracef(traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
	} else {
		localLog.Tracef(traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
	}
}

func GetGormLogger() *GormLogger {
	return &GormLogger{entry: Logger.WithFields(logrus.Fields{
		"component": "gorm",
	})}
}
