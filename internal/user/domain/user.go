package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the domain.
type User struct {
	ID        uuid.UUID
	Email     string
	CreatedAt time.Time
}

// NormalizeEmail normalises an email by trimming whitespace and converting to lowercase.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
