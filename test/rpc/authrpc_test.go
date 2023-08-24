package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/rpc/health"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func TestRegister(t *testing.T) {
	var Client auth.AuthServiceClient
	req := auth.RegisterRequest{
		Username: "epicmo12312",
		Password: "epicmo12312312",
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.AuthRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = auth.NewAuthServiceClient(conn)
	res, err := Client.Register(context.Background(), &req)
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestLogin(t *testing.T) {
	var Client auth.AuthServiceClient
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.AuthRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = auth.NewAuthServiceClient(conn)
	res, err := Client.Login(context.Background(), &auth.LoginRequest{
		Username: "epicmo",
		Password: "epicmo",
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}
