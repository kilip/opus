package domain

import (
	"regexp"
	"time"
)

var nodeNameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]{1,63}$`)

// ValidateNodeName checks that name is 1 to 63 characters of [A-Za-z0-9._-].
func ValidateNodeName(name string) error {
	if !nodeNameRegex.MatchString(name) {
		return ErrInvalidNodeName
	}
	return nil
}

// Role defines the role of a node in the cluster.
type Role string

const (
	RoleMaster Role = "master"
	RoleWorker Role = "worker"
)

// Status defines the operational status of a node.
type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
)

// Node represents a registered node in the domain.
type Node struct {
	ID             string
	Name           string
	Role           Role
	Status         Status
	LastSeenAt     *time.Time
	CredentialHash *string
}
