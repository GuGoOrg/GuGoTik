package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/feed"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestListVideos(t *testing.T) {

	var Client feed.FeedServiceClient
	req := feed.ListFeedRequest{
		LatestTime: proto.String("2023-08-04T12:34:56.789Z"),
		ActorId:    proto.Uint32(123),
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
