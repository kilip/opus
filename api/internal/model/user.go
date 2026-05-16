package model

import "time"

type User struct {
	ID         string
	Email      string
	Name       string
	AvatarURL  string
	Provider   string
	ProviderID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsSocialLogin returns true if the user logged in via an OAuth provider
func (u *User) IsSocialLogin() bool {
	return u.Provider != ""
}
