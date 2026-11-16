package domain

import "time"

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
