package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/profiling"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/chat"
	"GuGoTik/src/rpc/health"
	healthImpl "GuGoTik/src/services/health"
	"GuGoTik/src/utils/consul"
	"GuGoTik/src/utils/logging"
	"context"
	"net"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func main() {
	tp, err := tracing.SetTraceProvider(config.MessageRpcServerName)

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
	profiling.InitPyroscope("GuGoTik.ChatService")

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
	)

	log := logging.LogService(config.MessageRpcServerName)

	lis, err := net.Listen("tcp", config.EnvCfg.PodIpAddr+config.AuthRpcServerPort)

	if err != nil {
		log.Panicf("Rpc %s listen happens error: %v", config.MessageRpcServerName, err)
	}

	var srv MessageServiceImpl
	var probe healthImpl.ProbeImpl

	chat.RegisterChatServiceServer(s, srv)

	health.RegisterHealthServer(s, &probe)

	if err := consul.RegisterConsul(config.MessageRpcServerName, config.MessageRpcServerPort); err != nil {
		log.Panicf("Rpc %s register consul happens error for: %v", config.MessageRpcServerName, err)
	}
	srv.New()
	log.Infof("Rpc %s is running at %s now", config.MessageRpcServerName, config.MessageRpcServerPort)
	if err := s.Serve(lis); err != nil {
		log.Panicf("Rpc %s listen happens error for: %v", config.MessageRpcServerName, err)
	}
}
