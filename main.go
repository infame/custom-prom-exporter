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

	// Тут добавляем новые типы метрик
	metricTypes = []string{"images", "parser", "aggregator"}
)

// main - точка входа в приложение
func main() {
	loadEnvVariables()

	// Включаем логгер
	log = utilities.InitLogger()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Подключаемся к Redis
	redisClient = providers.InitRedisClient(redisAddr, redisPassword, redisDB)
	defer providers.CloseRedisClient()

	loadMetricsFromRedis()

	// Инициализируем http-server
	r := gin.Default()

	metricsMap := make(map[string]map[string]*uint64) // Создаем глобальную карту для метрик

	// Запускаем обработчики и биндим роуты
	imagesHandler := handlers.NewImagesHandler(redisClient, metricsMap)
	imagesHandler.SetupRoutes(r)

	// Создаем глобальный эндпоинт для всех метрик
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

	// Запускаем сервак
	serverErr := r.Run(serverAddr)
	if serverErr != nil {
		log.Fatal("Error starting server: ", err)
	}
}

// loadEnvVariables - загружаем переменные из .env
func loadEnvVariables() {
	serverAddr = getEnv("PORT", ":8200")
	redisAddr = getEnv("REDIS_DSN", "localhost:6379")
	redisPassword = getEnv("REDIS_PASSWORD", "")
	redisDB, _ = strconv.Atoi(getEnv("REDIS_DB", "0"))
	redisCooldown, _ = strconv.Atoi(getEnv("REDIS_SYNC_INTERVAL", "30"))
}

// loadMetricsFromRedis - загружаем первичные значения из Redis, если они есть
func loadMetricsFromRedis() {
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

// getEnv - чтение переменных окружения
func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

// saveAllMetricsToRedis - сохранение всех метрик в redis по ключам вида prometheus:<type>:<metric_key>
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
