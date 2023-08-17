package handlers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"net/http"
	"prom-exporter/providers"
	"prom-exporter/utilities"
	"sync"
)

type AbstractHandler struct {
	mutex          sync.Mutex
	metricsMap     map[string]map[string]*uint64
	metricRegistry map[string]prometheus.CounterVec
	metricType     string
	redisClient    *redis.Client
	metricSaver    MetricSaver
	log            *logrus.Logger
}

type MetricSaver interface {
	saveMetricToRedis(key string, value uint64)
}

var log = utilities.GetLogger()

func NewAbstractHandler(redisClient *redis.Client, metricsMap map[string]map[string]*uint64) *AbstractHandler {
	return &AbstractHandler{
		redisClient:    redisClient,
		metricsMap:     metricsMap,
		metricRegistry: make(map[string]prometheus.CounterVec),
	}
}

func (ah *AbstractHandler) createCounterMetric(key, help string) {
	if ah.metricsMap[ah.metricType] == nil {
		ah.metricsMap[ah.metricType] = make(map[string]*uint64)
	}
	ah.metricsMap[ah.metricType][key] = new(uint64)
}

func (ah *AbstractHandler) MetricsHandler(c *gin.Context) {
	ah.mutex.Lock()
	defer ah.mutex.Unlock()
	c.JSON(http.StatusOK, ah.metricsMap[ah.metricType])
}

func (ah *AbstractHandler) IncrementHandler(c *gin.Context) {
	metricType := ah.metricType
	key := c.Param("key")
	if metricTypeMetrics, ok := ah.metricsMap[metricType]; ok && key != "" {
		ah.mutex.Lock()
		defer ah.mutex.Unlock()

		if metric, metricExists := metricTypeMetrics[key]; metricExists {
			*metric++

			counterVec := ah.metricRegistry[key]
			counterVec.WithLabelValues(metricType).Inc()

			ah.saveMetricToRedis(key, *metric)
			c.JSON(http.StatusOK, gin.H{key: *metric})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key"})
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metric type or missing key"})
	}
}

func (ah *AbstractHandler) ResetHandler(c *gin.Context) {
	metricType := ah.metricType
	ah.mutex.Lock()
	defer ah.mutex.Unlock()

	for key := range ah.metricsMap[metricType] {
		ah.metricsMap[metricType][key] = new(uint64)
		ah.saveMetricToRedis(key, 0)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Metrics reset"})
}

func (ah *AbstractHandler) GetMetrics() map[string]*uint64 {
	ah.mutex.Lock()
	defer ah.mutex.Unlock()

	return ah.metricsMap[ah.metricType]
}

func (ah *AbstractHandler) saveMetricToRedis(key string, value uint64) {
	redisClient := providers.GetRedisClient()
	err := redisClient.Set(context.Background(), fmt.Sprintf("prometheus:%s:%s", ah.metricType, key), value, 0).Err()
	if err != nil {
		log.Error("Error saving metric to Redis: ", err)
	}
}

func (ah *AbstractHandler) registerMetric(name, help string) {
	if _, exists := ah.metricRegistry[name]; !exists {
		counterVec := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: name,
				Help: help,
			},
			[]string{"type"},
		)
		prometheus.MustRegister(counterVec)
		ah.metricRegistry[name] = *counterVec

		// Инициализируем метрику в metricsMap
		ah.createCounterMetric(name, help)
	}
}
