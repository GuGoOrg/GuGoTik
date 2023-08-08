package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/comment"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

func TestActionComment_Add(t *testing.T) {
	var Client comment.CommentServiceClient
	conn, err := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.CommentRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))
	assert.Empty(t, err)

	Client = comment.NewCommentServiceClient(conn)
	res, err := Client.ActionComment(context.Background(), &comment.ActionCommentRequest{
		ActorId:    1,
		VideoId:    0,
		ActionType: comment.ActionCommentType_ACTION_COMMENT_TYPE_ADD,
		Action:     &comment.ActionCommentRequest_CommentText{CommentText: "Test comment"},
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestActionComment_Delete(t *testing.T) {

}

func TestListComment(t *testing.T) {

}

func TestCountComment(t *testing.T) {

}
