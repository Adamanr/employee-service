package controllers

import (
	"log/slog"

	"github.com/adamanr/employes_service/internal/config"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type Controllers struct {
	AuthController       *AuthController
	DepartmentController *DepartmentController
	EmployeeController   *EmployeeController
}

type Dependens struct {
	DB     *pgx.Conn
	Redis  *redis.Client
	Logger *slog.Logger
	Config *config.Config
}
