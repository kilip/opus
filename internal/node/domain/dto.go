package domain

import "time"

// NodeResponse is the wire representation of a node in the registry.
type NodeResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Role       Role       `json:"role"`
	Status     Status     `json:"status"`
	LastSeenAt *time.Time `json:"lastSeenAt"`
}
