package repository

import (
	"context"
	"errors"

	"github.com/telhawk-systems/telhawk-stack/auth/internal/models"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user already exists")
	ErrSessionNotFound  = errors.New("session not found")
	ErrHECTokenNotFound = errors.New("HEC token not found")
)

type Repository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	ListUsers(ctx context.Context) ([]*models.User, error)
	DeleteUser(ctx context.Context, id string) error

	CreateSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, refreshToken string) (*models.Session, error)
	GetSessionByAccessToken(ctx context.Context, accessToken string) (*models.Session, error)
	RevokeSession(ctx context.Context, refreshToken string) error

	CreateHECToken(ctx context.Context, token *models.HECToken) error
	GetHECToken(ctx context.Context, token string) (*models.HECToken, error)
	GetHECTokenByID(ctx context.Context, id string) (*models.HECToken, error)
	ListHECTokensByUser(ctx context.Context, userID string) ([]*models.HECToken, error)
	ListAllHECTokens(ctx context.Context) ([]*models.HECToken, error)
	RevokeHECToken(ctx context.Context, token string) error
}
