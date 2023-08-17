package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"os"
	"prom-exporter/handlers"
	"prom-exporter/providers"
	"prom-exporter/utilities"
	"strconv"
)

var (
	serverAddr    string
	redisAddr     string
	redisPassword string
	redisDB       int
	redisCooldown int

	redisClient *redis.Client
	log         *logrus.Logger
	metricsMap  map[string]map[string]*uint64
)

func main() {
	loadEnvVariables()

	log = utilities.InitLogger()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	redisClient = providers.InitRedisClient(redisAddr, redisPassword, redisDB)
	defer providers.CloseRedisClient()

	loadMetricsFromRedis()

	r := gin.Default()

	metricsMap := make(map[string]map[string]*uint64) // Создаем глобальную карту для метрик

	imagesHandler := handlers.NewImagesHandler(redisClient, metricsMap)
	imagesHandler.SetupRoutes(r)

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Создаем новый экземпляр cron
	c := cron.New()

	// Добавляем задачу в cron для выполнения каждые 30 секунд
	_, err := c.AddFunc("@every "+strconv.Itoa(redisCooldown)+"s", func() {
		saveAllMetricsToRedis(redisClient, metricsMap)
	})
	if err != nil {
		log.Fatal("Error adding cron job: ", err)
	}
	c.Start()

	go saveAllMetricsToRedis(redisClient, metricsMap)

	serverErr := r.Run(serverAddr)
	if serverErr != nil {
		log.Fatal("Error starting server: ", err)
	}
}

func loadEnvVariables() {
	serverAddr = getEnv("PORT", ":8200")
	redisAddr = getEnv("REDIS_DSN", "localhost:6379")
	redisPassword = getEnv("REDIS_PASSWORD", "")
	redisDB, _ = strconv.Atoi(getEnv("REDIS_DB", "0"))
	redisCooldown, _ = strconv.Atoi(getEnv("REDIS_SYNC_INTERVAL", "30"))
}

func loadMetricsFromRedis() {
	metricTypes := []string{"images", "parser", "aggregator"}

	for _, metricType := range metricTypes {
		for key := range metricsMap {
			value, err := redisClient.Get(context.Background(), fmt.Sprintf("prometheus:%s:%s", metricType, key)).Uint64()
			if err != nil {
				log.Error("Error loading metric from Redis: ", err)
				continue
			}
			metricsMap[metricType][key] = &value
		}
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func saveAllMetricsToRedis(redisClient *redis.Client, metricsMap map[string]map[string]*uint64) {
	var succeed = true
	var counter = 0
	var total = 0
	for metricType, metricTypeMap := range metricsMap {
		for key, value := range metricTypeMap {
			err := redisClient.Set(context.Background(), fmt.Sprintf("prometheus:%s:%s", metricType, key), *value, 0).Err()
			total++
			if err != nil {
				log.Error("Error saving metric to Redis: ", err)
				succeed = false
			} else {
				counter++
			}
		}
	}

	if succeed {
		log.Info("Metrics were saved to redis (" + strconv.Itoa(counter) + " of " + strconv.Itoa(total) + ")")
	}
}
