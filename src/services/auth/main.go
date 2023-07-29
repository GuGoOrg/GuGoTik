package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/rpc/health"
	healthImpl "GuGoTik/src/services/health"
	"GuGoTik/src/utils/consul"
	"GuGoTik/src/utils/interceptor"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/middleware"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"net"
)

func main() {
	tracer, closer, err := middleware.NewTracer(config.AuthRpcServerName)
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
	s := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.OpentracingServerInterceptor(tracer)),
	)
	log := logging.LogService(config.AuthRpcServerName)
	lis, err := net.Listen("tcp", config.AuthRpcServerPort)

	if err != nil {
		log.Panicf("Rpc %s listen happens error: %v", config.AuthRpcServerName, err)
	}

	var srv AuthServiceImpl
	var probe healthImpl.ProbeImpl
	auth.RegisterAuthServiceServer(s, srv)
	health.RegisterHealthServer(s, &probe)
	if err := consul.RegisterConsul(config.AuthRpcServerName, config.AuthRpcServerPort); err != nil {
		log.Panicf("Rpc %s register consul hanpens error for: %v", config.AuthRpcServerName, err)
	}
	log.Infof("Rpc %s is running at %s now", config.AuthRpcServerName, config.AuthRpcServerPort)
	if err := s.Serve(lis); err != nil {
		log.Panicf("Rpc %s listen hanpens error for: %v", config.AuthRpcServerName, err)
	}
}
