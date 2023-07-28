package logging

import (
	"GuGoTik/src/constant/config"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.JSONFormatter{
		PrettyPrint: true,
	})
	switch config.EnvCfg.LoggerLevel {
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN", "WARNING":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "FATAL":
		log.SetLevel(log.FatalLevel)
	}
}

var Logger = log.WithFields(log.Fields{
	"Tied": config.EnvCfg.TiedLogging,
})

func LogMethod(name string) *log.Entry {
	return Logger.WithFields(log.Fields{
		"Action": name,
	})
}

func LogService(name string) *log.Entry {
	return Logger.WithFields(log.Fields{
		"Service": name,
	})
}
