// Package main - экспортер для prometheus
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
	"os/signal"
	"prom-exporter/handlers"
	"prom-exporter/providers"
	"prom-exporter/types"
	"prom-exporter/utilities"
	"strconv"
	"syscall"
)

var (
	serverAddr    string
	redisAddr     string
	redisPassword string
	redisDB       int
	redisCooldown int

	redisClient *redis.Client
	log         *logrus.Logger
	metricsMap  map[string]map[string]*float64
)

// metricDefinitions - сам массив определений
var metricDefinitions = []types.MetricDefinition{
	{
		Type: "images",
		Metrics: []types.MetricDetail{
			{Key: "already_stored_total", Description: "Total number of found images that are already stored in the bucket"},
			{Key: "upload_succeed_total", Description: "Total number of uploads of images"},
			{Key: "unhandled_errors_total", Description: "Total number of unhandled errors"},
			//{Key: "uploaded_total", Description: "Total number of new uploaded images"},
		},
	},
	// метрики ниже -- для примера
	//{
	//	Type: "playwright",
	//	Metrics: []types.MetricDetail{
	//		{Key: "pw_metric1", Description: "Description for playwright parser metric 1"},
	//		{Key: "pw_metric2", Description: "Description for playwright parser metric 2"},
	//	},
	//},
	//{
	//	Type: "mobile",
	//	Metrics: []types.MetricDetail{
	//		{Key: "mob_metric1", Description: "Description for mobile parser metric 1"},
	//		{Key: "mob_metric2", Description: "Description for mobile parser metric 2"},
	//	},
	//},
	//{
	//	Type: "aggregator",
	//	Metrics: []types.MetricDetail{
	//		{Key: "agg_metric1", Description: "Description for aggregator metric 1"},
	//		{Key: "agg_metric2", Description: "Description for aggregator metric 2"},
	//	},
	//},
}

func initMetricsMap() map[string]map[string]*float64 {
	metricsMap := make(map[string]map[string]*float64)

	for _, metricDefinition := range metricDefinitions {
		metricsMap[metricDefinition.Type] = make(map[string]*float64)
		for _, metricDetail := range metricDefinition.Metrics {
			metricsMap[metricDefinition.Type][metricDetail.Key] = new(float64)
		}
	}

	return metricsMap
}

// main - точка входа в приложение
func main() {
	loadEnvVariables()

	// Обрабатываем sigterm
	chSub := make(chan os.Signal, 1)
	signal.Notify(chSub, syscall.SIGTERM)
	signal.Notify(chSub, syscall.SIGINT)

	go func() {
		sig := <-chSub
		handleSignal(sig)
	}()

	// Включаем логгер
	log = utilities.InitLogger()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Подключаемся к Redis
	redisClient = providers.InitRedisClient(redisAddr, redisPassword, redisDB)
	defer providers.CloseRedisClient()

	// Инициализируем http-server
	r := gin.Default()

	metricsMap = initMetricsMap() // Инициализируем глобальную карту для метрик
	loadMetricsFromRedis()

	//if metricsMap["images"]["images_uploaded_total"] == nil || metricsMap["images"]["images_downloaded_total"] == nil {
	//	log.Fatal("Ошибка загрузки метрик из Redis")
	//}

	// Запускаем обработчики и биндим роуты
	imagesHandler := handlers.NewImagesHandler(redisClient, metricsMap, metricDefinitions)
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
	serverAddr = ":" + getEnv("PORT", "8200")
	redisAddr = getEnv("REDIS_DSN", "localhost:6379")
	redisPassword = getEnv("REDIS_PASSWORD", "")
	redisDB, _ = strconv.Atoi(getEnv("REDIS_DB", "0"))
	redisCooldown, _ = strconv.Atoi(getEnv("REDIS_SYNC_INTERVAL", "30"))
}

// loadMetricsFromRedis - загружаем первичные значения из Redis, если они есть
func loadMetricsFromRedis() {
	log.Info(`loading metrics from redis`)

	for _, metricDefinition := range metricDefinitions {
		metricType := metricDefinition.Type
		for _, metricDetail := range metricDefinition.Metrics {
			key := metricDetail.Key
			redisKey := fmt.Sprintf("prometheus:parser_%s_%s", metricType, key)
			value, err := redisClient.Get(context.Background(), redisKey).Float64()
			if err != nil {
				if err == redis.Nil {
					log.Info("Key not found in Redis, initializing to 0: ", redisKey)
					value = 0
				} else {
					log.Error("Error loading metric from Redis: ", err)
					continue
				}
			}
			valuePtr := value                       // Создаём новую переменную
			metricsMap[metricType][key] = &valuePtr // Используем адрес новой переменной
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
func saveAllMetricsToRedis(redisClient *redis.Client, metricsMap map[string]map[string]*float64) {
	var succeed = true
	var counter = 0
	var total = 0
	for metricType, metricTypeMap := range metricsMap {
		for key, value := range metricTypeMap {
			redisKey := fmt.Sprintf("prometheus:parser_%s_%s", metricType, key)
			err := redisClient.Set(context.Background(), redisKey, *value, 0).Err()
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

func handleSignal(sig os.Signal) {
	log.Warnf("Got sig: %v, terminating process...\n", sig)
	saveAllMetricsToRedis(redisClient, metricsMap)
	log.Info("Exiting...")
	os.Exit(0)
}
