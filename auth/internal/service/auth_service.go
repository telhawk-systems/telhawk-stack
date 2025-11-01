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
	repo       repository.Repository
	tokenGen   *tokens.TokenGenerator
	auditLog   *audit.Logger
}

func NewAuthService(repo repository.Repository) *AuthService {
	return &AuthService{
		repo:     repo,
		tokenGen: tokens.NewTokenGenerator("access-secret-key", "refresh-secret-key"),
		auditLog: audit.NewLogger("audit-secret-key"),
	}
}

func (s *AuthService) Register(req *models.RegisterRequest, ipAddress, userAgent string) (*models.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.auditLog.Log(
			models.ActorTypeSystem, "", req.Username,
			models.ActionRegister, "user", "",
			ipAddress, userAgent,
			models.ResultFailure, "password hashing failed",
			nil,
		)
		return nil, err
	}

	user := &models.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Roles:        req.Roles,
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if len(user.Roles) == 0 {
		user.Roles = []string{string(models.RoleViewer)}
	}

	if err := s.repo.CreateUser(user); err != nil {
		s.auditLog.Log(
			models.ActorTypeSystem, "", req.Username,
			models.ActionRegister, "user", user.ID,
			ipAddress, userAgent,
			models.ResultFailure, err.Error(),
			nil,
		)
		return nil, err
	}

	s.auditLog.Log(
		models.ActorTypeUser, user.ID, user.Username,
		models.ActionRegister, "user", user.ID,
		ipAddress, userAgent,
		models.ResultSuccess, "",
		map[string]interface{}{
			"email": user.Email,
			"roles": user.Roles,
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

	if !user.Enabled {
		s.auditLog.Log(
			models.ActorTypeUser, user.ID, user.Username,
			models.ActionLogin, "session", "",
			ipAddress, userAgent,
			models.ResultFailure, "account disabled",
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

	session := &models.Session{
		ID:           uuid.New().String(),
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:    time.Now(),
		Revoked:      false,
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

	if session.Revoked || time.Now().After(session.ExpiresAt) {
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

	if !hecToken.Enabled {
		s.auditLog.Log(
			models.ActorTypeService, hecToken.UserID, "",
			models.ActionHECTokenValidate, "hec_token", hecToken.ID,
			ipAddress, userAgent,
			models.ResultFailure, "token disabled",
			map[string]interface{}{"token_name": hecToken.Name},
		)
		return nil, ErrInvalidToken
	}

	if !hecToken.ExpiresAt.IsZero() && time.Now().After(hecToken.ExpiresAt) {
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
