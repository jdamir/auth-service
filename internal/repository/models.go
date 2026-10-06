package repository

import "time"

type UserType string

const (
	UserTypeReal    UserType = "REAL"
	UserTypeDigital UserType = "DIGITAL"
)

type User struct {
	ID           int64     `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	UserType     UserType  `db:"user_type"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	GroupID      string    `db:"group_id"` 
}
