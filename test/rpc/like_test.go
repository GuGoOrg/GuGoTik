package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/favorite"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var likeClient favorite.FavoriteServiceClient

func setups1() {
	conn, _ := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.FavoriteRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	likeClient = favorite.NewFavoriteServiceClient(conn)
}
func TestFavoriteAction(t *testing.T) {
	setups1()
	res, err := likeClient.FavoriteAction(context.Background(), &favorite.FavoriteRequest{
		ActorId:    2,
		VideoId:    20,
		ActionType: 1,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestFavoriteList(t *testing.T) {
	setups1()
	res, err := likeClient.FavoriteList(context.Background(), &favorite.FavoriteListRequest{
		ActorId: 2,
		UserId:  1,
	})

	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
	assert.Nil(t, res.VideoList)
}

func TestIsFavorite(t *testing.T) {
	setups1()
	res, err := likeClient.IsFavorite(context.Background(), &favorite.IsFavoriteRequest{
		ActorId: 1,
		VideoId: 1,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
	assert.Equal(t, true, res.Result)
}

func TestCountFavorite(t *testing.T) {
	setups1()
	res, err := likeClient.CountFavorite(context.Background(), &favorite.CountFavoriteRequest{
		VideoId: 88,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
	assert.Equal(t, uint32(1), res.Count)
}

func TestCountUserFavorite(t *testing.T) {
	setups1()
	res, err := likeClient.CountUserFavorite(context.Background(), &favorite.CountUserFavoriteRequest{
		UserId: 2,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
	assert.Equal(t, uint32(1), res.Count)
}

func TestCountUserTotalFavorited(t *testing.T) {
	setups1()
	res, err := likeClient.CountUserTotalFavorited(context.Background(), &favorite.CountUserTotalFavoritedRequest{
		ActorId: 100,
		UserId:  3,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
	assert.Equal(t, uint32(0), res.Count)
}
