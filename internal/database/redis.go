package database

import (
	"context"
	"employes_service/internal/config"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

func NewRedisConn(cfg *config.Config, logger *slog.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.RedisAddr,
		Password: cfg.Redis.RedisPassword,
		DB:       cfg.Redis.RedisDB,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		logger.Error("Failed to connect to Redis", slog.Any("error", err))
		return nil, err
	}

	logger.Info("Successfully connected to Redis")

	return rdb, nil
}
