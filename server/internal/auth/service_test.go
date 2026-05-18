package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/opus/server/internal/auth"
	"github.com/kilip/opus/server/internal/shared/logger"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := auth.NewMockRepository(ctrl)
	log := logger.NewMockLogger(ctrl)
	ps := auth.NewMockPolicyService(ctrl)
	reg := auth.NewProviderRegistry()

	svc := auth.NewService(repo, reg, ps, auth.Config{
		JWTSecret:       "secret",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}, log)

	ctx := context.Background()
	email := "test@test.com"
	password := "password123"
	name := "Test User"

	// Mock DB expectation
	repo.EXPECT().FindUserByEmail(ctx, email).Return(nil, auth.ErrUserNotFound)
	repo.EXPECT().CreateUserWithWorkspace(ctx, gomock.Any(), nil, name+"'s Workspace").DoAndReturn(
		func(ctx context.Context, u *auth.User, acc *auth.Account, wName string) (*auth.User, error) {
			// Verify ID generation (Must be compilable UUID format)
			if len(u.ID) != 36 || len(u.WorkspaceID) != 36 {
				t.Errorf("UUID v7 must be 36 characters, user=%d workspace=%d", len(u.ID), len(u.WorkspaceID))
			}
			return &auth.User{
				ID:          u.ID,
				Email:       u.Email,
				WorkspaceID: u.WorkspaceID,
			}, nil
		},
	)
	ps.EXPECT().AssignRole(ctx, gomock.Any(), gomock.Any(), "admin").Return(true, nil)
	repo.EXPECT().CreateSession(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().CreateToken(ctx, gomock.Any()).Times(2).Return(nil)

	user, tokens, err := svc.Register(ctx, email, password, name)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user.Email != email {
		t.Errorf("expected email test@test.com, got %s", user.Email)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Errorf("expected access and refresh tokens to be returned")
	}
}

func TestService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := auth.NewMockRepository(ctrl)
	ps := auth.NewMockPolicyService(ctrl)
	reg := auth.NewProviderRegistry()
	svc := auth.NewService(repo, reg, ps, auth.Config{
		JWTSecret:       "secret",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}, nil)

	ctx := context.Background()
	email := "test@test.com"
	password := "secret"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	repo.EXPECT().FindUserByEmail(ctx, email).Return(&auth.User{
		ID:           "u1",
		Email:        email,
		WorkspaceID:  "w1",
		PasswordHash: string(hash),
	}, nil)
	repo.EXPECT().CreateSession(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().CreateToken(ctx, gomock.Any()).Times(2).Return(nil)

	_, tokens, err := svc.Login(ctx, email, password)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tokens.AccessToken == "" {
		t.Errorf("expected valid access token")
	}
}

func TestService_Refresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := auth.NewMockRepository(ctrl)
	ps := auth.NewMockPolicyService(ctrl)
	reg := auth.NewProviderRegistry()
	svc := auth.NewService(repo, reg, ps, auth.Config{
		JWTSecret:       "secret",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}, nil)

	ctx := context.Background()
	refreshToken := "old-refresh-token"
	hashedToken := auth.HashToken(refreshToken)

	tokenRecord := &auth.Token{
		ID:        "t1",
		SessionID: "s1",
		UserID:    "u1",
		Type:      "refresh",
		Hash:      hashedToken,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	repo.EXPECT().FindTokenByHash(ctx, hashedToken).Return(tokenRecord, nil)
	repo.EXPECT().FindSessionByID(ctx, "s1").Return(&auth.Session{ID: "s1", UserID: "u1", WorkspaceID: "w1"}, nil)
	repo.EXPECT().FindUserByID(ctx, "u1").Return(&auth.User{ID: "u1", WorkspaceID: "w1"}, nil)
	repo.EXPECT().RevokeToken(ctx, "t1").Return(nil)
	repo.EXPECT().CreateToken(ctx, gomock.Any()).Times(2).Return(nil)

	tokens, err := svc.Refresh(ctx, refreshToken)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if tokens.RefreshToken == refreshToken {
		t.Errorf("expected refresh token to rotate")
	}
}

func TestService_OAuthCallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := auth.NewMockRepository(ctrl)
	ps := auth.NewMockPolicyService(ctrl)

	// Create mock provider registry and mock Google OAuthProvider
	googleMockProvider := auth.NewMockOAuthProvider(ctrl)
	googleMockProvider.EXPECT().Name().Return("google").AnyTimes()
	reg := auth.NewProviderRegistry(googleMockProvider)

	svc := auth.NewService(repo, reg, ps, auth.Config{
		JWTSecret:       "secret",
		AccessTokenTTL:  "15m",
		RefreshTokenTTL: "168h",
	}, nil)

	ctx := context.Background()
	code := "oauth-authorization-code"
	state := "oauth-secure-state"

	// Mocking verification sequence
	repo.EXPECT().ValidateOAuthState(ctx, state).Return("google", nil)
	googleMockProvider.EXPECT().Exchange(ctx, code).Return(&auth.OAuthProfile{
		ProviderID: "g1",
		Provider:   "google",
		Email:      "g@google.com",
		Name:       "Google User",
		AvatarURL:  "https://avatar.com/google",
	}, nil)

	// User not found in DB
	repo.EXPECT().FindAccountByProvider(ctx, "google", "g1").Return(nil, nil, auth.ErrUserNotFound)
	repo.EXPECT().FindUserByEmail(ctx, "g@google.com").Return(nil, auth.ErrUserNotFound)

	// Provisioning logic gets invoked
	repo.EXPECT().CreateUserWithWorkspace(ctx, gomock.Any(), gomock.Any(), "Google User's Workspace").Return(&auth.User{
		ID:          "u-google",
		Email:       "g@google.com",
		WorkspaceID: "ws-google",
	}, nil)
	ps.EXPECT().AssignRole(ctx, "u-google", "ws-google", "admin").Return(true, nil)
	repo.EXPECT().CreateSession(ctx, gomock.Any()).Return(nil)
	repo.EXPECT().CreateToken(ctx, gomock.Any()).Times(2).Return(nil)

	user, tokens, err := svc.OAuthCallback(ctx, "google", code, state)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if user.ID != "u-google" || tokens.AccessToken == "" {
		t.Errorf("failed to map token profile outputs")
	}
}
