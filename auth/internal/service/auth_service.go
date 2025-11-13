package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/audit"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/models"
	"github.com/telhawk-systems/telhawk-stack/auth/internal/repository"
	"github.com/telhawk-systems/telhawk-stack/auth/pkg/tokens"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthService struct {
	repo     repository.Repository
	tokenGen *tokens.TokenGenerator
	auditLog *audit.Logger
}

func NewAuthService(repo repository.Repository, ingestClient *audit.IngestClient) *AuthService {
	var auditLogger *audit.Logger
	if ingestClient != nil {
		auditLogger = audit.NewLoggerWithRepoAndIngest("audit-secret-key", repo.(audit.Repository), ingestClient)
	} else {
		auditLogger = audit.NewLoggerWithRepo("audit-secret-key", repo.(audit.Repository))
	}

	return &AuthService{
		repo:     repo,
		tokenGen: tokens.NewTokenGenerator("access-secret-key", "refresh-secret-key"),
		auditLog: auditLogger,
	}
}

func (s *AuthService) CreateUser(req *models.CreateUserRequest, actorID, ipAddress, userAgent string) (*models.User, error) {
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

	userID, _ := uuid.NewV7()
	user := &models.User{
		ID:           userID.String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Roles:        req.Roles,
		CreatedAt:    time.Now(),
	}

	if len(user.Roles) == 0 {
		user.Roles = []string{string(models.RoleViewer)}
	}

	if err := s.repo.CreateUser(user); err != nil {
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

func (s *AuthService) Login(req *models.LoginRequest, ipAddress, userAgent string) (*models.LoginResponse, error) {
	user, err := s.repo.GetUserByUsername(req.Username)
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

	accessToken, err := s.tokenGen.GenerateAccessToken(user.ID, user.Roles)
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
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateSession(session); err != nil {
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

func (s *AuthService) RefreshToken(refreshToken string) (*models.LoginResponse, error) {
	session, err := s.repo.GetSession(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if !session.IsActive() {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.GetUserByID(session.UserID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokenGen.GenerateAccessToken(user.ID, user.Roles)
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

func (s *AuthService) ValidateToken(tokenString string) (*models.ValidateTokenResponse, error) {
	claims, err := s.tokenGen.ValidateAccessToken(tokenString)
	if err != nil {
		return &models.ValidateTokenResponse{Valid: false}, nil
	}

	return &models.ValidateTokenResponse{
		Valid:  true,
		UserID: claims.UserID,
		Roles:  claims.Roles,
	}, nil
}

func (s *AuthService) RevokeToken(refreshToken string) error {
	return s.repo.RevokeSession(refreshToken)
}

func (s *AuthService) ValidateHECToken(token, ipAddress, userAgent string) (*models.HECToken, error) {
	hecToken, err := s.repo.GetHECToken(token)
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

func (s *AuthService) ListUsers() ([]*models.User, error) {
	return s.repo.ListUsers()
}

func (s *AuthService) GetUser(userID string) (*models.User, error) {
	return s.repo.GetUserByID(userID)
}

func (s *AuthService) UpdateUserDetails(userID string, req *models.UpdateUserRequest, actorID, ipAddress, userAgent string) (*models.User, error) {
	user, err := s.repo.GetUserByID(userID)
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

	if err := s.repo.UpdateUser(user); err != nil {
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

func (s *AuthService) DeleteUser(userID, actorID, ipAddress, userAgent string) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteUser(userID); err != nil {
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

func (s *AuthService) ResetPassword(userID string, newPassword, actorID, ipAddress, userAgent string) error {
	user, err := s.repo.GetUserByID(userID)
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

	if err := s.repo.UpdateUser(user); err != nil {
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

func (s *AuthService) CreateHECToken(userID, name, expiresIn, ipAddress, userAgent string) (*models.HECToken, error) {
	tokenUUID, _ := uuid.NewV7()
	token := tokenUUID.String()

	idUUID, _ := uuid.NewV7()
	hecToken := &models.HECToken{
		ID:        idUUID.String(),
		UserID:    userID,
		Token:     token,
		Name:      name,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateHECToken(hecToken); err != nil {
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

func (s *AuthService) ListHECTokensByUser(userID string) ([]*models.HECToken, error) {
	return s.repo.ListHECTokensByUser(userID)
}

func (s *AuthService) RevokeHECTokenByUser(token, userID, ipAddress, userAgent string) error {
	hecToken, err := s.repo.GetHECToken(token)
	if err != nil {
		return err
	}

	if hecToken.UserID != userID {
		return errors.New("unauthorized")
	}

	if err := s.repo.RevokeHECToken(token); err != nil {
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
func (s *AuthService) RevokeHECTokenByID(tokenID, userID, ipAddress, userAgent string) error {
	hecToken, err := s.repo.GetHECTokenByID(tokenID)
	if err != nil {
		return err
	}

	if hecToken.UserID != userID {
		return errors.New("unauthorized")
	}

	if err := s.repo.RevokeHECToken(hecToken.Token); err != nil {
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
