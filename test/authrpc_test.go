package test

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/health"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

func TestHealth(t *testing.T) {
	var Client health.HealthClient
	req := health.HealthCheckRequest{}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.AuthRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = health.NewHealthClient(conn)
	check, err := Client.Check(context.Background(), &req)
	assert.Empty(t, err)
	assert.Equal(t, "SERVING", check.Status.String())
}
