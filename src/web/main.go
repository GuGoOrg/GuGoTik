package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/profiling"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/web/about"
	"GuGoTik/src/web/auth"
	feed2 "GuGoTik/src/web/feed"
	"GuGoTik/src/web/middleware"
	"context"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {

	// Set Trace Provider
	tp, err := tracing.SetTraceProvider(config.WebServiceName)

	if err != nil {
		logging.Logger.WithFields(logrus.Fields{
			"err": err,
		}).Panicf("Error to set the trace")
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logging.Logger.WithFields(logrus.Fields{
				"err": err,
			}).Errorf("Error to set the trace")
		}
	}()

	g := gin.Default()
	// Configure Prometheus
	p := ginprometheus.NewPrometheus("gin")
	p.Use(g)
	// Configure Gzip
	g.Use(gzip.Gzip(gzip.DefaultCompression))
	// Configure Tracing
	g.Use(otelgin.Middleware(config.WebServiceName))
	g.Use(middleware.TokenAuthMiddleware())

	// Configure Pyroscope
	profiling.InitPyroscope("GuGoTik.GateWay")

	// Register Service
	// Test Service
	g.GET("/about", about.Handle)
	// Production Service
	rootPath := g.Group("/douyin")
	user := rootPath.Group("/user")
	{
		user.POST("/login", auth.LoginHandle)
		user.POST("/register", auth.RegisterHandle)
	}
	feed := rootPath.Group("/feed")
	{
		feed.GET("/", feed2.ListVideosHandle)
	}
	// Run Server
	if err := g.Run(config.WebServiceAddr); err != nil {
		panic("Can not run GuGoTik Gateway, binding port: " + config.WebServiceAddr)
	}
}
