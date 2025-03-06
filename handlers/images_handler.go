package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"prom-exporter/types"
	"prom-exporter/utilities"
)

// ImagesHandler - images parser handler struct
type ImagesHandler struct {
	*AbstractHandler
	metricsMap map[string]map[string]*float64
	log        *logrus.Logger // Логгер для этого конкретного ImagesHandler
	metricType string
}

// NewImagesHandler - images parser constructor
func NewImagesHandler(redisClient *redis.Client, metricsMap map[string]map[string]*float64, metricDefinition []types.MetricDefinition) *ImagesHandler {
	metricType := "parser_images" // todo: вынести в main.go или куда-то, чтоб не прописывать каждый раз в новом хэндлере
	ah := NewAbstractHandler(redisClient, metricsMap, metricType)
	ih := &ImagesHandler{
		AbstractHandler: ah,
		metricsMap:      metricsMap,
		log:             utilities.GetLogger(),
	}

	ih.metricSaver = ih
	ih.initMetrics(metricType, metricDefinition) // init metric for images

	return ih
}

// initMetrics - init prom metrics
func (ih *ImagesHandler) initMetrics(metricType string, metricDefinitions []types.MetricDefinition) {
	ih.log.Info("Init ImagesHandler")
	count := 0

	for _, metricDefinition := range metricDefinitions {
		for _, metricDetail := range metricDefinition.Metrics {
			if metricType == metricDefinition.Type {
				ih.registerMetric(metricDefinition.Type, metricDetail.Key, metricDetail.Description, *ih.metricsMap[metricDefinition.Type][metricDetail.Key])
				count++
			}
		}
	}

	ih.log.Info("Registered metrics: ", count)
}

// SetupRoutes - set up routes
func (ih *ImagesHandler) SetupRoutes(r *gin.Engine) {
	r.GET("/parser/images", ih.MetricsHandler)
	r.POST("/parser/images", ih.IncrementHandler)
	r.DELETE("/parser/images", ih.ResetHandler)
}
