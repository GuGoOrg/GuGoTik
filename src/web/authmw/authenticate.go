package authmw

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/utils/interceptor"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/trace"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
)

var client auth.AuthServiceClient

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/douyin/user/login" || c.Request.URL.Path == "/douyin/user/register" {
			c.Next()
			return
		}

		token := c.Query("token")
		span := trace.GetChildSpanFromGinContext(c, "GateWay-Auth")
		defer span.Finish()
		log := logging.GetSpanLogger(span, "GateWay.Auth")
		authenticate, err := client.Authenticate(c.Request.Context(), &auth.AuthenticateRequest{Token: token})

		if err != nil {
			log.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Gatewat Auth meet trouble")

			c.JSON(http.StatusOK, gin.H{
				"status_code": strings.GateWayErrorCode,
				"status_msg":  strings.GateWayError,
			})
			return
		}

		if authenticate.StatusCode != uint32(0) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status_code": strings.AuthUserNeededCode,
				"status_msg":  strings.AuthUserNeeded,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func init() {
	conn, err := grpc.Dial(
		fmt.Sprintf("consul://%s/%s?wait=15s", config.EnvCfg.ConsulAddr, config.AuthRpcServerName),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithUnaryInterceptor(interceptor.OpenTracingClientInterceptor()),
	)

	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Build AuthService Cient meet trouble")
	}
	client = auth.NewAuthServiceClient(conn)
}
