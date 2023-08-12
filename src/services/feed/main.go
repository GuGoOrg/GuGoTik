package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/profiling"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/feed"
	"GuGoTik/src/rpc/health"
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
	tp, err := tracing.SetTraceProvider(config.FeedRpcServerName)

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
	profiling.InitPyroscope("GuGoTik.FeedService")

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
	)

	log := logging.LogService(config.FeedRpcServerName)
	lis, err := net.Listen("tcp", config.FeedRpcServerPort)

	if err != nil {
		log.Panicf("Rpc %s listen happens error: %v", config.FeedRpcServerName, err)
	}

	var srv FeedServiceImpl
	var probe healthImpl.ProbeImpl
	feed.RegisterFeedServiceServer(s, srv)
	health.RegisterHealthServer(s, &probe)
	if err := consul.RegisterConsul(config.FeedRpcServerName, config.FeedRpcServerPort); err != nil {
		log.Panicf("Rpc %s register consul hanpens error for: %v", config.FeedRpcServerName, err)
	}
	log.Infof("Rpc %s is running at %s now", config.FeedRpcServerName, config.FeedRpcServerPort)
	if err := s.Serve(lis); err != nil {
		log.Panicf("Rpc %s listen hanpens error for: %v", config.FeedRpcServerName, err)
	}
}
