package file

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/utils/logging"
	"io"
	"net/url"
	"os"
	"path"

	"github.com/sirupsen/logrus"
)

type FSStorage struct {
}

func (f FSStorage) Upload(fileName string, content io.Reader) (output *PutObjectOutput, err error) {
	logger := logging.LogMethod("FSStorage.Upload")
	logger = logger.WithFields(logrus.Fields{
		"file_name": fileName,
	})
	logger.Debugf("Process start")

	all, err := io.ReadAll(content)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed reading content")
		return nil, err
	}

	filePath := path.Join(config.EnvCfg.FileSystemStartPath, fileName)
	dir := path.Dir(filePath)
	err = os.MkdirAll(dir, os.FileMode(0755))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed creating directory before writing file")
		return nil, err
	}

	err = os.WriteFile(filePath, all, os.FileMode(0755))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Debug("Failed writing content to file")
		return nil, err
	}

	return &PutObjectOutput{}, nil
}

func (f FSStorage) GetLink(fileName string) (string, error) {
	return url.JoinPath(config.EnvCfg.FileSystemBaseUrl, fileName)
}
