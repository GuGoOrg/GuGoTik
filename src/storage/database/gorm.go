package database

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/models"
	"GuGoTik/src/utils/logging"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
	"time"
)

var Client *gorm.DB

func init() {
	var err error

	gormLogrus := logging.GetGormLogger()

	if Client, err = gorm.Open(
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

	sqlDB, err := Client.DB()
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := Client.AutoMigrate(&models.User{}); err != nil {
		panic(err)
	}

	if err := Client.Use(tracing.NewPlugin()); err != nil {
		panic(err)
	}
}
