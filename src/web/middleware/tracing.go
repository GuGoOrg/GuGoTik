package middleware

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/utils/logging"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	jaegerConfig "github.com/uber/jaeger-client-go/config"
	"io"
	"runtime/debug"
	"time"
)

var GatewayTracer opentracing.Tracer

func Jaeger() gin.HandlerFunc {
	return func(c *gin.Context) {
		var parentSpan opentracing.Span
		tracer, closer, err := NewTracer(config.WebServiceName)
		GatewayTracer = tracer
		if err != nil {
			logging.Logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Can not init Jaeger")
			return
		}

		defer func(closer io.Closer) {
			err := closer.Close()
			if err != nil {
				logging.Logger.WithFields(logrus.Fields{
					"err": err,
				}).Errorf("Error when close closer")
			}
		}(closer)
		spCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		if err != nil {
			parentSpan = tracer.StartSpan(c.Request.URL.Path)
			defer parentSpan.Finish()
		} else {
			parentSpan = opentracing.StartSpan(
				c.Request.URL.Path,
				opentracing.ChildOf(spCtx),
				ext.SpanKindRPCServer,
			)
			defer parentSpan.Finish()
		}

		ext.HTTPUrl.Set(parentSpan, c.Request.URL.Path)
		ext.HTTPMethod.Set(parentSpan, c.Request.Method)
		ext.SpanKind.Set(parentSpan, ext.SpanKindRPCClientEnum)
		ext.Component.Set(parentSpan, "Gin-Http")
		opentracing.Tag{Key: "http.headers.x-forwarded-for", Value: c.Request.Header.Get("X-Forwarded-For")}.Set(parentSpan)
		opentracing.Tag{Key: "http.headers.user-agent", Value: c.Request.Header.Get("User-Agent")}.Set(parentSpan)
		opentracing.Tag{Key: "http.query", Value: c.Request.RequestURI}.Set(parentSpan)
		opentracing.Tag{Key: "http.remote-ip", Value: c.RemoteIP()}.Set(parentSpan)
		opentracing.Tag{Key: "http.server.mode", Value: gin.Mode()}.Set(parentSpan)
		opentracing.Tag{Key: "request.time", Value: time.Now().Format(time.RFC3339)}.Set(parentSpan)
		opentracing.Tag{Key: "user.token", Value: c.Query("token")}.Set(parentSpan)

		c.Request = c.Request.WithContext(opentracing.ContextWithSpan(c.Request.Context(), parentSpan))
		c.Next()

		if gin.Mode() == gin.DebugMode {
			opentracing.Tag{Key: "debug.trace", Value: string(debug.Stack())}.Set(parentSpan)
			body, err := io.ReadAll(c.Request.Body)
			if err == nil {
				opentracing.Tag{Key: "http.request_body", Value: string(body)}.Set(parentSpan)
			}
		}

		ext.HTTPStatusCode.Set(parentSpan, uint16(c.Writer.Status()))
		opentracing.Tag{Key: "request.errors", Value: c.Errors.String()}.Set(parentSpan)
	}
}

func NewTracer(service string) (opentracing.Tracer, io.Closer, error) {
	cfg := jaegerConfig.Configuration{
		ServiceName: service,
		Sampler: &jaegerConfig.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegerConfig.ReporterConfig{
			LogSpans:          true,
			CollectorEndpoint: config.EnvCfg.TracingEndPoint,
		},
	}
	tracer, closer, err := cfg.NewTracer(jaegerConfig.Logger(logging.GetJaegerLogger()))
	if err != nil {
		return nil, nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return tracer, closer, nil
}
