package rpc

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/rpc/comment"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"sync"
	"testing"
)

var Client comment.CommentServiceClient

func setup() {
	conn, _ := grpc.Dial(fmt.Sprintf("127.0.0.1%s", config.CommentRpcServerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`))

	Client = comment.NewCommentServiceClient(conn)
}

func TestActionComment_Add(t *testing.T) {
	res, err := Client.ActionComment(context.Background(), &comment.ActionCommentRequest{
		ActorId:    1,
		VideoId:    0,
		ActionType: comment.ActionCommentType_ACTION_COMMENT_TYPE_ADD,
		Action:     &comment.ActionCommentRequest_CommentText{CommentText: "I want to kill them all"},
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestActionComment_Limiter(t *testing.T) {
	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = Client.ActionComment(context.Background(), &comment.ActionCommentRequest{
				ActorId:    1,
				VideoId:    1,
				ActionType: comment.ActionCommentType_ACTION_COMMENT_TYPE_ADD,
				Action:     &comment.ActionCommentRequest_CommentText{CommentText: "富强民主文明和谐"},
			})
			_, _ = Client.ActionComment(context.Background(), &comment.ActionCommentRequest{
				ActorId:    2,
				VideoId:    1,
				ActionType: comment.ActionCommentType_ACTION_COMMENT_TYPE_ADD,
				Action:     &comment.ActionCommentRequest_CommentText{CommentText: "自由平等公正法治"},
			})
		}()
	}
	wg.Wait()
}

func TestActionComment_Delete(t *testing.T) {
	res, err := Client.ActionComment(context.Background(), &comment.ActionCommentRequest{
		ActorId:    1,
		VideoId:    0,
		ActionType: comment.ActionCommentType_ACTION_COMMENT_TYPE_DELETE,
		Action:     &comment.ActionCommentRequest_CommentId{CommentId: 1},
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestListComment(t *testing.T) {
	res, err := Client.ListComment(context.Background(), &comment.ListCommentRequest{
		ActorId: 1,
		VideoId: 0,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestCountComment(t *testing.T) {
	res, err := Client.CountComment(context.Background(), &comment.CountCommentRequest{
		ActorId: 1,
		VideoId: 0,
	})
	assert.Empty(t, err)
	assert.Equal(t, int32(0), res.StatusCode)
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}
