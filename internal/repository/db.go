package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// DB содержит пулы подключений к Postgres и Redis
type DB struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
}

// NewDB Инициализирует подключения к базам данных
func NewDB(ctx context.Context, pgDSN, redisAddr string) (*DB, error) {
	// 1. Инициализация PostgreSQL (используем пул соединений pgxpool)
	pgConfig, err := pgxpool.ParseConfig(pgDSN)
	if err != nil {
		return nil, fmt.Errorf("unable to parse postgres dsn: %w", err)
	}

	// Настройки пула (опционально)
	pgConfig.MaxConns = 25
	pgConfig.MinConns = 5
	pgConfig.MaxConnLifetime = 30 * time.Minute

	pgPool, err := pgxpool.NewWithConfig(ctx, pgConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create postgres pool: %w", err)
	}

	// Проверяем соединение с Postgres
	if err := pgPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping failed: %w", err)
	}

	// 2. Инициализация Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // если есть пароль, добавьте сюда
		DB:       0,  // дефолтная база данных
	})

	// Проверяем соединение с Redis
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &DB{
		Postgres: pgPool,
		Redis:    rdb,
	}, nil
}

// Close корректно закрывает все соединения
func (db *DB) Close() {
	if db.Postgres != nil {
		db.Postgres.Close()
	}
	if db.Redis != nil {
		db.Redis.Close()
	}
}
