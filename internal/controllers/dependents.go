package controllers

import (
	"context"
	"log/slog"
	"time"

	"github.com/adamanr/employes_service/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

type Controllers struct {
	AuthController       *AuthController
	DepartmentController *DepartmentController
	EmployeeController   *EmployeeController
}

type Dependens struct {
	DB interface {
		Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
		QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
		Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	}
	Redis interface {
		Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
		Get(ctx context.Context, key string) *redis.StringCmd
		Del(ctx context.Context, keys ...string) *redis.IntCmd
	}
	Logger *slog.Logger
	Config *config.Config
}
