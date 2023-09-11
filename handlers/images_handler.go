package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"prom-exporter/types"
	"prom-exporter/utilities"
)

// ImagesHandler - структура обработчика для парсера изображений
type ImagesHandler struct {
	*AbstractHandler
	metricsMap map[string]map[string]*float64
	log        *logrus.Logger // Логгер для этого конкретного ImagesHandler
	metricType string
}

// NewImagesHandler - конструктор обработчика метрик парсера изображений
func NewImagesHandler(redisClient *redis.Client, metricsMap map[string]map[string]*float64, metricDefinition []types.MetricDefinition) *ImagesHandler {
	metricType := "images" // todo: вынести в main.go или куда-то, чтоб не прописывать каждый раз в новом хэндлере
	ah := NewAbstractHandler(redisClient, metricsMap, metricType)
	ih := &ImagesHandler{
		AbstractHandler: ah,
		metricsMap:      metricsMap,
		log:             utilities.GetLogger(),
	}

	ih.metricSaver = ih
	ih.initMetrics(metricType, metricDefinition) // Инициализация метрик для images

	return ih
}

// initMetrics - инициализация метрик в проме
func (ih *ImagesHandler) initMetrics(metricType string, metricDefinitions []types.MetricDefinition) {
	ih.log.Info("Init ImagesHandler")
	count := 0

	for _, metricDefinition := range metricDefinitions {
		for _, metricDetail := range metricDefinition.Metrics {
			if metricType == metricDefinition.Type {
				ih.registerMetric(metricDetail.Key, metricDetail.Description, *ih.metricsMap[metricDefinition.Type][metricDetail.Key])
				count++
			}
		}
	}

	ih.log.Info("Registered metrics: ", count)
}

// SetupRoutes - настройка роутов
func (ih *ImagesHandler) SetupRoutes(r *gin.Engine) {
	r.GET("/metrics/images", ih.MetricsHandler)
	r.POST("/metrics/images/:key", ih.IncrementHandler)
	r.DELETE("/metrics/images", ih.ResetHandler)
}
