package providers

import (
	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

// InitRedisClient - init redis client
func InitRedisClient(addr, password string, db int) *redis.Client {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return redisClient
}

// GetRedisClient - return redis instanced
func GetRedisClient() *redis.Client {
	return redisClient
}

// CloseRedisClient - close db connection
func CloseRedisClient() {
	_ = redisClient.Close()
}
