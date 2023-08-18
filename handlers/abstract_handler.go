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

// AbstractHandler - структура обработчика
type AbstractHandler struct {
	mutex          sync.Mutex
	metricsMap     map[string]map[string]*uint64
	metricRegistry map[string]prometheus.CounterVec
	metricType     string
	redisClient    *redis.Client
	metricSaver    MetricSaver
	log            *logrus.Logger
}

// MetricSaver - интерфейс для записи ключей в Redis
type MetricSaver interface {
	saveMetricToRedis(key string, value uint64)
}

// NewAbstractHandler - конструктор абстрактного обработчика
func NewAbstractHandler(redisClient *redis.Client, metricsMap map[string]map[string]*uint64) *AbstractHandler {
	return &AbstractHandler{
		redisClient:    redisClient,
		metricsMap:     metricsMap,
		metricRegistry: make(map[string]prometheus.CounterVec),
		log:            utilities.GetLogger(),
	}
}

// createCounterMetric - создание метрики и запись в обычную мапу
func (ah *AbstractHandler) createCounterMetric(key, help string) {
	if ah.metricsMap[ah.metricType] == nil {
		ah.metricsMap[ah.metricType] = make(map[string]*uint64)
	}
	ah.metricsMap[ah.metricType][key] = new(uint64)
}

// registerMetric - регистрация метрики в мапе prometheus (поддерживаются только counter)
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

// MetricsHandler - обработчик GET-запросов
func (ah *AbstractHandler) MetricsHandler(c *gin.Context) {
	ah.mutex.Lock()
	defer ah.mutex.Unlock()
	c.JSON(http.StatusOK, ah.metricsMap[ah.metricType])
}

// IncrementHandler - обработчик POST-запросов
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

			//ah.saveMetricToRedis(key, *metric) // todo? подумать, можно ли сохранять каждое изменение
			c.JSON(http.StatusOK, gin.H{key: *metric})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key"})
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metric type or missing key"})
	}
}

// ResetHandler - обработчик DELETE-запросов
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

// GetMetrics - хелпер для возврата значений из мапы
func (ah *AbstractHandler) GetMetrics() map[string]*uint64 {
	ah.mutex.Lock()
	defer ah.mutex.Unlock()

	return ah.metricsMap[ah.metricType]
}

// saveMetricsToRedis - хелпер для сохранения атомарных метрик в Redis
func (ah *AbstractHandler) saveMetricToRedis(key string, value uint64) {
	redisClient := providers.GetRedisClient()
	err := redisClient.Set(context.Background(), fmt.Sprintf("prometheus:%s:%s", ah.metricType, key), value, 0).Err()
	if err != nil {
		ah.log.Error("Error saving metric to Redis: ", err)
	}
}
