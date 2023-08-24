package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/user"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

func TestGetUserInfo(t *testing.T) {
	var Client user.UserServiceClient
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.UserRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = user.NewUserServiceClient(conn)
	res, err := Client.GetUserInfo(context.Background(), &user.UserRequest{
		UserId:  2,
		ActorId: 1,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestGetUserExistedInfo(t *testing.T) {
	var Client user.UserServiceClient
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.UserRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = user.NewUserServiceClient(conn)
	res, err := Client.GetUserExistInformation(context.Background(), &user.UserExistRequest{
		UserId: 1,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}
