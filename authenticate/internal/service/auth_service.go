package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/audit"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/config"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/models"
	"github.com/telhawk-systems/telhawk-stack/authenticate/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/authenticate/pkg/tokens"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

// stringOrEmpty returns empty string if the pointer is nil, otherwise the value.
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type AuthService struct {
	repo     repository.Repository
	tokenGen *tokens.TokenGenerator
	auditLog *audit.Logger
}

func NewAuthService(repo repository.Repository, ingestClient *audit.IngestClient, cfg *config.AuthConfig) *AuthService {
	var auditLogger *audit.Logger
	auditRepo, ok := repo.(audit.Repository)
	if !ok {
		panic("repository does not implement audit.Repository interface")
	}

	if ingestClient != nil {
		auditLogger = audit.NewLoggerWithRepoAndIngest(cfg.AuditSecret, auditRepo, ingestClient)
	} else {
		auditLogger = audit.NewLoggerWithRepo(cfg.AuditSecret, auditRepo)
	}

	return &AuthService{
		repo:     repo,
		tokenGen: tokens.NewTokenGenerator(cfg.JWTSecret, cfg.JWTRefreshSecret),
		auditLog: auditLogger,
	}
}

func (s *AuthService) CreateUser(ctx context.Context, req *models.CreateUserRequest, actorID, ipAddress, userAgent string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, actorID, "",
			models.ActionUserCreate, "user", "",
			ipAddress, userAgent,
			models.ResultFailure, "password hashing failed",
			nil,
		)
		return nil, err
	}

	userID, err := uuid.NewV7()
	if err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, actorID, "",
			models.ActionUserCreate, "user", "",
			ipAddress, userAgent,
			models.ResultFailure, "failed to generate user ID",
			nil,
		)
		return nil, fmt.Errorf("failed to generate user ID: %w", err)
	}

	user := &models.User{
		ID:           userID.String(),
		VersionID:    userID.String(), // Same as ID for initial version
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Roles:        req.Roles,
	}

	if len(user.Roles) == 0 {
		user.Roles = []string{string(models.LegacyRoleViewer)}
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, actorID, "",
			models.ActionUserCreate, "user", user.ID,
			ipAddress, userAgent,
			models.ResultFailure, err.Error(),
			nil,
		)
		return nil, err
	}

	s.auditLog.Log(
		models.ActorTypeUser, actorID, "",
		models.ActionUserCreate, "user", user.ID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		map[string]interface{}{
			"username": user.Username,
			"email":    user.Email,
			"roles":    user.Roles,
		},
	)

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest, ipAddress, userAgent string) (*models.LoginResponse, error) {
	user, err := s.repo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, "", req.Username,
			models.ActionLogin, "session", "",
			ipAddress, userAgent,
			models.ResultFailure, "user not found",
			nil,
		)
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive() {
		s.auditLog.Log(
			models.ActorTypeUser, user.ID, user.Username,
			models.ActionLogin, "session", "",
			ipAddress, userAgent,
			models.ResultFailure, "account disabled or deleted",
			nil,
		)
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, user.ID, user.Username,
			models.ActionLogin, "session", "",
			ipAddress, userAgent,
			models.ResultFailure, "invalid password",
			nil,
		)
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokenGen.GenerateAccessToken(
		user.ID, user.Roles, user.PermissionsVersion,
		stringOrEmpty(user.PrimaryOrganizationID), stringOrEmpty(user.PrimaryClientID),
	)
	if err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, user.ID, user.Username,
			models.ActionLogin, "session", "",
			ipAddress, userAgent,
			models.ResultFailure, "token generation failed",
			nil,
		)
		return nil, err
	}

	refreshToken, err := s.tokenGen.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	sessionID, _ := uuid.NewV7()
	session := &models.Session{
		ID:           sessionID.String(),
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	s.auditLog.Log(
		models.ActorTypeUser, user.ID, user.Username,
		models.ActionLogin, "session", session.ID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		map[string]interface{}{
			"session_id": session.ID,
			"roles":      user.Roles,
		},
	)

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.LoginResponse, error) {
	session, err := s.repo.GetSession(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if !session.IsActive() {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokenGen.GenerateAccessToken(
		user.ID, user.Roles, user.PermissionsVersion,
		stringOrEmpty(user.PrimaryOrganizationID), stringOrEmpty(user.PrimaryClientID),
	)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*models.ValidateTokenResponse, error) {
	claims, err := s.tokenGen.ValidateAccessToken(tokenString)
	if err != nil {
		return &models.ValidateTokenResponse{Valid: false}, nil
	}

	// Verify session exists in database and is active
	// This ensures tokens are invalidated when sessions are revoked or database is wiped
	session, err := s.repo.GetSessionByAccessToken(ctx, tokenString)
	if err != nil {
		// Session not found in database - token is invalid
		return &models.ValidateTokenResponse{Valid: false}, nil
	}

	if !session.IsActive() {
		// Session has been revoked or expired
		return &models.ValidateTokenResponse{Valid: false}, nil
	}

	// Check if JWT permissions version matches current DB version
	// This detects stale permissions when roles have changed since token was issued
	currentVersion, err := s.repo.GetUserPermissionsVersion(ctx, claims.UserID)
	permissionsStale := false
	if err == nil && currentVersion != claims.PermissionsVersion {
		permissionsStale = true
	}

	return &models.ValidateTokenResponse{
		Valid:              true,
		UserID:             claims.UserID,
		Roles:              claims.Roles,
		PermissionsVersion: currentVersion,
		PermissionsStale:   permissionsStale,
		OrganizationID:     claims.OrganizationID,
		ClientID:           claims.ClientID,
	}, nil
}

func (s *AuthService) RevokeToken(ctx context.Context, refreshToken string) error {
	return s.repo.RevokeSession(ctx, refreshToken)
}

// GetUserByID retrieves a user by their ID (used for refreshing stale permission data)
func (s *AuthService) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

// GetUserWithRoles retrieves a user with their full RBAC data (UserRoles -> Role -> Permissions)
// Use this for permission checking in middleware
func (s *AuthService) GetUserWithRoles(ctx context.Context, userID string) (*models.User, error) {
	return s.repo.GetUserWithRoles(ctx, userID)
}

func (s *AuthService) ValidateHECToken(ctx context.Context, token, ipAddress, userAgent string) (*models.HECToken, error) {
	hecToken, err := s.repo.GetHECToken(ctx, token)
	if err != nil {
		s.auditLog.Log(
			models.ActorTypeService, "", "",
			models.ActionHECTokenValidate, "hec_token", "",
			ipAddress, userAgent,
			models.ResultFailure, "token not found",
			nil,
		)
		return nil, err
	}

	if !hecToken.IsActive() {
		s.auditLog.Log(
			models.ActorTypeService, hecToken.UserID, "",
			models.ActionHECTokenValidate, "hec_token", hecToken.ID,
			ipAddress, userAgent,
			models.ResultFailure, "token disabled or revoked",
			map[string]interface{}{"token_name": hecToken.Name},
		)
		return nil, ErrInvalidToken
	}

	if hecToken.ExpiresAt != nil && !hecToken.ExpiresAt.IsZero() && time.Now().After(*hecToken.ExpiresAt) {
		s.auditLog.Log(
			models.ActorTypeService, hecToken.UserID, "",
			models.ActionHECTokenValidate, "hec_token", hecToken.ID,
			ipAddress, userAgent,
			models.ResultFailure, "token expired",
			map[string]interface{}{"token_name": hecToken.Name},
		)
		return nil, ErrInvalidToken
	}

	s.auditLog.Log(
		models.ActorTypeService, hecToken.UserID, "",
		models.ActionHECTokenValidate, "hec_token", hecToken.ID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		map[string]interface{}{"token_name": hecToken.Name},
	)

	return hecToken, nil
}

func (s *AuthService) ListUsers(ctx context.Context) ([]*models.User, error) {
	return s.repo.ListUsers(ctx)
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*models.User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

func (s *AuthService) UpdateUserDetails(ctx context.Context, userID string, req *models.UpdateUserRequest, actorID, ipAddress, userAgent string) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Roles != nil {
		user.Roles = req.Roles
	}
	// Note: Use DisableUser/EnableUser methods to manage user lifecycle

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, actorID, "",
			models.ActionUpdate, "user", userID,
			ipAddress, userAgent,
			models.ResultFailure, err.Error(),
			map[string]interface{}{"changes": req},
		)
		return nil, err
	}

	s.auditLog.Log(
		models.ActorTypeUser, actorID, "",
		models.ActionUpdate, "user", userID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		map[string]interface{}{"changes": req},
	)

	return user, nil
}

