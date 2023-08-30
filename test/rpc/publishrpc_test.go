package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/publish"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"os"
	"sync"
	"testing"
)

var publishClient publish.PublishServiceClient

func TestListVideo(t *testing.T) {
	req := publish.ListVideoRequest{
		UserId:  123,
		ActorId: 123,
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.PublishRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	publishClient = publish.NewPublishServiceClient(conn)
	//调用服务端方法
	res, err := publishClient.ListVideo(context.Background(), &req)
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestCountVideo(t *testing.T) {
	req := publish.CountVideoRequest{
		UserId: 1,
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.PublishRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	publishClient = publish.NewPublishServiceClient(conn)
	res, err := publishClient.CountVideo(context.Background(), &req)
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestPublishVideo(t *testing.T) {
	reader, err := os.Open("/home/yangfeng/Repos/youthcamp/videos/upload_video_2_1080p.mp4")
	assert.Empty(t, err)
	bytes, err := io.ReadAll(reader)
	assert.Empty(t, err)
	req := publish.CreateVideoRequest{
		ActorId: 2,
		Data:    bytes,
		Title:   "原神，启动！",
	}
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.PublishRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	publishClient = publish.NewPublishServiceClient(conn)
	res, err := publishClient.CreateVideo(context.Background(), &req)
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestPublishVideo_Limiter(t *testing.T) {
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.PublishRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)
	publishClient = publish.NewPublishServiceClient(conn)

	reader, err := os.Open("/home/yangfeng/Repos/youthcamp/videos/upload_video_4.mp4")
	assert.Empty(t, err)
	bytes, err := io.ReadAll(reader)
	assert.Empty(t, err)
	req := publish.CreateVideoRequest{
		ActorId: 1,
		Data:    bytes,
		Title:   "原神，启动！",
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = publishClient.CreateVideo(context.Background(), &req)
		}()
	}
}
