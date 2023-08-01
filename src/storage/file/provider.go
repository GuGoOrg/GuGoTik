package file

import (
	"GuGoTik/src/constant/config"
	"context"
	"io"
)

var Client storageProvider

type storageProvider interface {
	Upload(ctx context.Context, fileName string, content io.Reader) (*PutObjectOutput, error)
	GetLink(ctx context.Context, fileName string) (string, error)
}

type PutObjectOutput struct{}

func init() {
	switch config.EnvCfg.StorageType { // Append more type here to provide more file action ability
	case "fs":
		Client = FSStorage{}
	}
}

func Upload(ctx context.Context, fileName string, content io.Reader) (*PutObjectOutput, error) {
	return Client.Upload(ctx, fileName, content)
}

func GetLink(ctx context.Context, fileName string) (link string, err error) {
	return Client.GetLink(ctx, fileName)
}
