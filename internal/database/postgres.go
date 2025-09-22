package database

import (
	"context"
	"employes_service/internal/config"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

func NewConnect(config *config.Config, logger *slog.Logger) (*pgx.Conn, error) {
	var url = fmt.Sprintf("postgres://%s:%s@%s/%s",
		config.Database.User, config.Database.Password, config.Database.Host, config.Database.Database)

	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		logger.Error("Error connecting to DB", slog.String("error", err.Error()))
		return nil, err
	}

	logger.Info("Connected to DB successfully")
	return conn, err
}