func (s *AuthService) DeleteUser(ctx context.Context, userID, actorID, ipAddress, userAgent string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteUser(ctx, userID); err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, actorID, "",
			models.ActionDelete, "user", userID,
			ipAddress, userAgent,
			models.ResultFailure, err.Error(),
			map[string]interface{}{"username": user.Username},
		)
		return err
	}

	s.auditLog.Log(
		models.ActorTypeUser, actorID, "",
		models.ActionDelete, "user", userID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		map[string]interface{}{"username": user.Username},
	)

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, userID string, newPassword, actorID, ipAddress, userAgent string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, actorID, "",
			models.ActionPasswordReset, "user", userID,
			ipAddress, userAgent,
			models.ResultFailure, "password hashing failed",
			nil,
		)
		return err
	}

	user.PasswordHash = string(hashedPassword)

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, actorID, "",
			models.ActionPasswordReset, "user", userID,
			ipAddress, userAgent,
			models.ResultFailure, err.Error(),
			nil,
		)
		return err
	}

	s.auditLog.Log(
		models.ActorTypeUser, actorID, "",
		models.ActionPasswordReset, "user", userID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		nil,
	)

	return nil
}

func (s *AuthService) CreateHECToken(ctx context.Context, userID, clientID, name, expiresIn, ipAddress, userAgent string) (*models.HECToken, error) {
	if clientID == "" {
		return nil, fmt.Errorf("client_id is required for HEC token creation")
	}

	tokenUUID, _ := uuid.NewV7()
	token := tokenUUID.String()

	idUUID, _ := uuid.NewV7()
	hecToken := &models.HECToken{
		ID:        idUUID.String(),
		UserID:    userID,
		ClientID:  clientID,
		CreatedBy: userID,
		Token:     token,
		Name:      name,
	}

	if err := s.repo.CreateHECToken(ctx, hecToken); err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, userID, "",
			models.ActionHECTokenCreate, "hec_token", hecToken.ID,
			ipAddress, userAgent,
			models.ResultFailure, err.Error(),
			nil,
		)
		return nil, err
	}

	s.auditLog.Log(
		models.ActorTypeUser, userID, "",
		models.ActionHECTokenCreate, "hec_token", hecToken.ID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		map[string]interface{}{
			"token_name": name,
		},
	)

	return hecToken, nil
}

