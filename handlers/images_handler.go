package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"prom-exporter/utilities"
)

// ImagesHandler - структура обработчика для парсера изображений
type ImagesHandler struct {
	*AbstractHandler
	metricsMap map[string]map[string]*float64
	log        *logrus.Logger // Логгер для этого конкретного ImagesHandler
}

// NewImagesHandler - конструктор обработчика метрик парсера изображений
func NewImagesHandler(redisClient *redis.Client, metricsMap map[string]map[string]*float64) *ImagesHandler {
	metricType := "images" // todo: вынести в main.go или куда-то, чтоб не прописывать каждый раз в новом хэндлере
	ah := NewAbstractHandler(redisClient, metricsMap, metricType)
	ih := &ImagesHandler{
		AbstractHandler: ah,
		metricsMap:      metricsMap,
		log:             utilities.GetLogger(),
	}

	ih.metricSaver = ih
	ih.initMetrics() // Инициализация метрик для images

	return ih
}

// initMetrics - инициализация метрик
func (ih *ImagesHandler) initMetrics() {
	ih.log.Info("Init ImagesHandler")

	ih.registerMetric("images_uploaded_total", "Total number of images uploaded", *ih.metricsMap["images"]["images_uploaded_total"])
	ih.registerMetric("images_downloaded_total", "Total number of images downloaded", *ih.metricsMap["images"]["images_downloaded_total"])
}

// SetupRoutes - настройка роутов
func (ih *ImagesHandler) SetupRoutes(r *gin.Engine) {
	r.GET("/metrics/images", ih.MetricsHandler)
	r.POST("/metrics/images/:key", ih.IncrementHandler)
	r.DELETE("/metrics/images", ih.ResetHandler)
}
