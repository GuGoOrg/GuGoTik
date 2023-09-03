package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/profiling"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/relation"
	"GuGoTik/src/utils/audit"
	"GuGoTik/src/utils/consul"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/prom"
	"context"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"net/http"
	"os"
	"syscall"
)

var conn = &amqp.Connection{}
var channel = &amqp.Channel{}

func main() {
	tp, err := tracing.SetTraceProvider(config.RelationRpcServerName)

	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Panicf("Error to set the trace")
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logging.Logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error to set the trace")
		}
	}()

	// Configure Pyroscope
	profiling.InitPyroscope("GuGoTik.RelationService")

	log := logging.LogService(config.RelationRpcServerName)
	lis, err := net.Listen("tcp", config.EnvCfg.PodIpAddr+config.RelationRpcServerPort)

	if err != nil {
		log.Panicf("Rpc %s listen happens error: %v", config.RelationRpcServerName, err)
	}

	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)

	reg := prom.Client
	reg.MustRegister(srvMetrics)

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.ChainUnaryInterceptor(srvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(prom.ExtractContext))),
		grpc.ChainStreamInterceptor(srvMetrics.StreamServerInterceptor(grpcprom.WithExemplarFromContext(prom.ExtractContext))),
	)

	if err := consul.RegisterConsul(config.RelationRpcServerName, config.RelationRpcServerPort); err != nil {
		log.Panicf("Rpc %s register consul happens error for: %v", config.RelationRpcServerName, err)
	}
	log.Infof("Rpc %s is running at %s now", config.RelationRpcServerName, config.RelationRpcServerPort)

	var srv RelationServiceImpl
	relation.RegisterRelationServiceServer(s, srv)
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())

	srv.New()

	// Initialize the audit_exchange
	audit.DeclareAuditExchange(channel)
	defer CloseMQConn()

	srvMetrics.InitializeMetrics(s)

	g := &run.Group{}
	g.Add(func() error {
		return s.Serve(lis)
	}, func(err error) {
		s.GracefulStop()
		s.Stop()
		log.Errorf("Rpc %s listen happens error for: %v", config.RelationRpcServerName, err)
	})

	httpSrv := &http.Server{Addr: config.EnvCfg.PodIpAddr + config.Metrics}
	g.Add(func() error {
		m := http.NewServeMux()
		m.Handle("/metrics", promhttp.HandlerFor(
			reg,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
			},
		))
		httpSrv.Handler = m
		log.Infof("Promethus now running")
		return httpSrv.ListenAndServe()
	}, func(error) {
		if err := httpSrv.Close(); err != nil {
			log.Errorf("Prometheus %s listen happens error for: %v", config.RelationRpcServerName, err)
		}
	})

	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	if err := g.Run(); err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when runing http server")
		os.Exit(1)
	}
}
