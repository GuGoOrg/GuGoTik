package file

import (
	"GuGoTik/src/constant/config"
	"context"
	"fmt"
	"io"
)

var client storageProvider

type storageProvider interface {
	Upload(ctx context.Context, fileName string, content io.Reader) (*PutObjectOutput, error)
	GetLink(ctx context.Context, fileName string) (string, error)
	GetLocalPath(ctx context.Context, fileName string) string
	IsFileExist(ctx context.Context, fileName string) (bool, error)
}

type PutObjectOutput struct{}

func init() {
	switch config.EnvCfg.StorageType { // Append more type here to provide more file action ability
	case "fs":
		client = FSStorage{}
	}
}

func Upload(ctx context.Context, fileName string, content io.Reader) (*PutObjectOutput, error) {
	return client.Upload(ctx, fileName, content)
}

func GetLocalPath(ctx context.Context, fileName string) string {
	return client.GetLocalPath(ctx, fileName)
}

func GetLink(ctx context.Context, fileName string, userId uint32) (link string, err error) {
	originLink, err := client.GetLink(ctx, fileName)
	link = fmt.Sprintf("%s?user_id=%d", originLink, userId)
	return
}

func IsFileExist(ctx context.Context, fileName string) (bool, error) {
	return client.IsFileExist(ctx, fileName)
}
