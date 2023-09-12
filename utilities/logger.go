package utilities

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

var log *logrus.Logger

type CustomJSONFormatter struct{}

// Format - Создаем кастомный форматтер под kibana/opensearch с сортировкой полей для восприятия в логах kubectl logs
func (f *CustomJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(time.RFC3339Nano)
	level := entry.Level.String()
	message := entry.Message

	serialized, err := json.Marshal(struct {
		Timestamp string                 `json:"@timestamp"`
		Level     string                 `json:"@level"`
		Message   string                 `json:"@message"`
		Fields    map[string]interface{} `json:"fields,omitempty"`
	}{
		Timestamp: timestamp,
		Level:     level,
		Message:   message,
		Fields:    entry.Data,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %w", err)
	}

	return append(serialized, '\n'), nil
}

// InitLogger - инициализация логгера
func InitLogger() *logrus.Logger {
	log = logrus.New()
	log.SetFormatter(&CustomJSONFormatter{})

	log.WithFields(logrus.Fields{
		"@service_name": "prom-exporter",
		"@environment":  "production",
	})
	return log
}

// GetLogger - возвращаем инстанс логгера
func GetLogger() *logrus.Logger {
	return log
}

// Пример использования дефолтного дефолтного форматтера:
// log.SetFormatter(&logrus.JSONFormatter{
//		FieldMap: logrus.FieldMap{
//			logrus.FieldKeyTime:  "@timestamp",
//			logrus.FieldKeyLevel: "@level",
//			logrus.FieldKeyMsg:   "@message",
//		},
//		TimestampFormat: time.RFC3339Nano,
//	})
