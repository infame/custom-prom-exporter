package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"prom-exporter/utilities"
)

// ImagesHandler - структура обработчика для парсера изображений
type ImagesHandler struct {
	*AbstractHandler
	metricsMap map[string]map[string]*uint64
}

var log = utilities.GetLogger() // todo: подумать, как сделать удобнее - не факт, что заработает с новыми хэндлерами

// NewImagesHandler - конструктор обработчика метрик парсера изображений
func NewImagesHandler(redisClient *redis.Client, metricsMap map[string]map[string]*uint64) *ImagesHandler {
	ah := NewAbstractHandler(redisClient, metricsMap)
	ih := &ImagesHandler{
		AbstractHandler: ah,
		metricsMap:      metricsMap,
	}

	ih.metricSaver = ih
	ih.initMetrics() // Инициализация метрик для images

	return ih
}

// initMetrics - инициализация метрик
func (ih *ImagesHandler) initMetrics() {
	//log.Info(`Init ImagesHandler`)
	ih.metricsMap["images"] = make(map[string]*uint64)

	ih.registerMetric("images_uploaded_total", "Total number of images uploaded")
	ih.registerMetric("images_downloaded_total", "Total number of images downloaded")
}

// SetupRoutes - настройка роутов
func (ih *ImagesHandler) SetupRoutes(r *gin.Engine) {
	r.GET("/metrics/images", ih.MetricsHandler)
	r.POST("/metrics/images/:key", ih.IncrementHandler)
	r.DELETE("/metrics/images", ih.ResetHandler)
}
