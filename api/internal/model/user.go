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
