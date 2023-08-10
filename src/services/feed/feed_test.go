package main

import (
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/comment"
	"GuGoTik/src/rpc/favorite"
	"GuGoTik/src/rpc/feed"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/file"
	"GuGoTik/src/utils/logging"
	"context"
	"database/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"
)

const mockVideoCount = 50

var (
	testVideos = make([]*models.Video, mockVideoCount)
	respVideos = make([]*feed.Video, mockVideoCount)
)
var DBMock sqlmock.Sqlmock
var Conn *sql.DB

func TestMain(m *testing.M) {

	logger := logging.LogService("MockDB")
	var err error
	Conn, DBMock, err = sqlmock.New()
	_, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 Conn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		logger.Error("an error '%s' was not expected when opening a stub database connection", err)
	}

	file.Client = MockStorageProvider{}
	now := time.Now().UnixMilli()
	for i := 0; i < mockVideoCount; i++ {
		test := &models.Video{
			Model:     gorm.Model{ID: uint(i + 1), CreatedAt: time.UnixMilli(now - int64(i)*1000)},
			UserId:    int64(mockUser.Id),
			Title:     "Test Video " + strconv.Itoa(i),
			FileName:  "test_video_file_" + strconv.Itoa(i) + ".mp4",
			CoverName: "test_video_cover_file_" + strconv.Itoa(i) + ".png",
		}
		resp := &feed.Video{
			Id:            uint32(i),
			Author:        &mockUser,
			PlayUrl:       "https://test.com/test_video_file_" + strconv.Itoa(i) + ".mp4",
			CoverUrl:      "https://test.com/test_video_cover_file_" + strconv.Itoa(i) + ".png",
			FavoriteCount: 0,
			CommentCount:  0,
			IsFavorite:    false,
			Title:         "Test Video " + strconv.Itoa(i),
		}
		testVideos[i] = test
		respVideos[i] = resp
	}
	testVideos = reverseModelVideo(testVideos)
	respVideos = reverseFeedVideo(respVideos)
	code := m.Run()
	os.Exit(code)
}

