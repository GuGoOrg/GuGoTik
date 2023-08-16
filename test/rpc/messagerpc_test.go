package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/chat"
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var chatClient chat.ChatServiceClient

func setups() {
	conn, _ := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.MessageRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	chatClient = chat.NewChatServiceClient(conn)
}

func TestActionMessage_Add(t *testing.T) {
	setups()
	res, err := chatClient.ChatAction(context.Background(), &chat.ActionRequest{
		ActorId:    2,
		UserId:     2,
		ActionType: 1,
		Content:    "Test message13241234",
	})

	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)

}

func TestChat(t *testing.T) {
	setups()
	res, err := chatClient.Chat(context.Background(), &chat.ChatRequest{
		ActorId:    1,
		UserId:     1,
		PreMsgTime: 0,
	})

	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)

}
