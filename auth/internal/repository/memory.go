package repository

import (
	"context"
	"sync"
	"time"

	"github.com/telhawk-systems/telhawk-stack/auth/internal/models"
)

type InMemoryRepository struct {
	users       map[string]*models.User
	usersByName map[string]*models.User
	sessions    map[string]*models.Session
	hecTokens   map[string]*models.HECToken
	mu          sync.RWMutex
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		users:       make(map[string]*models.User),
		usersByName: make(map[string]*models.User),
		sessions:    make(map[string]*models.Session),
		hecTokens:   make(map[string]*models.HECToken),
	}
}

func (r *InMemoryRepository) CreateUser(ctx context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.usersByName[user.Username]; exists {
		return ErrUserExists
	}

	r.users[user.ID] = user
	r.usersByName[user.Username] = user
	return nil
}

func (r *InMemoryRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.usersByName[username]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *InMemoryRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (r *InMemoryRepository) UpdateUser(ctx context.Context, user *models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return ErrUserNotFound
	}

	r.users[user.ID] = user
	r.usersByName[user.Username] = user
	return nil
}

func (r *InMemoryRepository) CreateSession(ctx context.Context, session *models.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.RefreshToken] = session
	return nil
}

func (r *InMemoryRepository) GetSession(ctx context.Context, refreshToken string) (*models.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[refreshToken]
	if !exists {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

func (r *InMemoryRepository) RevokeSession(ctx context.Context, refreshToken string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[refreshToken]
	if !exists {
		return ErrSessionNotFound
	}

	now := time.Now()
	session.RevokedAt = &now
	return nil
}

func (r *InMemoryRepository) CreateHECToken(ctx context.Context, token *models.HECToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hecTokens[token.Token] = token
	return nil
}

func (r *InMemoryRepository) GetHECToken(ctx context.Context, token string) (*models.HECToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hecToken, exists := r.hecTokens[token]
	if !exists {
		return nil, ErrHECTokenNotFound
	}
	return hecToken, nil
}

// GetHECTokenByID retrieves an HEC token by its ID
func (r *InMemoryRepository) GetHECTokenByID(ctx context.Context, id string) (*models.HECToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, token := range r.hecTokens {
		if token.ID == id {
			return token, nil
		}
	}
	return nil, ErrHECTokenNotFound
}

func (r *InMemoryRepository) ListHECTokensByUser(ctx context.Context, userID string) ([]*models.HECToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tokens []*models.HECToken
	for _, token := range r.hecTokens {
		if token.UserID == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (r *InMemoryRepository) ListAllHECTokens(ctx context.Context) ([]*models.HECToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tokens []*models.HECToken
	for _, token := range r.hecTokens {
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (r *InMemoryRepository) RevokeHECToken(ctx context.Context, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hecToken, exists := r.hecTokens[token]
	if !exists {
		return ErrHECTokenNotFound
	}

	now := time.Now()
	hecToken.RevokedAt = &now
	return nil
}

func (r *InMemoryRepository) LogAudit(ctx context.Context, entry *models.AuditLogEntry) error {
	// In-memory implementation doesn't persist audit logs
	// This is for development only
	return nil
}

func (r *InMemoryRepository) ListUsers(ctx context.Context) ([]*models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var users []*models.User
	for _, user := range r.users {
		users = append(users, user)
	}
	return users, nil
}

func (r *InMemoryRepository) DeleteUser(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return ErrUserNotFound
	}

	delete(r.users, id)
	delete(r.usersByName, user.Username)
	return nil
}
