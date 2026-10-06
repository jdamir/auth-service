package server

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type contextKey string

const RequestIDKey contextKey = "x-request-id"

// NewLoggingInterceptor создает Unary-интерцептор для gRPC сервера
func NewLoggingInterceptor() grpc.UnaryServerInterceptor {
	// Инициализируем JSON логгер для удобного сбора логов в Docker/ELK
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()

		// 1. Извлекаем или генерируем x-request-id
		requestID := getOrCreateRequestID(ctx)

		// Помещаем request_id в контекст, чтобы логи внутри методов тоже могли его использовать
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Создаем логгер с предустановленным request_id для текущего запроса
		reqLogger := logger.With(
			slog.String("request_id", requestID),
			slog.String("method", info.FullMethod),
		)

		reqLogger.Info("gRPC request started")

		// 2. Выполняем сам gRPC метод
		resp, err := handler(ctx, req)

		// 3. Логируем завершение запроса
		duration := time.Since(startTime)
		if err != nil {
			reqLogger.Error("gRPC request failed",
				slog.String("error", err.Error()),
				slog.Duration("duration_ms", duration),
			)
		} else {
			reqLogger.Info("gRPC request completed successfully",
				slog.Duration("duration_ms", duration),
			)
		}

		return resp, err
	}
}

// Вспомогательная функция для работы с метаданными gRPC
func getOrCreateRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ids := md.Get("x-request-id"); len(ids) > 0 && ids[0] != "" {
			return ids[0]
		}
	}
	// Если клиент или Nginx не передали id, генерируем новый UUID
	return uuid.New().String()
}
