package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/profiling"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/favorite"
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
	tp, err := tracing.SetTraceProvider(config.FavoriteRpcServerName)

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
	profiling.InitPyroscope("GuGoTik.LikeService")

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
	)

	log := logging.LogService(config.FavoriteRpcServerName)

	lis, err := net.Listen("tcp", config.EnvCfg.PodIpAddr+config.FavoriteRpcServerPort)

	if err != nil {
		log.Panicf("Rpc %s listen happens error: %v", config.FavoriteRpcServerName, err)
	}

	var srv FavoriteServiceServerImpl
	var probe healthImpl.ProbeImpl

	favorite.RegisterFavoriteServiceServer(s, srv)

	health.RegisterHealthServer(s, &probe)

	if err := consul.RegisterConsul(config.FavoriteRpcServerName, config.FavoriteRpcServerPort); err != nil {
		log.Panicf("Rpc %s register consul happens error for: %v", config.FavoriteRpcServerName, err)
	}

	srv.New()
	log.Infof("Rpc %s is running at %s now", config.FavoriteRpcServerName, config.FavoriteRpcServerPort)
	if err := s.Serve(lis); err != nil {
		log.Panicf("Rpc %s listen happens error for: %v", config.FavoriteRpcServerName, err)
	}
}
