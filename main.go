// Package main - prometheus exporter
package main

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
	"prom-exporter/handlers"
	"prom-exporter/helpers"
	"prom-exporter/providers"
	"prom-exporter/types"
	"prom-exporter/utilities"
	"strconv"
	"syscall"
	"time"
)

var (
	serverAddr    string
	redisAddr     string
	redisPassword string
	redisDB       int
	redisCooldown int
	ginMode       string

	redisClient *redis.Client
	log         *logrus.Logger
	metricsMap  map[string]map[string]*float64
)

// metricDefinitions - definitions array
var metricDefinitions = []types.MetricDefinition{
	{
		Type: "parser_images",
		Metrics: []types.MetricDetail{
			{Key: "cached_images_total", Description: "Total number of found images that are already stored in the bucket"},
			{Key: "successful_uploads_total", Description: "Total number of uploads of images"},
			{Key: "empty_images_total", Description: "Total number of sku without images"},
			{Key: "unhandled_errors_total", Description: "Total number of unhandled errors"},
		},
	},
	// // example
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

// main - app entry point
func main() {
	loadEnvVariables()

	// handle sigterm
	chSub := make(chan os.Signal, 1)
	signal.Notify(chSub, syscall.SIGTERM)
	signal.Notify(chSub, syscall.SIGINT)

	go func() {
		sig := <-chSub
		handleSignal(sig)
	}()

	// enable logger
	log = utilities.InitLogger()

	// connect to redis
	redisClient = providers.InitRedisClient(redisAddr, redisPassword, redisDB)
	defer providers.CloseRedisClient()

	// init http-server
	r := gin.New()
	gin.SetMode(ginMode)

	// activate middleware
	r.Use(gin.Recovery())

	// turn off standard logs
	gin.DisableConsoleColor()
	gin.DefaultWriter = io.Discard

	// format logs to ELK/EFK format
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		log.WithFields(logrus.Fields{
			"client_ip":  c.ClientIP(),
			"method":     c.Request.Method,
			"status":     c.Writer.Status(),
			"user_agent": c.Request.UserAgent(),
			"latency":    latency,
			"endpoint":   c.Request.URL.Path,
			"time":       start.Format(time.RFC3339),
		}).Info("Request processed")
	})

	metricsMap = initMetricsMap() // init global map
	loadMetricsFromRedis()

	//if metricsMap["images"]["images_uploaded_total"] == nil || metricsMap["images"]["images_downloaded_total"] == nil {
	//	log.Fatal("Err loading metrics from redis")
	//}

	// Launch handlers & bind routes
	imagesHandler := handlers.NewImagesHandler(redisClient, metricsMap, metricDefinitions)
	imagesHandler.SetupRoutes(r)

	// Create global endpoint for all metrics
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// New cron
	c := cron.New()

	// Add task to cron 30s
	_, err := c.AddFunc("@every "+strconv.Itoa(redisCooldown)+"s", func() {
		saveAllMetricsToRedis(redisClient, metricsMap)
	})
	if err != nil {
		log.Fatal("Error adding cron job: ", err)
	}
	c.Start()

	// Launch the server
	serverErr := r.Run(serverAddr)
	if serverErr != nil {
		log.Fatal("Error starting server: ", err)
	}
}

// loadEnvVariables - env load
func loadEnvVariables() {
	ginMode = getEnv("GIN_MODE", "release")
	serverAddr = ":" + getEnv("PORT", "8200")
	redisAddr = getEnv("REDIS_DSN", "localhost:6379")
	redisPassword = getEnv("REDIS_PASSWORD", "")
	redisDB, _ = strconv.Atoi(getEnv("REDIS_DB", "0"))
	redisCooldown, _ = strconv.Atoi(getEnv("REDIS_SYNC_INTERVAL", "30"))
}

// loadMetricsFromRedis - load initial values from redis
func loadMetricsFromRedis() {
	log.Info(`loading metrics from redis`)

	for _, metricDefinition := range metricDefinitions {
		metricType := metricDefinition.Type
		for _, metricDetail := range metricDefinition.Metrics {
			key := metricDetail.Key
			redisKey := helpers.GetFormattedRedisKey(metricType, key)
			metricName := helpers.GetFormattedMetricName(metricType, key)
			value, err := redisClient.Get(context.Background(), redisKey).Float64()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					log.Info("Key not found in Redis, initializing to 0: ", redisKey)
					value = 0
				} else {
					log.Error("Error loading metric from Redis: ", err)
					continue
				}
			} else {
				log.Infof("Loaded metric %s with value = %d", metricName, int(value))
			}
			valuePtr := value                       // new var
			metricsMap[metricType][key] = &valuePtr // use var addr
		}
	}
}

// getEnv - read env
func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

// saveAllMetricsToRedis - save metrics to redis, key template is prometheus:<type>:<metric_key>
func saveAllMetricsToRedis(redisClient *redis.Client, metricsMap map[string]map[string]*float64) {
	var succeed = true
	var counter = 0
	var total = 0
	for metricType, metricTypeMap := range metricsMap {
		for key, value := range metricTypeMap {
			redisKey := helpers.GetFormattedRedisKey(metricType, key)
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
