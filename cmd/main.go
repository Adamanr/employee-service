package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	grpc_api "github.com/adamanr/employes_service/internal/api/grpc"
	pb "github.com/adamanr/employes_service/internal/api/grpc/proto"
	api "github.com/adamanr/employes_service/internal/api/http"
	"github.com/adamanr/employes_service/internal/config"
	"github.com/adamanr/employes_service/internal/controllers"
	"github.com/adamanr/employes_service/internal/database"
	logging "github.com/adamanr/employes_service/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx := context.Background()

	logger, err := logging.SetupLogger("server.log", slog.LevelInfo)
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

	deps := &controllers.Dependens{
		DB:     db,
		Config: cfg,
		Redis:  rdb,
		Logger: logger,
	}

	grpcServer := grpc.NewServer()
	grpcMyServer := grpc_api.NewServer(deps)
	pb.RegisterEmployeeServiceServer(grpcServer, grpcMyServer)

	netConfig := net.ListenConfig{}

	lis, err := netConfig.Listen(ctx, "tcp", cfg.Server.GrpcHost)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC: %v", err)
	}

	go func() {
		logger.Info("gRPC server is starting", slog.String("address", "localhost:50051"))
		if err = grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	gwMux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err = pb.RegisterEmployeeServiceHandlerFromEndpoint(
		ctx,
		gwMux,
		cfg.Server.GrpcHost,
		opts,
	); err != nil {
		log.Fatalf("Failed to register gRPC-Gateway: %v", err)
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(logging.Middleware(logger))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			httpRequestsTotal.WithLabelValues(r.URL.Path, r.Method, strconv.Itoa(ww.Status())).Inc()
		})
	})

	r.Handle("/metrics", promhttp.Handler())
	r.Mount("/api/v1", gwMux) // gRPC-Gateway

	server := api.NewServer(deps)
	r.Mount("/rest/v1", api.HandlerFromMux(server, chi.NewRouter()))

	s := &http.Server{
		Handler:           r,
		Addr:              cfg.Server.Host,
		WriteTimeout:      cfg.Server.WriteTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
	}

	logger.Info("HTTP server is starting", slog.String("address", cfg.Server.Host))
	log.Fatal(s.ListenAndServe())
}
