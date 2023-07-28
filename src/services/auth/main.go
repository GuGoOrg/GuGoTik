package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/rpc/health"
	healthImpl "GuGoTik/src/services/health"
	"GuGoTik/src/utils/consul"
	"GuGoTik/src/utils/logging"
	"google.golang.org/grpc"
	"net"
)

func main() {
	s := grpc.NewServer()
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
