package interceptor

import (
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/middleware"
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"strings"
)

type MDReaderWriter struct {
	metadata.MD
}

func (c MDReaderWriter) ForeachKey(handler func(key, val string) error) error {
	for k, vs := range c.MD {
		for _, v := range vs {
			if err := handler(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c MDReaderWriter) Set(key, val string) {
	key = strings.ToLower(key)
	c.MD[key] = append(c.MD[key], val)
}

func OpenTracingClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, resp interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var parentCtx opentracing.SpanContext

		if parent := opentracing.SpanFromContext(ctx); parent != nil {
			parentCtx = parent.Context()
		}

		cliSpan := middleware.GatewayTracer.StartSpan(
			method,
			opentracing.ChildOf(parentCtx),
			ext.SpanKindRPCClient,
			opentracing.Tag{Key: string(ext.Component), Value: "gRPC"},
		)
		defer cliSpan.Finish()
		log := logging.GetSpanLogger(cliSpan, "GateWay.GrpcClientInject")
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		mdWriter := MDReaderWriter{md}

		err := middleware.GatewayTracer.Inject(cliSpan.Context(), opentracing.TextMap, mdWriter)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Grpc Invoker inject to metadata err")
		}

		ctx = metadata.NewOutgoingContext(ctx, md)

		err = invoker(ctx, method, req, resp, cc, opts...)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Grpc Invoker trouble.")
		}
		return err
	}
}
