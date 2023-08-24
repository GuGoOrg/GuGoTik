package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/profiling"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/health"
	"GuGoTik/src/rpc/relation"
	healthImpl "GuGoTik/src/services/health"
	"GuGoTik/src/utils/consul"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"net"
)

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

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
	)

	log := logging.LogService(config.RelationRpcServerName)
	lis, err := net.Listen("tcp", config.EnvCfg.PodIpAddr+config.AuthRpcServerPort)

	if err != nil {
		log.Panicf("Rpc %s listen happens error: %v", config.RelationRpcServerName, err)
	}

	var srv RelationServiceImpl
	var probe healthImpl.ProbeImpl
	relation.RegisterRelationServiceServer(s, srv)
	health.RegisterHealthServer(s, &probe)
	if err := consul.RegisterConsul(config.RelationRpcServerName, config.RelationRpcServerPort); err != nil {
		log.Panicf("Rpc %s register consul happens error for: %v", config.RelationRpcServerName, err)
	}
	srv.New()
	log.Infof("Rpc %s is running at %s now", config.RelationRpcServerName, config.RelationRpcServerPort)
	if err := s.Serve(lis); err != nil {
		log.Panicf("Rpc %s listen happens error for: %v", config.RelationRpcServerName, err)
	}
}
