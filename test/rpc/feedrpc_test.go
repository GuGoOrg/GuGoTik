package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/feed"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
	"time"
)

func TestListVideos(t *testing.T) {

	var Client feed.FeedServiceClient
	currentTime := time.Now().Unix()
	latestTime := fmt.Sprintf("%d", currentTime)
	actorId := uint32(1)
	req := feed.ListFeedRequest{
		LatestTime: &latestTime,
		ActorId:    &actorId,
	}

	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.FeedRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = feed.NewFeedServiceClient(conn)

	res, err := Client.ListVideos(context.Background(), &req)
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestQueryVideoExisted(t *testing.T) {

	var Client feed.FeedServiceClient
	req := feed.VideoExistRequest{
		VideoId: 1,
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.FeedRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	Client = feed.NewFeedServiceClient(conn)

	res, err := Client.QueryVideoExisted(context.Background(), &req)
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}
