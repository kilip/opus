package domain

import "time"

// NodeResponse is the wire representation of a node in the registry.
type NodeResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Role       Role       `json:"role"`
	Status     Status     `json:"status"`
	LastSeenAt *time.Time `json:"lastSeenAt"`
	Hostname   *string    `json:"hostname,omitempty"`
	IPAddress  *string    `json:"ipAddress,omitempty"`
	MACAddress *string    `json:"macAddress,omitempty"`
}

// UpdateNodeRequest holds optional fields to update on a node.
type UpdateNodeRequest struct {
	Name       *string `json:"name,omitempty"`
	IPAddress  *string `json:"ipAddress,omitempty"`
	MACAddress *string `json:"macAddress,omitempty"`
}

// JoinTokenResponse represents the response when a node join token is created.
type JoinTokenResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// JoinNodeRequest holds parameters for a node joining the cluster.
type JoinNodeRequest struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

// JoinNodeResponse is returned when a node successfully joins the cluster.
type JoinNodeResponse struct {
	NodeID     string `json:"nodeId"`
	Credential string `json:"credential"`
}
