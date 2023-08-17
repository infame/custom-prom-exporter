package utilities

import (
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

func InitLogger() *logrus.Logger {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return log
}

func GetLogger() *logrus.Logger {
	return log
}
