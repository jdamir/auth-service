package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionRepository struct {
	rdb *redis.Client
}

func NewSessionRepository(rdb *redis.Client) *SessionRepository {
	return &SessionRepository{rdb: rdb}
}

// SetSession сохраняет UUID refresh-токена, привязывая его к ID пользователя с TTL
func (r *SessionRepository) SetSession(ctx context.Context, tokenUUID string, userID int64, ttl time.Duration) error {
	key := fmt.Sprintf("refresh:%s", tokenUUID)
	return r.rdb.Set(ctx, key, userID, ttl).Err()
}

// GetUserIDBySession проверяет токен в Redis и возвращает ID пользователя
func (r *SessionRepository) GetUserIDBySession(ctx context.Context, tokenUUID string) (int64, error) {
	key := fmt.Sprintf("refresh:%s", tokenUUID)
	
	val, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, errors.New("session not found or expired")
		}
		return 0, fmt.Errorf("failed to get session from redis: %w", err)
	}

	userID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user id: %w", err)
	}

	return userID, nil
}

// DeleteSession удаляет сессию из Redis (используется для Logout и при ротации)
func (r *SessionRepository) DeleteSession(ctx context.Context, tokenUUID string) error {
	key := fmt.Sprintf("refresh:%s", tokenUUID)
	return r.rdb.Del(ctx, key).Err()
}
