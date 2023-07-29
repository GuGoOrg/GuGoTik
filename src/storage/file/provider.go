package file

import (
	"GuGoTik/src/constant/config"
	"context"
	"io"
)

var Instance storageProvider

type storageProvider interface {
	Upload(ctx context.Context, fileName string, content io.Reader) (*PutObjectOutput, error)
	GetLink(ctx context.Context, fileName string) (string, error)
}

type PutObjectOutput struct{}

func init() {
	switch config.EnvCfg.StorageType { // Append more type here to provide more file action ability
	case "fs":
		Instance = FSStorage{}
	}
}

func Upload(ctx context.Context, fileName string, content io.Reader) (*PutObjectOutput, error) {
	return Instance.Upload(ctx, fileName, content)
}

func GetLink(ctx context.Context, fileName string) (string, error) {
	return Instance.GetLink(ctx, fileName)
}
