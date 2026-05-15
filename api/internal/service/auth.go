package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/opus/api/internal/config"
	"github.com/kilip/opus/api/internal/model"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByProviderID(ctx context.Context, provider, providerID string) (*model.User, error)
	Create(ctx context.Context, user *model.User) (*model.User, error)
	Update(ctx context.Context, user *model.User) (*model.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) (*model.Session, error)
	FindByTokenHash(ctx context.Context, hash string) (*model.Session, error)
	RevokeByID(ctx context.Context, id string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
}

type AuthServiceInterface interface {
	UpsertOAuthUser(ctx context.Context, provider, providerID, email, name, avatarURL string) (*model.User, error)
	IssueTokens(ctx context.Context, userID string) (string, string, error)
	RefreshTokens(ctx context.Context, rawRefreshToken string) (string, string, error)
	Logout(ctx context.Context, rawRefreshToken string) error
	ValidateAccessToken(tokenString string) (string, error)
}

type AuthService struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
	cfg         *config.Config
}

func NewAuthService(userRepo UserRepository, sessionRepo SessionRepository, cfg *config.Config) AuthServiceInterface {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		cfg:         cfg,
	}
}

func (s *AuthService) UpsertOAuthUser(ctx context.Context, provider, providerID, email, name, avatarURL string) (*model.User, error) {
	u, err := s.userRepo.FindByProviderID(ctx, provider, providerID)
	if err != nil && err != model.ErrUserNotFound {
		return nil, fmt.Errorf("authService.UpsertOAuthUser: %w", err)
	}

	if u == nil {
		// Try find by email
		u, err = s.userRepo.FindByEmail(ctx, email)
		if err != nil && err != model.ErrUserNotFound {
			return nil, fmt.Errorf("authService.UpsertOAuthUser: %w", err)
		}
	}

	if u == nil {
		// Create new user
		u = &model.User{
			ID:         fmt.Sprintf("user_%d", time.Now().UnixNano()), // Simple ID generation
			Email:      email,
			Name:       name,
			AvatarURL:  avatarURL,
			Provider:   provider,
			ProviderID: providerID,
		}
		u, err = s.userRepo.Create(ctx, u)
		if err != nil {
			return nil, fmt.Errorf("authService.UpsertOAuthUser: %w", err)
		}
	} else {
		// Update existing user
		u.Name = name
		u.AvatarURL = avatarURL
		u.Provider = provider
		u.ProviderID = providerID
		u, err = s.userRepo.Update(ctx, u)
		if err != nil {
			return nil, fmt.Errorf("authService.UpsertOAuthUser: %w", err)
		}
	}

	return u, nil
}

func (s *AuthService) IssueTokens(ctx context.Context, userID string) (string, string, error) {
	// Generate Access Token (JWT)
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return "", "", err
	}

	// Generate Refresh Token (Opaque)
	refreshToken, err := s.generateRandomToken()
	if err != nil {
		return "", "", err
	}

	// Store session
	hash := s.hashToken(refreshToken)
	session := &model.Session{
		ID:        fmt.Sprintf("sess_%d", time.Now().UnixNano()),
		TokenHash: hash,
		UserID:    userID,
		ExpiresAt: time.Now().Add(time.Duration(s.cfg.Auth.RefreshTokenTTL) * time.Minute),
		Revoked:   false,
	}

	_, err = s.sessionRepo.Create(ctx, session)
	if err != nil {
		return "", "", fmt.Errorf("authService.IssueTokens: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (s *AuthService) RefreshTokens(ctx context.Context, rawRefreshToken string) (string, string, error) {
	hash := s.hashToken(rawRefreshToken)
	sess, err := s.sessionRepo.FindByTokenHash(ctx, hash)
	if err != nil {
		return "", "", fmt.Errorf("authService.RefreshTokens: %w", err)
	}

	if sess.Revoked {
		// Replay attack detected
		_ = s.sessionRepo.RevokeAllByUserID(ctx, sess.UserID)
		return "", "", model.ErrTokenRevoked
	}

	if time.Now().After(sess.ExpiresAt) {
		return "", "", model.ErrTokenExpired
	}

	// Rotate tokens
	err = s.sessionRepo.RevokeByID(ctx, sess.ID)
	if err != nil {
		return "", "", fmt.Errorf("authService.RefreshTokens: %w", err)
	}

	return s.IssueTokens(ctx, sess.UserID)
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := s.hashToken(rawRefreshToken)
	sess, err := s.sessionRepo.FindByTokenHash(ctx, hash)
	if err != nil {
		if err == model.ErrSessionNotFound {
			return nil
		}
		return fmt.Errorf("authService.Logout: %w", err)
	}

	return s.sessionRepo.RevokeByID(ctx, sess.ID)
}

func (s *AuthService) ValidateAccessToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.Auth.Secret), nil
	})

	if err != nil {
		return "", model.ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["sub"].(string)
		if !ok {
			return "", model.ErrInvalidToken
		}
		return userID, nil
	}

	return "", model.ErrInvalidToken
}

func (s *AuthService) generateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(s.cfg.Auth.AccessTokenTTL) * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Auth.Secret))
}

func (s *AuthService) generateRandomToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *AuthService) hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
