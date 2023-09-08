package publish

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/publish"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"GuGoTik/src/web/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"mime/multipart"
	"net/http"
)

var Client publish.PublishServiceClient

func init() {
	conn := grpc2.Connect(config.PublishRpcServerName)
	Client = publish.NewPublishServiceClient(conn)
}

func ListPublishHandle(c *gin.Context) {
	_, span := tracing.Tracer.Start(c.Request.Context(), "Publish-ListHandle")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("GateWay.PublishList").WithContext(c.Request.Context())
	var req models.ListPublishReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.ListPublishRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
			VideoList:  nil,
		})
		return
	}
	logger.WithFields(logrus.Fields{
		"ActorId": req.ActorId,
		"UserId":  req.UserId,
	}).Debugf("List user video information")
	res, err := Client.ListVideo(c.Request.Context(), &publish.ListVideoRequest{
		ActorId: req.ActorId,
		UserId:  req.UserId,
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"UserId": req.UserId,
		}).Errorf("Error when trying to connect with PublishService")
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}
	userId := req.UserId
	logger.WithFields(logrus.Fields{
		"UserId": userId,
	}).Debugf("Publish List videos")

	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}

func paramValidate(c *gin.Context) (err error) {
	var wrappedError error
	form, err := c.MultipartForm()
	if err != nil {
		wrappedError = fmt.Errorf("invalid form: %w", err)
	}
	title := form.Value["title"]
	if len(title) <= 0 {
		wrappedError = fmt.Errorf("not title")
	}

	data := form.File["data"]
	if len(data) <= 0 {
		wrappedError = fmt.Errorf("not data")
	}
	if wrappedError != nil {
		return wrappedError
	}
	return nil
}

func ActionPublishHandle(c *gin.Context) {
	_, span := tracing.Tracer.Start(c.Request.Context(), "Publish-ActionHandle")
	defer span.End()
	logger := logging.LogService("GateWay.PublishAction").WithContext(c.Request.Context())

	if err := paramValidate(c); err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Param Validate failed")
		c.JSON(http.StatusOK, models.ActionPublishRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}

	form, _ := c.MultipartForm()
	title := form.Value["title"][0]
	file := form.File["data"][0]
	opened, _ := file.Open()
	defer func(opened multipart.File) {
		err := opened.Close()
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("opened.Close() failed")
		}
	}(opened)

	if file.Size > config.MaxVideoSize {
		logger.WithFields(logrus.Fields{
			"FileSize": file.Size,
		}).Errorf("Maximum file size is 200MB")
		c.JSON(http.StatusOK, models.ActionPublishRes{
			StatusCode: strings.OversizeVideoCode,
			StatusMsg:  strings.OversizeVideo,
		})
		return
	}

	var data = make([]byte, file.Size)
	readSize, err := opened.Read(data)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("opened.Read(data) failed")
		c.JSON(http.StatusOK, models.ActionPublishRes{
			StatusCode: strings.GateWayErrorCode,
			StatusMsg:  strings.GateWayError,
		})
		return
	}
	if readSize != int(file.Size) {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("file.Size != readSize")
		c.JSON(http.StatusOK, models.ActionPublishRes{
			StatusCode: strings.GateWayErrorCode,
			StatusMsg:  strings.GateWayError,
		})
		return
	}
	var req models.ActionPublishReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.ActionPublishRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		return
	}
	logger.WithFields(logrus.Fields{
		"actorId":  req.ActorId,
		"title":    title,
		"dataSize": len(data),
	}).Debugf("Executing create video")
	res, err := Client.CreateVideo(c.Request.Context(), &publish.CreateVideoRequest{
		ActorId: req.ActorId,
		Data:    data,
		Title:   title,
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Warnf("Error when trying to connect with CreateVideoService")
		c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
		return
	}
	logger.WithFields(logrus.Fields{
		"response": res,
	}).Debugf("Create video success")
	c.Render(http.StatusOK, utils.CustomJSON{Data: res, Context: c})
}
