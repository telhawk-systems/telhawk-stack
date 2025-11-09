package repository

import (
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
	CreateUser(user *models.User) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateUser(user *models.User) error
	ListUsers() ([]*models.User, error)
	DeleteUser(id string) error

	CreateSession(session *models.Session) error
	GetSession(refreshToken string) (*models.Session, error)
	RevokeSession(refreshToken string) error

	CreateHECToken(token *models.HECToken) error
	GetHECToken(token string) (*models.HECToken, error)
	GetHECTokenByID(id string) (*models.HECToken, error)
	ListHECTokensByUser(userID string) ([]*models.HECToken, error)
	RevokeHECToken(token string) error
}
