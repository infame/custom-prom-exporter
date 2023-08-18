package utilities

import (
	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// InitLogger - инициализация логгера
func InitLogger() *logrus.Logger {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return log
}

// GetLogger - возвращаем инстанс логгера
func GetLogger() *logrus.Logger {
	return log
}
