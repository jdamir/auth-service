package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auth-service/internal/repository"
	"auth-service/internal/server"
	"auth-service/internal/service"
	"auth-service/pb"

	"google.golang.org/grpc"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	pgDSN := getEnv("DB_DSN", "postgres://auth_user:auth_password@localhost:5432/auth_database?sslmode=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	jwtSecret := []byte(getEnv("JWT_SECRET", "change-me-in-production-super-secret-key-2026"))
	grpcPort := getEnv("PORT", ":50051")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("Connecting to databases...")
	db, err := repository.NewDB(ctx, pgDSN, redisAddr)
	if err != nil {
		logger.Error("Failed to initialize databases", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Databases connected successfully")

	userRepo := repository.NewUserRepository(db.Postgres)
	sessionRepo := repository.NewSessionRepository(db.Redis)
	authService := service.NewAuthService(userRepo, sessionRepo, jwtSecret)
	grpcAuthServer := server.NewGRPCServer(authService)

	logger.Info("Initializing gRPC server...")
	listen, err := net.Listen("tcp", grpcPort)
	if err != nil {
		logger.Error("Failed to listen port", slog.String("port", grpcPort), slog.String("error", err.Error()))
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(server.NewLoggingInterceptor()),
	)

	pb.RegisterAuthServiceServer(grpcServer, grpcAuthServer)

	go func() {
		logger.Info("gRPC server is running", slog.String("address", "localhost"+grpcPort))
		if err := grpcServer.Serve(listen); err != nil && err != grpc.ErrServerStopped {
			logger.Error("gRPC server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	logger.Info("Shutting down gRPC server gracefully...")

	grpcServer.GracefulStop()
	logger.Info("gRPC server stopped cleanly")
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
