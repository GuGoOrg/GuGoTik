package user

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/user"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"GuGoTik/src/web/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var userClient user.UserServiceClient

func init() {
	userConn := grpc2.Connect(config.UserRpcServerName)
	userClient = user.NewUserServiceClient(userConn)
}

func UserHandler(c *gin.Context) {
	var req models.UserReq
	_, span := tracing.Tracer.Start(c.Request.Context(), "UserInfoHandler")
	defer span.End()
	logging.SetSpanWithHostname(span)
	logger := logging.LogService("GateWay.UserInfo").WithContext(c.Request.Context())

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.UserRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
		})
		logging.SetSpanError(span, err)
		return
	}

	resp, err := userClient.GetUserInfo(c.Request.Context(), &user.UserRequest{
		UserId:  req.UserId,
		ActorId: req.ActorId,
	})

	if err != nil {
		logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when gateway get info from UserInfo Service")
		logging.SetSpanError(span, err)
		c.Render(http.StatusOK, utils.CustomJSON{Data: resp, Context: c})
		return
	}

	c.Render(http.StatusOK, utils.CustomJSON{Data: resp, Context: c})
}
