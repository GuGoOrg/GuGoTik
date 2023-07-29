package logging

import "github.com/sirupsen/logrus"

type JaegerLogger struct {
	entry *logrus.Entry
}

func (l *JaegerLogger) Error(msg string) {
	l.entry.Error(msg)
}

func (l *JaegerLogger) Infof(msg string, args ...interface{}) {
	l.entry.Infof(msg, args...)
}

func GetJaegerLogger() *JaegerLogger {
	return &JaegerLogger{entry: Logger.WithFields(logrus.Fields{
		"component": "jaeger",
	})}
}
