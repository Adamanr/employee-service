package config

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server struct {
		Host              string        `toml:"host"`
		WriteTimeout      time.Duration `toml:"write_timeout"`
		ReadTimeout       time.Duration `toml:"read_timeout"`
		ReadHeaderTimeout time.Duration `toml:"read_header_timeout"`
		JWTSecret         string        `toml:"jwt_secret"`
	} `toml:"server"`
	Database struct {
		Host     string `toml:"host"`
		User     string `toml:"user"`
		Password string `toml:"password"`
		Database string `toml:"database"`
	} `toml:"database"`
	Redis struct {
		RedisAddr          string        `toml:"redis_addr"`
		RedisPassword      string        `toml:"redis_password"`
		RedisDB            int           `toml:"redis_db"`
		AccessTokenTTL     time.Duration `toml:""`
		RefreshTokenTTL    time.Duration `toml:""`
		StrAccessTokenTTL  string        `toml:"access_token_ttl"`
		StrRefreshTokenTTL string        `toml:"refresh_token_ttl"`
	} `toml:"redis"`
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

	if cfg.Server.JWTSecret == "" {
		return nil, errors.New("jwt_secret is empty")
	}

	logger.Info("Config is loaded")
	return cfg, nil
}
