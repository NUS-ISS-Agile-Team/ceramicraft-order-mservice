package redis

import (
	"fmt"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/config"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func Init() {
	addr := fmt.Sprintf("%s:%d", config.Config.RedisConfig.Host, config.Config.RedisConfig.Port)
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
		PoolSize: 20,
	})
}
