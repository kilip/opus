package domain

import (
	"encoding/json"
	"fmt"
)

// Envelope type constants.
const (
	EnvelopeTypeHello     = "hello"
	EnvelopeTypeHeartbeat = "heartbeat"
	EnvelopeTypeCommand   = "command"
	EnvelopeTypeEvent     = "event"
)

// IsValidEnvelopeType returns true if t is one of the supported envelope types.
func IsValidEnvelopeType(t string) bool {
	switch t {
	case EnvelopeTypeHello, EnvelopeTypeHeartbeat, EnvelopeTypeCommand, EnvelopeTypeEvent:
		return true
	default:
		return false
	}
}

// Envelope is the standard JSON frame for control channel messages.
type Envelope struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ToolStatus represents the availability state of a local worker tool.
type ToolStatus string

const (
	ToolStatusReady           ToolStatus = "ready"
	ToolStatusUnauthenticated ToolStatus = "unauthenticated"
	ToolStatusMissing         ToolStatus = "missing"
)

// Capabilities reports local tool availability on a worker node.
type Capabilities struct {
	Agy ToolStatus `json:"agy"`
	Gh  ToolStatus `json:"gh"`
}

// HelloPayload is the payload carried by a hello envelope.
type HelloPayload struct {
	NodeID       string       `json:"nodeId,omitempty"`
	Capabilities Capabilities `json:"capabilities"`
}

// Encode serializes the Envelope to JSON bytes after validating the envelope type.
func (e Envelope) Encode() ([]byte, error) {
	if !IsValidEnvelopeType(e.Type) {
		return nil, fmt.Errorf("%w: %s", ErrUnknownEnvelopeType, e.Type)
	}
	return json.Marshal(e)
}

// DecodeEnvelope deserializes JSON data into an Envelope and validates its type.
func DecodeEnvelope(data []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return Envelope{}, err
	}
	if !IsValidEnvelopeType(env.Type) {
		return env, fmt.Errorf("%w: %s", ErrUnknownEnvelopeType, env.Type)
	}
	return env, nil
}
