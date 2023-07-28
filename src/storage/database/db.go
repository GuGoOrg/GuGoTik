package database

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/models"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/logging/logrus"
	"gorm.io/plugin/opentelemetry/tracing"
	"time"
)

var DB *gorm.DB

func init() {
	var err error
	gormLogrus := logger.New(
		logrus.NewWriter(),
		logger.Config{
			SlowThreshold: time.Millisecond,
			Colorful:      false,
			LogLevel:      logger.Info,
		},
	)

	if DB, err = gorm.Open(
		postgres.Open(
			fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s",
				config.EnvCfg.PostgreSQLHost,
				config.EnvCfg.PostgreSQLUser,
				config.EnvCfg.PostgreSQLPassword,
				config.EnvCfg.PostgreSQLDataBase,
				config.EnvCfg.PostgreSQLPort)),
		&gorm.Config{
			PrepareStmt: true,
			Logger:      gormLogrus,
		},
	); err != nil {
		panic(err)
	}

	if err := DB.AutoMigrate(&models.User{}); err != nil {
		panic(err)
	}

	if err := DB.Use(tracing.NewPlugin()); err != nil {
		panic(err)
	}
}
