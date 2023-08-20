package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/relation"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

func TestFollow(t *testing.T) {

	var Client relation.RelationServiceClient
	req := relation.RelationActionRequest{
		UserId:  4,
		ActorId: 3,
	}

	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.RelationRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = relation.NewRelationServiceClient(conn)

	res, err := Client.Follow(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestUnfollow(t *testing.T) {
	var Client relation.RelationServiceClient
	req := relation.RelationActionRequest{
		UserId:  4,
		ActorId: 3,
	}

	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.RelationRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = relation.NewRelationServiceClient(conn)

	res, err := Client.Unfollow(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestGetFollowList(t *testing.T) {

	var Client relation.RelationServiceClient
	req := relation.FollowListRequest{
		ActorId: 1,
		UserId:  1,
	}

	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.RelationRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = relation.NewRelationServiceClient(conn)

	res, err := Client.GetFollowList(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), res.StatusCode)

}

func TestGetFollowerList(t *testing.T) {

	var Client relation.RelationServiceClient
	req := relation.FollowerListRequest{
		ActorId: 1,
		UserId:  1,
	}

	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.RelationRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = relation.NewRelationServiceClient(conn)

	res, err := Client.GetFollowerList(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestCountFollowList(t *testing.T) {
	var Client relation.RelationServiceClient
	req := relation.CountFollowListRequest{
		UserId: 1,
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.RelationRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = relation.NewRelationServiceClient(conn)

	res, err := Client.CountFollowList(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestCountFollowerList(t *testing.T) {

	var Client relation.RelationServiceClient
	req := relation.CountFollowerListRequest{
		UserId: 1,
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.RelationRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = relation.NewRelationServiceClient(conn)

	res, err := Client.CountFollowerList(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), res.StatusCode)

}

func TestIsFollow(t *testing.T) {

	var Client relation.RelationServiceClient
	req := relation.IsFollowRequest{
		ActorId: 1,
		UserId:  2,
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.RelationRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = relation.NewRelationServiceClient(conn)

	res, err := Client.IsFollow(context.Background(), &req)
	assert.NoError(t, err)
	assert.Equal(t, true, res.Result)

}

func TestGetFriendList(t *testing.T) {
	var Client relation.RelationServiceClient
	req := relation.FriendListRequest{
		ActorId: 3,
		UserId:  3,
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.RelationRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = relation.NewRelationServiceClient(conn)

	res, err := Client.GetFriendList(context.Background(), &req)
	fmt.Println(res)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), res.StatusCode)

}