func TestListVideos(t *testing.T) {
	logger := logging.LogService("TestListVideos")
	pTime := strconv.FormatInt(time.Now().Add(time.Duration(1)*time.Hour).UnixMilli(), 10)
	var successArg = struct {
		ctx context.Context
		req *feed.ListFeedRequest
	}{ctx: context.Background(), req: &feed.ListFeedRequest{
		LatestTime: &pTime,
		ActorId:    nil,
	}}

	expectedNextTime := uint32(testVideos[strings.VideoCount-1].CreatedAt.Add(time.Duration(-1)).UnixMilli())

	var successResp = &feed.ListFeedResponse{
		StatusCode: strings.ServiceOKCode,
		StatusMsg:  strings.ServiceOK,
		NextTime:   &expectedNextTime,
		VideoList:  respVideos[:strings.VideoCount],
	}

	UserClient = MockUserClient{}
	CommentClient = MockCommentClient{}
	FavoriteClient = MockFavoriteClient{}
	mockVideos := make([]*models.Video, strings.VideoCount)
	for _, v := range testVideos[:strings.VideoCount] {
		mockVideo := models.Video{
			Model:     gorm.Model{ID: v.ID, CreatedAt: v.CreatedAt},
			UserId:    v.UserId,
			Title:     v.Title,
			FileName:  v.FileName,
			CoverName: v.CoverName,
		}
		mockVideos = append(mockVideos, &mockVideo)
	}
	//实际参数
	type args struct {
		ctx context.Context
		req *feed.ListFeedRequest
	}
	tests := []struct {
		name     string
		args     args
		wantResp *feed.ListFeedResponse
		wantErr  bool
	}{
		{name: "Feed Video", args: successArg, wantResp: successResp},
	}
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "user_id", "title", "file_name", "cover_name"})
	for _, v := range testVideos[:strings.VideoCount] {
		rows.AddRow(v.ID, v.CreatedAt, nil, nil, v.UserId, v.Title, v.FileName, v.CoverName)
	}

	DBMock.
		ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "videos" WHERE "videos"."created_at" <= $1 AND "videos"."deleted_at" IS NULL ORDER BY "videos"."created_at" DESC LIMIT ` + strconv.Itoa(strings.VideoCount))).
		WillReturnRows(rows)

	//遍历测试用例
	for _, tt := range tests {
		t.Run(tt.name,
			func(t *testing.T) {
				s := &FeedServiceImpl{}
				gotResp, err := s.ListVideos(tt.args.ctx, tt.args.req)
				if (err != nil) != tt.wantErr {
					t.Errorf("ListVideos() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if len(gotResp.VideoList) != len(tt.wantResp.VideoList) {
					t.Errorf("ListVideos() lens got %v, want %v", len(gotResp.VideoList), len(tt.wantResp.VideoList))
				}
				if len(gotResp.VideoList) != strings.VideoCount {
					t.Errorf("ListVideos() lens got %v, want %v", len(gotResp.VideoList), strings.VideoCount)
				}
				if !reflect.DeepEqual(gotResp, tt.wantResp) {
					logger.Debug("gotResp: ", gotResp)
					logger.Debug("wantResp: ", tt.wantResp)
					t.Errorf("ListVideos() gotResp %v, want %v", gotResp, tt.wantResp)
				}
			})
	}
}

func reverseModelVideo(s []*models.Video) []*models.Video {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func reverseFeedVideo(s []*feed.Video) []*feed.Video {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

var mockContext = context.Background()

type MockUserClient struct {
}

var mockUser = user.User{Id: 65535}

func (m MockUserClient) GetUserInfo(ctx context.Context, in *user.UserRequest, opts ...grpc.CallOption) (*user.UserResponse, error) {
	ctx = mockContext
	in = nil
	opts = nil
	return &user.UserResponse{StatusCode: strings.ServiceOKCode, User: &mockUser}, nil
}

type MockCommentClient struct {
}

func (m MockCommentClient) ActionComment(ctx context.Context, in *comment.ActionCommentRequest, opts ...grpc.CallOption) (*comment.ActionCommentResponse, error) {
	ctx = mockContext
	in = nil
	opts = nil
	panic("unimplemented")
}

func (m MockCommentClient) ListComment(ctx context.Context, in *comment.ListCommentRequest, opts ...grpc.CallOption) (r *comment.ListCommentResponse, err error) {
	ctx = mockContext
	in = nil
	opts = nil
	return &comment.ListCommentResponse{StatusCode: strings.ServiceOKCode, StatusMsg: strings.ServiceOK, CommentList: []*comment.Comment{}}, nil
}

func (m MockCommentClient) CountComment(ctx context.Context, in *comment.CountCommentRequest, opts ...grpc.CallOption) (r *comment.CountCommentResponse, err error) {
	ctx = mockContext
	in = nil
	opts = nil
	panic("unimplemented")
}

type MockStorageProvider struct {
}

func (m MockStorageProvider) Upload(ctx context.Context, fileName string, content io.Reader) (*file.PutObjectOutput, error) {
	// Nothing to do
	ctx = mockContext
	fileName = "test.mp4"
	content = nil
	return &file.PutObjectOutput{}, nil
}

func (m MockStorageProvider) GetLink(ctx context.Context, fileName string) (string, error) {
	ctx = mockContext
	return "https://test.com/" + fileName, nil
}

type MockFavoriteClient struct {
}

func (m MockFavoriteClient) CountUserFavorite(ctx context.Context, in *favorite.CountUserFavoriteRequest, opts ...grpc.CallOption) (r *favorite.CountUserFavoriteResponse, err error) {
	ctx = mockContext
	in = nil
	opts = nil
	return &favorite.CountUserFavoriteResponse{
		Count: 0,
	}, nil
}

func (m MockFavoriteClient) CountUserTotalFavorited(ctx context.Context, in *favorite.CountUserTotalFavoritedRequest, opts ...grpc.CallOption) (r *favorite.CountUserTotalFavoritedResponse, err error) {
	ctx = mockContext
	in = nil
	opts = nil
	return &favorite.CountUserTotalFavoritedResponse{
		Count: 0,
	}, nil
}

func (m MockFavoriteClient) FavoriteAction(ctx context.Context, in *favorite.FavoriteRequest, opts ...grpc.CallOption) (*favorite.FavoriteResponse, error) {
	ctx = mockContext
	in = nil
	opts = nil
	panic("unimplemented")
}

func (m MockFavoriteClient) FavoriteList(ctx context.Context, in *favorite.FavoriteListRequest, opts ...grpc.CallOption) (r *favorite.FavoriteListResponse, err error) {
	ctx = mockContext
	in = nil
	opts = nil
	panic("unimplemented")
}

func (m MockFavoriteClient) IsFavorite(ctx context.Context, in *favorite.IsFavoriteRequest, opts ...grpc.CallOption) (r *favorite.IsFavoriteResponse, err error) {
	ctx = mockContext
	in = nil
	opts = nil
	return &favorite.IsFavoriteResponse{
		Result: false,
	}, nil
}

func (m MockFavoriteClient) CountFavorite(ctx context.Context, in *favorite.CountFavoriteRequest, opts ...grpc.CallOption) (r *favorite.CountFavoriteResponse, err error) {
	ctx = mockContext
	in = nil
	opts = nil
	return &favorite.CountFavoriteResponse{
		Count: 0,
	}, nil
}
