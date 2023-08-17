package providers

import (
	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

func InitRedisClient(addr, password string, db int) *redis.Client {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return redisClient
}

func GetRedisClient() *redis.Client {
	return redisClient
}

func CloseRedisClient() {
	_ = redisClient.Close()
}
