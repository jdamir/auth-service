package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"auth-service/internal/repository"

	
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	
)


type UserRepo interface {
	// Исправлено: теперь метод возвращает ID пользователя и UUID его группы
	CreateUser(ctx context.Context, email, passwordHash string, userType repository.UserType) (int64, string, error)
	GetByEmail(ctx context.Context, email string) (*repository.User, error)
	GetByID(ctx context.Context, id int64) (*repository.User, error)
}

type SessionRepo interface {
	SetSession(ctx context.Context, tokenUUID string, userID int64, ttl time.Duration) error
	GetUserIDBySession(ctx context.Context, tokenUUID string) (int64, error)
	DeleteSession(ctx context.Context, tokenUUID string) error
}

type AuthService struct {
	userRepo        UserRepo
	sessionRepo     SessionRepo
	jwtSecret       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthService(ur UserRepo, sr SessionRepo, jwtSecret []byte) *AuthService {
	return &AuthService{
		userRepo:        ur,
		sessionRepo:     sr,
		jwtSecret:       jwtSecret,
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 30 * 24 * time.Hour,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string, userType string) (int64, error) {
	// ВНЕДРЕНО: Синхронная верификация через Гос SOAP-сервис (возвращает ошибку, если блок)
	if err := s.verifyWithGovPlatform(ctx, email); err != nil {
		return 0, err // Сигнатура: int64, error — возвращаем 0 и ошибку!
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	repoUserType := repository.UserTypeReal
	if userType == "DIGITAL" {
		repoUserType = repository.UserTypeDigital
	}

	// Метод возвращает userID и сгенерированный базой groupID
	userID, _, err := s.userRepo.CreateUser(ctx, email, string(hashedPassword), repoUserType)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	// ИСПРАВЛЕНО: возвращаем две пустые строки вместо 0, чтобы соответствовать сигнатуре (string, string, error)
	if err := s.verifyWithGovPlatform(ctx, email); err != nil {
		return "", "", err
	}
	
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", errors.New("invalid email or password")
	}

	if !user.IsActive {
		return "", "", errors.New("user account is suspended")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", errors.New("invalid email or password")
	}

	// Передаем извлеченный group_id в генератор токенов
	return s.generateTokensPair(ctx, user.ID, string(user.UserType), user.GroupID)
}


func (s *AuthService) RefreshToken(ctx context.Context, oldRefreshToken string) (string, string, error) {
	userID, err := s.sessionRepo.GetUserIDBySession(ctx, oldRefreshToken)
	if err != nil {
		return "", "", errors.New("invalid or expired refresh token")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || !user.IsActive {
		return "", "", errors.New("user is blocked or no longer exists")
	}

	_ = s.sessionRepo.DeleteSession(ctx, oldRefreshToken)

	uType := string(user.UserType)

	// Передаем group_id при обновлении пары сессионных токенов
	return s.generateTokensPair(ctx, user.ID, uType, user.GroupID)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.sessionRepo.DeleteSession(ctx, refreshToken)
}

// Изменено: теперь метод возвращает три параметра (id, тип, группа)
func (s *AuthService) ValidateToken(ctx context.Context, accessTokenStr string) (int64, string, string, error) {
	token, err := jwt.Parse(accessTokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return 0, "", "", errors.New("invalid token signature or token expired")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", "", errors.New("invalid token claims")
	}

	userIDFloat, ok1 := claims["user_id"].(float64)
	userType, ok2 := claims["user_type"].(string)
	groupID, ok3 := claims["group_id"].(string) // Достали группу напрямую из криптографического клейма JWT!
	
	if !ok1 || !ok2 || !ok3 {
		return 0, "", "", errors.New("invalid data structure inside token")
	}

	userID := int64(userIDFloat)

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, "", "", errors.New("user not found")
	}
	if !user.IsActive {
		return 0, "", "", errors.New("user is blocked")
	}

	return user.ID, userType, groupID, nil
}

// Изменено: добавлен аргумент groupID
func (s *AuthService) generateTokensPair(ctx context.Context, userID int64, userType string, groupID string) (string, string, error) {
	_ = userType
	
	// Вшиваем group_id в Claims, чтобы избавить другие сервисы от повторных SQL-запросов к базе авторизации
	accessClaims := jwt.MapClaims{
		"user_id":   userID,
		"user_type": userType,
		"group_id":  groupID, // Зашили группу в JWT сессию!
		"exp":       time.Now().Add(s.accessTokenTTL).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshStr := uuid.New().String()

	err = s.sessionRepo.SetSession(ctx, refreshStr, userID, s.refreshTokenTTL)
	if err != nil {
		return "", "", err
	}

	return accessStr, refreshStr, nil
}
