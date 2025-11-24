package repository

import (
	"context"
	"errors"

	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserExists           = errors.New("user already exists")
	ErrSessionNotFound      = errors.New("session not found")
	ErrHECTokenNotFound     = errors.New("HEC token not found")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrClientNotFound       = errors.New("client not found")
)

type Repository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserWithRoles(ctx context.Context, id string) (*models.User, error) // Loads UserRoles with Role and Permissions
	GetUserPermissionsVersion(ctx context.Context, userID string) (int, error)
	UpdateUser(ctx context.Context, user *models.User) error
	ListUsers(ctx context.Context) ([]*models.User, error)
	ListUsersByScope(ctx context.Context, scopeType string, orgID, clientID *string) ([]*models.User, error)
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

	// Organization queries
	GetOrganization(ctx context.Context, id string) (*models.Organization, error)
	ListOrganizations(ctx context.Context) ([]*models.Organization, error)

	// Client queries
	GetClient(ctx context.Context, id string) (*models.Client, error)
	ListClients(ctx context.Context) ([]*models.Client, error)
	ListClientsByOrganization(ctx context.Context, orgID string) ([]*models.Client, error)
}
