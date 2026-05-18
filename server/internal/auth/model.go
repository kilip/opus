package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents a user domain model.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	AvatarURL    string    `json:"avatar_url"`
	Provider     string    `json:"provider"`
	ProviderID   string    `json:"provider_id"`
	WorkspaceID  string    `json:"workspace_id"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Account represents the third-party account linkage.
type Account struct {
	ID                   string     `json:"id"`
	UserID               string     `json:"user_id"`
	AccountID            string     `json:"account_id"`
	ProviderID           string     `json:"provider_id"`
	AccessToken          string     `json:"access_token"`
	RefreshToken         string     `json:"refresh_token"`
	AccessTokenExpiresAt *time.Time `json:"access_token_expires_at"`
	Scope                string     `json:"scope"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// Session groups tokens together inside a workspace context.
type Session struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	WorkspaceID string    `json:"workspace_id"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	CreatedAt   time.Time `json:"created_at"`
}

// Token represents a single stateful access or refresh token record.
type Token struct {
	ID        string     `json:"id"`
	SessionID string     `json:"session_id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"` // "access" or "refresh"
	Hash      string     `json:"hash"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// OAuthProfile contains parsed values from standard provider callback responses.
type OAuthProfile struct {
	ProviderID string `json:"provider_id"`
	Provider   string `json:"provider"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatar_url"`
}

// Claims represents the cryptographically signed access token payload.
type Claims struct {
	SessionID   string `json:"sid"`
	WorkspaceID string `json:"wid"`
	Role        string `json:"role"`
	jwt.RegisteredClaims
}
