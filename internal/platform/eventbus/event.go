package eventbus

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event represents a domain event published on the event bus.
type Event struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Subject string          `json:"subject"`
	At      time.Time       `json:"at"`
	Payload json.RawMessage `json:"payload"`
}

// NewEvent creates a new Event with a UUIDv7 ID and current UTC timestamp.
// If payload is already json.RawMessage or []byte, it is copied.
// Otherwise, it is marshaled to JSON.
func NewEvent(eventType, subject string, payload any) (Event, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Event{}, err
	}

	var raw json.RawMessage
	switch p := payload.(type) {
	case nil:
		raw = nil
	case json.RawMessage:
		raw = bytes.Clone(p)
	case []byte:
		raw = bytes.Clone(p)
	default:
		b, err := json.Marshal(p)
		if err != nil {
			return Event{}, err
		}
		raw = b
	}

	return Event{
		ID:      id.String(),
		Type:    eventType,
		Subject: subject,
		At:      time.Now().UTC(),
		Payload: raw,
	}, nil
}

// Decode unmarshals the event payload into T without exposing the shared Payload slice for mutation.
func Decode[T any](e Event) (T, error) {
	var zero T
	if len(e.Payload) == 0 {
		return zero, nil
	}

	if raw, ok := any(&zero).(*json.RawMessage); ok {
		*raw = bytes.Clone(e.Payload)
		return zero, nil
	}

	if b, ok := any(&zero).(*[]byte); ok {
		*b = bytes.Clone(e.Payload)
		return zero, nil
	}

	cloned := bytes.Clone(e.Payload)
	if err := json.Unmarshal(cloned, &zero); err != nil {
		var empty T
		return empty, err
	}
	return zero, nil
}
