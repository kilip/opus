package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kilip/opus/server/internal/shared/logger"
)

// Service provides authorization mechanisms and handles user creation.
type Service struct {
	repo          Repository
	registry      *ProviderRegistry
	policyService PolicyService
	config        Config
	log           logger.Logger
}

// NewService instantiates standard Service fields satisfying DI parameters.
func NewService(repo Repository, reg *ProviderRegistry, ps PolicyService, config Config, log logger.Logger) *Service {
	return &Service{
		repo:          repo,
		registry:      reg,
		policyService: ps,
		config:        config,
		log:           log,
	}
}

// Registry exposes provider registers dynamically to controllers.
func (s *Service) Registry() *ProviderRegistry {
	return s.registry
}

// Register creates a new credential user account alongside a personal workspace.
func (s *Service) Register(ctx context.Context, email, password, name string) (*User, *TokenPair, error) {
	_, err := s.repo.FindUserByEmail(ctx, email)
	if err == nil {
		return nil, nil, ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	userUUID, err := uuid.NewV7()
	if err != nil {
		return nil, nil, fmt.Errorf("auth: failed to generate user uuid: %w", err)
	}
	uid := userUUID.String()

	workspaceUUID, err := uuid.NewV7()
	if err != nil {
		return nil, nil, fmt.Errorf("auth: failed to generate workspace uuid: %w", err)
	}
	wid := workspaceUUID.String()

	user := &User{
		ID:           uid,
		Email:        email,
		Name:         name,
		Provider:     "credential",
		ProviderID:   uid,
		WorkspaceID:  wid,
		PasswordHash: string(hash),
	}

	createdUser, err := s.repo.CreateUserWithWorkspace(ctx, user, nil, name+"'s Workspace")
	if err != nil {
		return nil, nil, err
	}

	_, err = s.policyService.AssignRole(ctx, createdUser.ID, createdUser.WorkspaceID, "admin")
	if err != nil {
		return nil, nil, err
	}

	tokens, err := s.issueTokens(ctx, createdUser)
	if err != nil {
		return nil, nil, err
	}

	return createdUser, tokens, nil
}

// Login verifies incoming credentials and generates standard access configurations.
func (s *Service) Login(ctx context.Context, email, password string) (*User, *TokenPair, error) {
	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	if user.PasswordHash == "" {
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// Refresh performs stateful JWT rotation and implements refresh token replay protection.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	hashedToken := HashToken(refreshToken)
	tokenRecord, err := s.repo.FindTokenByHash(ctx, hashedToken)
	if err != nil {
		return nil, ErrTokenNotFound
	}

	// Replay detection check
	if tokenRecord.RevokedAt != nil {
		_ = s.repo.RevokeSessionTokens(ctx, tokenRecord.SessionID)
		return nil, ErrTokenReplay
	}

	if time.Now().After(tokenRecord.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	session, err := s.repo.FindSessionByID(ctx, tokenRecord.SessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if err := s.repo.RevokeToken(ctx, tokenRecord.ID); err != nil {
		return nil, err
	}

	accTTL, err := time.ParseDuration(s.config.AccessTokenTTL)
	if err != nil {
		accTTL = 15 * time.Minute
	}
	refTTL, err := time.ParseDuration(s.config.RefreshTokenTTL)
	if err != nil {
		refTTL = 168 * time.Hour
	}

	claimUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate claim uuid: %w", err)
	}

	claims := Claims{
		SessionID:   session.ID,
		WorkspaceID: user.WorkspaceID,
		Role:        "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: user.ID,
			ID:      claimUUID.String(),
		},
	}

	tokens, err := GenerateTokenPair(s.config.JWTSecret, claims, accTTL, refTTL)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	accessTokenHash := HashToken(tokens.AccessToken)
	refreshTokenHash := HashToken(tokens.RefreshToken)

	accessUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate access token uuid: %w", err)
	}
	refreshUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate refresh token uuid: %w", err)
	}

	accessRecord := &Token{
		ID:        accessUUID.String(),
		SessionID: session.ID,
		UserID:    user.ID,
		Type:      "access",
		Hash:      accessTokenHash,
		ExpiresAt: now.Add(accTTL),
	}

	refreshRecord := &Token{
		ID:        refreshUUID.String(),
		SessionID: session.ID,
		UserID:    user.ID,
		Type:      "refresh",
		Hash:      refreshTokenHash,
		ExpiresAt: now.Add(refTTL),
	}

	if err := s.repo.CreateToken(ctx, accessRecord); err != nil {
		return nil, err
	}
	if err := s.repo.CreateToken(ctx, refreshRecord); err != nil {
		return nil, err
	}

	return tokens, nil
}

