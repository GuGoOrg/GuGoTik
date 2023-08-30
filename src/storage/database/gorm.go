package database

import (
	"GuGoTik/src/constant/config"
	"GuGoTik/src/utils/logging"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/opentelemetry/tracing"
	"time"
)

var Client *gorm.DB

func init() {
	var err error

	gormLogrus := logging.GetGormLogger()

	var cfg gorm.Config
	if config.EnvCfg.PostgreSQLSchema == "" {
		cfg = gorm.Config{
			PrepareStmt: true,
			Logger:      gormLogrus,
			NamingStrategy: schema.NamingStrategy{
				TablePrefix: config.EnvCfg.PostgreSQLSchema + ".",
			},
		}
	} else {
		cfg = gorm.Config{
			PrepareStmt: true,
			Logger:      gormLogrus,
		}
	}

	if Client, err = gorm.Open(
		postgres.Open(
			fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s",
				config.EnvCfg.PostgreSQLHost,
				config.EnvCfg.PostgreSQLUser,
				config.EnvCfg.PostgreSQLPassword,
				config.EnvCfg.PostgreSQLDataBase,
				config.EnvCfg.PostgreSQLPort)),
		&cfg,
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

	if err := Client.Use(tracing.NewPlugin()); err != nil {
		panic(err)
	}
}
