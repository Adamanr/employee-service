package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server struct {
		Host      string
		JWTSecret string `toml:"jwt_secret"`
	}
	Database struct {
		Host     string
		User     string
		Password string
		Database string
	}
	Redis struct {
		RedisAddr          string `toml:"redis_addr"`
		RedisPassword      string `toml:"redis_password"`
		RedisDB            int    `toml:"redis_db"`
		AccessTokenTTL     time.Duration
		RefreshTokenTTL    time.Duration
		StrAccessTokenTTL  string `toml:"access_token_ttl"`
		StrRefreshTokenTTL string `toml:"refresh_token_ttl"`
	}
}

func GetConfig(logger *slog.Logger) (*Config, error) {
	data, err := os.ReadFile("configs/config.toml")
	if err != nil {
		logger.Error("Error read config.toml file", slog.String("error", err.Error()))
		return nil, err
	}

	var cfg *Config

	if _, tomlErr := toml.Decode(string(data), &cfg); tomlErr != nil {
		logger.Error("Error decode config.toml file", slog.String("error", tomlErr.Error()))
		return nil, tomlErr
	}

	cfg.Redis.AccessTokenTTL, err = time.ParseDuration(cfg.Redis.StrAccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid access_token_ttl: %w", err)
	}
	cfg.Redis.RefreshTokenTTL, err = time.ParseDuration(cfg.Redis.StrRefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh_token_ttl: %w", err)
	}

	if cfg.Server.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	logger.Info("Config is loaded")
	return cfg, nil
}
