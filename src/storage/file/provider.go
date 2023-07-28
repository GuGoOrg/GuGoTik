package file

import (
	"GuGoTik/src/constant/config"
	"io"
)

var Instance storageProvider

type storageProvider interface {
	Upload(fileName string, content io.Reader) (*PutObjectOutput, error)
	GetLink(fileName string) (string, error)
}

type PutObjectOutput struct{}

func init() {
	switch config.EnvCfg.StorageType { // Append more type here to provide more file action ability
	case "fs":
		Instance = FSStorage{}
	}
}

func Upload(fileName string, content io.Reader) (*PutObjectOutput, error) {
	return Instance.Upload(fileName, content)
}

func GetLink(fileName string) (string, error) {
	return Instance.GetLink(fileName)
}