func (s *AuthService) ListHECTokensByUser(ctx context.Context, userID string) ([]*models.HECToken, error) {
	return s.repo.ListHECTokensByUser(ctx, userID)
}

func (s *AuthService) ListAllHECTokensWithUsernames(ctx context.Context) (map[string]string, []*models.HECToken, error) {
	tokens, err := s.repo.ListAllHECTokens(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Get unique user IDs
	userIDs := make(map[string]bool)
	for _, token := range tokens {
		userIDs[token.UserID] = true
	}

	// Fetch usernames for all user IDs
	usernames := make(map[string]string)
	for userID := range userIDs {
		user, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get username for user %s: %w", userID, err)
		}
		usernames[userID] = user.Username
	}

	return usernames, tokens, nil
}

func (s *AuthService) RevokeHECTokenByUser(ctx context.Context, token, userID, ipAddress, userAgent string) error {
	hecToken, err := s.repo.GetHECToken(ctx, token)
	if err != nil {
		return err
	}

	if hecToken.UserID != userID {
		return errors.New("unauthorized")
	}

	if err := s.repo.RevokeHECToken(ctx, token); err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, userID, "",
			models.ActionHECTokenRevoke, "hec_token", hecToken.ID,
			ipAddress, userAgent,
			models.ResultFailure, err.Error(),
			nil,
		)
		return err
	}

	s.auditLog.Log(
		models.ActorTypeUser, userID, "",
		models.ActionHECTokenRevoke, "hec_token", hecToken.ID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		nil,
	)

	return nil
}

