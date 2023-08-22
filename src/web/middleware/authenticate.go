package middleware

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/constant/strings"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/rpc/auth"
	"GuGoTik/src/utils/logging"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"strconv"
)

var client auth.AuthServiceClient

func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/douyin/user/login" || c.Request.URL.Path == "/douyin/user/register" {
			c.Next()
			return
		}

		token := c.Query("token")
		ctx, span := tracing.Tracer.Start(c.Request.Context(), "AuthMiddleWare")
		defer span.End()
		span.SetAttributes(attribute.String("token", token))
		logger := logging.LogService("GateWay.AuthMiddleWare").WithContext(ctx)
		// Verify User Token
		authenticate, err := client.Authenticate(c.Request.Context(), &auth.AuthenticateRequest{Token: token})
		if err != nil {
			logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Gatewat Auth meet trouble")
			span.RecordError(err)
			c.JSON(http.StatusOK, gin.H{
				"status_code": strings.GateWayErrorCode,
				"status_msg":  strings.GateWayError,
			})
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
	conn, err := grpc.Dial(
		fmt.Sprintf("consul://%s/%s?wait=15s",
			config.EnvCfg.ConsulAddr,
			config.EnvCfg.ConsulAnonymityPrefix+config.AuthRpcServerName),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
	)

	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Build AuthService Cient meet trouble")
	}
	client = auth.NewAuthServiceClient(conn)
}
