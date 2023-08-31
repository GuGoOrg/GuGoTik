package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/profiling"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/user"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/utils/consul"
	"GuGoTik/src/utils/logging"
	"GuGoTik/src/utils/prom"
	"context"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/gorm/clause"
	"net"
	"net/http"
	"os"
	"syscall"
)

func main() {
	tp, err := tracing.SetTraceProvider(config.UserRpcServerName)

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

	// Configure Pyroscope
	profiling.InitPyroscope("GuGoTik.UserService")

	log := logging.LogService(config.UserRpcServerName)
	lis, err := net.Listen("tcp", config.EnvCfg.PodIpAddr+config.UserRpcServerPort)

	if err != nil {
		log.Panicf("Rpc %s listen happens error: %v", config.UserRpcServerName, err)
	}

	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)

	reg := prom.Client
	reg.MustRegister(srvMetrics)

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.ChainUnaryInterceptor(srvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(prom.ExtractContext))),
		grpc.ChainStreamInterceptor(srvMetrics.StreamServerInterceptor(grpcprom.WithExemplarFromContext(prom.ExtractContext))),
	)

	if err := consul.RegisterConsul(config.UserRpcServerName, config.UserRpcServerPort); err != nil {
		log.Panicf("Rpc %s register consul happens error for: %v", config.UserRpcServerName, err)
	}
	log.Infof("Rpc %s is running at %s now", config.UserRpcServerName, config.UserRpcServerPort)

	var srv UserServiceImpl
	user.RegisterUserServiceServer(s, srv)
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())
	srv.New()
	createMagicUser()
	srvMetrics.InitializeMetrics(s)

	g := &run.Group{}
	g.Add(func() error {
		return s.Serve(lis)
	}, func(err error) {
		s.GracefulStop()
		s.Stop()
		log.Errorf("Rpc %s listen happens error for: %v", config.UserRpcServerName, err)
	})

	httpSrv := &http.Server{Addr: config.EnvCfg.PodIpAddr + config.Metrics}
	g.Add(func() error {
		m := http.NewServeMux()
		m.Handle("/metrics", promhttp.HandlerFor(
			reg,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
			},
		))
		httpSrv.Handler = m
		log.Infof("Promethus now running")
		return httpSrv.ListenAndServe()
	}, func(error) {
		if err := httpSrv.Close(); err != nil {
			log.Errorf("Prometheus %s listen happens error for: %v", config.UserRpcServerName, err)
		}
	})

	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	if err := g.Run(); err != nil {
		log.WithFields(logrus.Fields{
			"err": err,
		}).Errorf("Error when runing http server")
		os.Exit(1)
	}
}

func createMagicUser() {
	// Create magic user: show video summary and keywords, and act as ChatGPT
	magicUser := models.User{
		UserName:        "ChatGPT",
		Password:        "chatgpt",
		Role:            2,
		Avatar:          "https://maples31-blog.oss-cn-beijing.aliyuncs.com/img/ChatGPT_logo.svg.png",
		BackgroundImage: "https://maples31-blog.oss-cn-beijing.aliyuncs.com/img/ChatGPT.jpg",
		Signature:       "GuGoTik 小助手",
	}
	result := database.Client.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"password", "role", "avatar", "background_image", "signature"}),
	}).Create(&magicUser)

	if result.Error != nil {
		logging.Logger.Errorf("Cannot create magic user because of %s", result.Error)
	}

	config.EnvCfg.MagicUserId = magicUser.ID
	logging.Logger.WithFields(logrus.Fields{
		"MagicUserId": magicUser.ID,
	}).Infof("Successfully create the magic user")
}