// RevokeHECTokenByID revokes an HEC token by its ID (used for RESTful endpoint)
func (s *AuthService) RevokeHECTokenByID(ctx context.Context, tokenID, userID, ipAddress, userAgent string) error {
	hecToken, err := s.repo.GetHECTokenByID(ctx, tokenID)
	if err != nil {
		return err
	}

	if hecToken.UserID != userID {
		return errors.New("unauthorized")
	}

	if err := s.repo.RevokeHECToken(ctx, hecToken.Token); err != nil {
		s.auditLog.Log(
			models.ActorTypeUser, userID, "",
			models.ActionHECTokenRevoke, "hec_token", hecToken.ID,
			ipAddress, userAgent,
			models.ResultFailure, err.Error(),
			nil,
		)
		return err
	}

	s.auditLog.Log(
		models.ActorTypeUser, userID, "",
		models.ActionHECTokenRevoke, "hec_token", hecToken.ID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		nil,
	)

	return nil
}

// UserScopeResponse contains the user's accessible organizations and clients
type UserScopeResponse struct {
	MaxTier       string                         `json:"max_tier"` // platform, organization, or client
	Organizations []*models.OrganizationResponse `json:"organizations"`
	Clients       []*ClientWithOrgResponse       `json:"clients"`
}

// ClientWithOrgResponse is a client with its parent organization ID
type ClientWithOrgResponse struct {
	*models.ClientResponse
}

// GetUserScope returns the organizations and clients accessible to the user
// For now, this returns all active organizations and clients (admin gets everything)
// TODO: Filter based on user's role assignments when full RBAC is implemented
func (s *AuthService) GetUserScope(ctx context.Context, userID string, roles []string) (*UserScopeResponse, error) {
	// Determine max tier from roles
	// Platform users (admin role) can see everything
	// Others are client-scoped for now
	maxTier := "client"
	isAdmin := false
	for _, role := range roles {
		if role == "admin" {
			maxTier = "platform"
			isAdmin = true
			break
		}
	}

	response := &UserScopeResponse{
		MaxTier:       maxTier,
		Organizations: []*models.OrganizationResponse{},
		Clients:       []*ClientWithOrgResponse{},
	}

	// Get organizations (only for platform-tier users)
	if isAdmin {
		orgs, err := s.repo.ListOrganizations(ctx)
		if err != nil {
			return nil, err
		}
		for _, org := range orgs {
			response.Organizations = append(response.Organizations, org.ToResponse())
		}
	}

	// Get clients for each organization
	if isAdmin {
		// Admin sees all clients
		for _, org := range response.Organizations {
			clients, err := s.repo.ListClientsByOrganization(ctx, org.ID)
			if err != nil {
				return nil, err
			}
			for _, client := range clients {
				response.Clients = append(response.Clients, &ClientWithOrgResponse{
					ClientResponse: client.ToResponse(),
				})
			}
		}
	} else {
		// Non-admin: get client from user's primary client_id
		user, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if user.PrimaryClientID != nil {
			client, err := s.repo.GetClient(ctx, *user.PrimaryClientID)
			if err == nil {
				response.Clients = append(response.Clients, &ClientWithOrgResponse{
					ClientResponse: client.ToResponse(),
				})
			}
		}
	}

	return response, nil
}

// ClientBelongsToOrg checks if a client belongs to a specific organization.
// This is used for scope-aware permission checking.
func (s *AuthService) ClientBelongsToOrg(clientID, orgID string) bool {
	client, err := s.repo.GetClient(context.Background(), clientID)
	if err != nil {
		return false
	}
	return client.OrganizationID == orgID
}

// ClientBelongsToOrgFunc returns a function suitable for passing to User.CanActInScope.
// This allows the service to provide the lookup capability to the model layer.
func (s *AuthService) ClientBelongsToOrgFunc() func(clientID, orgID string) bool {
	return s.ClientBelongsToOrg
}
