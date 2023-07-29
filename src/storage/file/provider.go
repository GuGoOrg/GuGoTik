package file

import (
	"GuGoTik/src/constant/config"
	"context"
	"github.com/opentracing/opentracing-go"
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
	span, _ := opentracing.StartSpanFromContext(ctx, "File-Upload")
	defer span.Finish()
	span.SetTag("fileName", fileName)
	return Client.Upload(ctx, fileName, content)
}

func GetLink(ctx context.Context, fileName string) (link string, err error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "File-GetLink")
	defer span.Finish()
	span.SetTag("fileName", fileName)
	link, err = Client.GetLink(ctx, fileName)
	span.SetTag("fileLink", link)
	return
}
