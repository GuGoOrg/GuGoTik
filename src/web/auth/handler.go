package auth

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/models"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/mbobakov/grpc-consul-resolver"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
)

var Client auth.AuthServiceClient

func LoginHandle(c *gin.Context) {
	var req models.LoginReq
	log := logging.LogMethod("Login")

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusOK, models.LoginRes{
			StatusCode: strings.GateWayParamsErrorCode,
			StatusMsg:  strings.GateWayParamsError,
			UserId:     0,
			Token:      "",
		})
	}

	res, err := Client.Login(context.Background(), &auth.LoginRequest{
		Username: req.UserName,
		Password: req.Password,
	})
	if err != nil {
		log.WithFields(logrus.Fields{
			"Username": req.UserName,
		}).Warnf("Error when trying to connect with AuthService")
		c.JSON(http.StatusOK, res)
		return
	}

	log.WithFields(logrus.Fields{
		"Username": req.UserName,
		"Token":    res.Token,
		"UserId":   res.UserId,
	}).Infof("User log in")

	c.JSON(http.StatusOK, res)
}

func init() {
	conn, err := grpc.Dial(
		fmt.Sprintf("consul://%s/%s?wait=15s", config.EnvCfg.ConsulAddr, config.AuthRpcServerName),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
	)

	if err != nil {

	}
	defer conn.Close()
	Client = auth.NewAuthServiceClient(conn)
}
