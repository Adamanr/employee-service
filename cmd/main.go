package main

import (
	"employes_service/internal/api"
	"employes_service/internal/config"
	"employes_service/internal/database"
	"employes_service/internal/utils"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logger, err := utils.SetupLogger("server.log", slog.LevelInfo)
	if err != nil {
		log.Fatal("Failed to setup logger:", err)
		return
	}
	slog.SetDefault(logger)

	cfg, err := config.GetConfig(logger)
	if err != nil {
		log.Fatal("Failed to load config:", err)
		return
	}

	rdb, redisErr := database.NewRedisConn(cfg, logger)
	if redisErr != nil {
		log.Fatal("Failed to connect to Redis:", redisErr)
		return
	}

	db, dbErr := database.NewConnect(cfg, logger)
	if dbErr != nil {
		logger.Error("Failed to connect to database", slog.Any("error", dbErr))
		return
	}

	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)
	prometheus.MustRegister(httpRequestsTotal)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)

	r.Use(utils.LoggingMiddleware(logger))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			httpRequestsTotal.WithLabelValues(r.URL.Path, r.Method, fmt.Sprint(ww.Status())).Inc()
		})
	})

	r.Handle("/metrics", promhttp.Handler())

	server := api.NewServer(cfg, db, rdb, logger)
	h := api.HandlerFromMux(server, r)

	s := &http.Server{
		Handler: h,
		Addr:    cfg.Server.Host,
	}

	logger.Info("Server is starting", slog.String("address", cfg.Server.Host))
	log.Fatal(s.ListenAndServe())
}