// OAuthCallback processes OAuth exchanges and provisions users and workspaces.
func (s *Service) OAuthCallback(ctx context.Context, providerID, code, state string) (*User, *TokenPair, error) {
	storedProvider, err := s.repo.ValidateOAuthState(ctx, state)
	if err != nil || storedProvider != providerID {
		return nil, nil, ErrInvalidOAuthState
	}

	provider, err := s.registry.Get(providerID)
	if err != nil {
		return nil, nil, err
	}

	profile, err := provider.Exchange(ctx, code)
	if err != nil {
		return nil, nil, err
	}

	_, user, err := s.repo.FindAccountByProvider(ctx, providerID, profile.ProviderID)
	if err == nil {
		tokens, err := s.issueTokens(ctx, user)
		return user, tokens, err
	}

	user, err = s.repo.FindUserByEmail(ctx, profile.Email)
	if err == nil {
		newAccUUID, err := uuid.NewV7()
		if err != nil {
			return nil, nil, fmt.Errorf("auth: failed to generate account uuid: %w", err)
		}
		newAcc := &Account{
			ID:         newAccUUID.String(),
			UserID:     user.ID,
			AccountID:  profile.ProviderID,
			ProviderID: providerID,
		}
		if err := s.repo.LinkAccount(ctx, newAcc); err != nil {
			return nil, nil, err
		}
		tokens, err := s.issueTokens(ctx, user)
		return user, tokens, err
	}

	u, err := uuid.NewV7()
	if err != nil {
		return nil, nil, fmt.Errorf("auth: failed to generate user uuid: %w", err)
	}
	uid := u.String()

	w, err := uuid.NewV7()
	if err != nil {
		return nil, nil, fmt.Errorf("auth: failed to generate workspace uuid: %w", err)
	}
	wid := w.String()

	newUser := &User{
		ID:          uid,
		Email:       profile.Email,
		Name:        profile.Name,
		AvatarURL:   profile.AvatarURL,
		Provider:    providerID,
		ProviderID:  profile.ProviderID,
		WorkspaceID: wid,
	}

	newAccUUID, err := uuid.NewV7()
	if err != nil {
		return nil, nil, fmt.Errorf("auth: failed to generate account uuid: %w", err)
	}
	newAcc := &Account{
		ID:         newAccUUID.String(),
		UserID:     uid,
		AccountID:  profile.ProviderID,
		ProviderID: providerID,
	}

	createdUser, err := s.repo.CreateUserWithWorkspace(ctx, newUser, newAcc, profile.Name+"'s Workspace")
	if err != nil {
		return nil, nil, err
	}

	_, err = s.policyService.AssignRole(ctx, createdUser.ID, createdUser.WorkspaceID, "admin")
	if err != nil {
		return nil, nil, err
	}

	tokens, err := s.issueTokens(ctx, createdUser)
	if err != nil {
		return nil, nil, err
	}

	return createdUser, tokens, nil
}

// Logout invalidates all database access signatures tied to a session ID.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.repo.RevokeSessionTokens(ctx, sessionID)
}

func (s *Service) issueTokens(ctx context.Context, user *User) (*TokenPair, error) {
	sessionUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate session uuid: %w", err)
	}
	sessionID := sessionUUID.String()

	session := &Session{
		ID:          sessionID,
		UserID:      user.ID,
		WorkspaceID: user.WorkspaceID,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	accTTL, err := time.ParseDuration(s.config.AccessTokenTTL)
	if err != nil {
		accTTL = 15 * time.Minute
	}
	refTTL, err := time.ParseDuration(s.config.RefreshTokenTTL)
	if err != nil {
		refTTL = 168 * time.Hour
	}

	claimUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate claim uuid: %w", err)
	}

	claims := Claims{
		SessionID:   sessionID,
		WorkspaceID: user.WorkspaceID,
		Role:        "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: user.ID,
			ID:      claimUUID.String(),
		},
	}

	tokens, err := GenerateTokenPair(s.config.JWTSecret, claims, accTTL, refTTL)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	accessTokenHash := HashToken(tokens.AccessToken)
	refreshTokenHash := HashToken(tokens.RefreshToken)

	accessUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate access token uuid: %w", err)
	}
	refreshUUID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("auth: failed to generate refresh token uuid: %w", err)
	}

	accessRecord := &Token{
		ID:        accessUUID.String(),
		SessionID: sessionID,
		UserID:    user.ID,
		Type:      "access",
		Hash:      accessTokenHash,
		ExpiresAt: now.Add(accTTL),
	}

	refreshRecord := &Token{
		ID:        refreshUUID.String(),
		SessionID: sessionID,
		UserID:    user.ID,
		Type:      "refresh",
		Hash:      refreshTokenHash,
		ExpiresAt: now.Add(refTTL),
	}

	if err := s.repo.CreateToken(ctx, accessRecord); err != nil {
		return nil, err
	}
	if err := s.repo.CreateToken(ctx, refreshRecord); err != nil {
		return nil, err
	}

	return tokens, nil
}
