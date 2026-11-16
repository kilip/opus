package domain

// Event type constants for node domain events.
const (
	EventOnline  = "node.online"
	EventOffline = "node.offline"
)

// NodeEventPayload is the event payload for node status changes.
type NodeEventPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
