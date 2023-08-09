package grpc

import (
	"fmt"
	capi "github.com/hashicorp/consul/api"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Connect(service *capi.CatalogService) (conn *grpc.ClientConn, err error) {
	conn, err = grpc.Dial(fmt.Sprintf("%v:%v", service.Address, service.ServicePort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()))
	return
}
