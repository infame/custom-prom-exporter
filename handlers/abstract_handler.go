package handlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"net/http"
	"prom-exporter/helpers"
	"prom-exporter/providers"
	"prom-exporter/utilities"
)

// NewAbstractHandler - конструктор абстрактного обработчика
func NewAbstractHandler(redisClient *redis.Client, metricsMap map[string]map[string]*float64, metricType string) *AbstractHandler {
	return &AbstractHandler{
		redisClient:    redisClient,
		metricsMap:     metricsMap,
		metricRegistry: make(map[string]prometheus.CounterVec),
		log:            utilities.GetLogger(),
		metricType:     metricType,
	}
}

// createCounterMetric - создание метрики и запись в обычную мапу
func (ah *AbstractHandler) createCounterMetric(key string, initValue float64) {
	ah.metricsMap[ah.metricType][key] = new(float64)
	*ah.metricsMap[ah.metricType][key] = initValue
}

// registerMetric - регистрация метрики в мапе prometheus (поддерживаются только counter)
func (ah *AbstractHandler) registerMetric(metricType, metricName, help string, initValue float64) {
	name := helpers.GetFormattedMetricName(metricType, metricName)
	if _, exists := ah.metricRegistry[name]; !exists {
		counterVec := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: name,
				Help: help,
			},
			[]string{"marketplaceCode"},
		)
		prometheus.MustRegister(counterVec)
		ah.metricRegistry[name] = *counterVec

		// Инициализируем метрику в metricsMap с сокращенным ключом
		ah.createCounterMetric(metricName, initValue)
		ah.log.Info(name, " ", help)

		//counterVec.WithLabelValues("your_label_value_here").Add(initValue)
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
	var payload MetricsPayload

	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	if metricTypeMetrics, ok := ah.metricsMap[ah.metricType]; ok {
		ah.mutex.Lock()
		defer ah.mutex.Unlock()

		for _, metricData := range payload.Metrics {
			if metricData.Key == "" {
				logrus.Warn("Empty metric key received")
				c.JSON(http.StatusBadRequest, gin.H{"error": "Empty metric key"})
				return
			}

			if metric, metricExists := metricTypeMetrics[metricData.Key]; metricExists {
				*metric += float64(metricData.Value)

				counterVec := ah.metricRegistry[helpers.GetFormattedMetricName(ah.metricType, metricData.Key)]
				counterVec.WithLabelValues(payload.MarketplaceCode).Add(float64(metricData.Value))

				//ah.saveMetricToRedis(metricData.Key, *metric) // взвешивание включения этой функции, возможно, стоит обсудить
			} else {
				ah.log.Warnf("Invalid metric key received: %s", metricData.Key)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid key"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metric type"})
	}
}

// ResetHandler - обработчик DELETE-запросов
func (ah *AbstractHandler) ResetHandler(c *gin.Context) {
	metricType := ah.metricType
	ah.mutex.Lock()
	defer ah.mutex.Unlock()

	for key := range ah.metricsMap[metricType] {
		ah.metricsMap[metricType][key] = new(float64)
		ah.saveMetricToRedis(key, 0)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Metrics reset"})
}

// GetMetrics - хелпер для возврата значений из мапы
func (ah *AbstractHandler) GetMetrics() map[string]*float64 {
	ah.mutex.Lock()
	defer ah.mutex.Unlock()

	return ah.metricsMap[ah.metricType]
}

// saveMetricToRedis - хелпер для сохранения атомарных метрик в Redis
func (ah *AbstractHandler) saveMetricToRedis(key string, value float64) {
	redisClient := providers.GetRedisClient()
	err := redisClient.Set(context.Background(), helpers.GetFormattedRedisKey(ah.metricType, key), value, 0).Err()
	if err != nil {
		ah.log.Error("Error saving metric to Redis: ", err)
	}
}
