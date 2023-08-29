package middleware

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/auth"
	grpc2 "GuGoTik/src/utils/grpc"
	"GuGoTik/src/utils/logging"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"net/http"
	"strconv"
)

var client auth.AuthServiceClient

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/douyin/user/login/" || c.Request.URL.Path == "/douyin/user/register/" {
			c.Next()
			return
		}

		var token string
		if c.Request.URL.Path == "/douyin/publish/action/" {
			token = c.PostForm("token")
		} else {
			token = c.Query("token")
		}

		if token == "" && c.Request.URL.Path == "/douyin/feed/" {
			c.Next()
			return
		}

		ctx, span := tracing.Tracer.Start(c.Request.Context(), "AuthMiddleWare")
		defer span.End()
		span.SetAttributes(attribute.String("token", token))
		logger := logging.LogService("GateWay.AuthMiddleWare").WithContext(ctx)
		// Verify User Token
		authenticate, err := client.Authenticate(c.Request.Context(), &auth.AuthenticateRequest{Token: token})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Gateway Auth meet trouble")
			span.RecordError(err)
			c.JSON(http.StatusOK, gin.H{
				"status_code": strings.GateWayErrorCode,
				"status_msg":  strings.GateWayError,
			})
			c.Abort()
			return
		}

		if authenticate.StatusCode != 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status_code": strings.AuthUserNeededCode,
				"status_msg":  strings.AuthUserNeeded,
			})
			c.Abort()
			return
		}

		c.Request.URL.RawQuery += "&actor_id=" + strconv.FormatUint(uint64(authenticate.UserId), 10)
		c.Next()
	}
}

func init() {
	authConn := grpc2.Connect(config.AuthRpcServerName)
	client = auth.NewAuthServiceClient(authConn)
}
