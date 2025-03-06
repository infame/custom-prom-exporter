package handlers

import (
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"sync"
)

// AbstractHandler - handler struct
type AbstractHandler struct {
	mutex          sync.Mutex
	metricsMap     map[string]map[string]*float64
	metricRegistry map[string]prometheus.CounterVec
	metricType     string
	redisClient    *redis.Client
	metricSaver    MetricSaver
	log            *logrus.Logger
}

// MetricSaver - redis keys interface [unused]
type MetricSaver interface {
	saveMetricToRedis(key string, value float64)
}

type Metric struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

type MetricsPayload struct {
	MarketplaceCode string   `json:"marketplaceCode"`
	Timestamp       string   `json:"timestamp"`
	ParserId        string   `json:"parserId"`
	Metrics         []Metric `json:"metrics"`
}
