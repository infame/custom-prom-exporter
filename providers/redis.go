package providers

import (
	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

// InitRedisClient - инициализация Redis-клиента
func InitRedisClient(addr, password string, db int) *redis.Client {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return redisClient
}

// GetRedisClient - возвращаем инстанс Redis-клиента
func GetRedisClient() *redis.Client {
	return redisClient
}

// CloseRedisClient - закрываем соединение с Redis
func CloseRedisClient() {
	_ = redisClient.Close()
}
