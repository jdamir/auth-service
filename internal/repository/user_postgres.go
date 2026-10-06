package repository

import (
	"context"
	"errors"
	"fmt" 

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// CreateUser сохраняет нового пользователя в базу данных и возвращает его ID и сгенерированный GroupID
func (r *UserRepository) CreateUser(ctx context.Context, email, passwordHash string, userType UserType) (int64, string, error) {
	// Добавили RETURNING group_id, чтобы сразу узнать UUID созданной семьи
	query := `
		INSERT INTO users (email, password_hash, user_type) 
		VALUES ($1, $2, $3) 
		RETURNING id, group_id;
	`
	var id int64
	var groupID string
	err := r.pool.QueryRow(ctx, query, email, passwordHash, userType).Scan(&id, &groupID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to insert user: %w", err)
	}

	return id, groupID, nil
}

// GetByEmail ищет активного пользователя по его email и возвращает его группу
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	// Добавили group_id в SELECT
	query := `
		SELECT id, email, password_hash, user_type, is_active, created_at, updated_at, group_id 
		FROM users 
		WHERE email = $1 AND is_active = true;
	`
	var u User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.UserType, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.GroupID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return &u, nil
}

// GetByID ищет пользователя по ID и возвращает его группу
func (r *UserRepository) GetByID(ctx context.Context, id int64) (*User, error) {
	// Добавили group_id в SELECT
	query := `
		SELECT id, email, password_hash, user_type, is_active, created_at, updated_at, group_id 
		FROM users 
		WHERE id = $1;
	`
	var u User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.UserType, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.GroupID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to fetch user by id: %w", err)
	}

	return &u, nil
}

