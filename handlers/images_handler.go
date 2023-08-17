package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type ImagesHandler struct {
	*AbstractHandler
	metricsMap map[string]map[string]*uint64
}

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

func (ih *ImagesHandler) initMetrics() {
	ih.metricsMap["images"] = make(map[string]*uint64)

	ih.registerMetric("images_uploaded_total", "Total number of images uploaded")
	ih.registerMetric("images_downloaded_total", "Total number of images downloaded")
}

func (ih *ImagesHandler) SetupRoutes(r *gin.Engine) {
	r.GET("/metrics/images", ih.MetricsHandler)
	r.POST("/metrics/images/:key", ih.IncrementHandler)
	r.DELETE("/metrics/images", ih.ResetHandler)
}
