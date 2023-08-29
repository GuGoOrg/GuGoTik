package main

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/extra/profiling"
	"GuGoTik/src/extra/tracing"
	"GuGoTik/src/models"
	"GuGoTik/src/rpc/health"
	"GuGoTik/src/rpc/user"
	healthImpl "GuGoTik/src/services/health"
	"GuGoTik/src/storage/database"
	"GuGoTik/src/utils/consul"
	"GuGoTik/src/utils/logging"
	"context"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"gorm.io/gorm/clause"
	"net"
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

	s := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
	)

	log := logging.LogService(config.UserRpcServerName)
	lis, err := net.Listen("tcp", config.EnvCfg.PodIpAddr+config.UserRpcServerPort)

	if err != nil {
		log.Panicf("Rpc %s listen happens error: %v", config.UserRpcServerName, err)
	}

	var srv UserServiceImpl
	var probe healthImpl.ProbeImpl
	user.RegisterUserServiceServer(s, srv)
	health.RegisterHealthServer(s, &probe)
	if err := consul.RegisterConsul(config.UserRpcServerName, config.UserRpcServerPort); err != nil {
		log.Panicf("Rpc %s register consul hanpens error for: %v", config.UserRpcServerName, err)
	}
	srv.New()
	createMagicUser()
	log.Infof("Rpc %s is running at %s now", config.UserRpcServerName, config.UserRpcServerPort)
	if err := s.Serve(lis); err != nil {
		log.Panicf("Rpc %s listen hanpens error for: %v", config.UserRpcServerName, err)
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
