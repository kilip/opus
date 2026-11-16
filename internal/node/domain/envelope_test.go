package domain_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/kilip/opus/internal/node/domain"
)

func TestEnvelope_RoundTripTypes(t *testing.T) {
	types := []string{
		domain.EnvelopeTypeHello,
		domain.EnvelopeTypeHeartbeat,
		domain.EnvelopeTypeCommand,
		domain.EnvelopeTypeEvent,
	}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			payload := json.RawMessage(`{"key":"value"}`)
			original := domain.Envelope{
				Type:    typ,
				ID:      "test-id-" + typ,
				Payload: payload,
			}

			encoded, err := original.Encode()
			if err != nil {
				t.Fatalf("unexpected encode error for %s: %v", typ, err)
			}

			decoded, err := domain.DecodeEnvelope(encoded)
			if err != nil {
				t.Fatalf("unexpected decode error for %s: %v", typ, err)
			}

			if decoded.Type != original.Type {
				t.Errorf("expected type %q, got %q", original.Type, decoded.Type)
			}
			if decoded.ID != original.ID {
				t.Errorf("expected id %q, got %q", original.ID, decoded.ID)
			}
			if !bytes.Equal(decoded.Payload, original.Payload) {
				t.Errorf("expected payload %s, got %s", string(original.Payload), string(decoded.Payload))
			}
		})
	}
}

func TestEnvelope_EmptyIDAndPayload(t *testing.T) {
	env := domain.Envelope{
		Type: domain.EnvelopeTypeHeartbeat,
	}
	encoded, err := env.Encode()
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}

	// id and payload should be omitted
	expectedJSON := `{"type":"heartbeat"}`
	if string(encoded) != expectedJSON {
		t.Errorf("expected %s, got %s", expectedJSON, string(encoded))
	}

	decoded, err := domain.DecodeEnvelope(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if decoded.Type != domain.EnvelopeTypeHeartbeat {
		t.Errorf("expected heartbeat, got %s", decoded.Type)
	}
	if decoded.ID != "" {
		t.Errorf("expected empty ID, got %s", decoded.ID)
	}
	if decoded.Payload != nil {
		t.Errorf("expected nil Payload, got %s", string(decoded.Payload))
	}
}

func TestEnvelope_UnknownTypeRejected(t *testing.T) {
	// Encode unknown type
	env := domain.Envelope{
		Type: "unknown_type",
	}
	_, err := env.Encode()
	if !errors.Is(err, domain.ErrUnknownEnvelopeType) {
		t.Fatalf("expected ErrUnknownEnvelopeType on Encode, got %v", err)
	}

	// Decode unknown type
	raw := []byte(`{"type":"invalid_foo","id":"123"}`)
	_, err = domain.DecodeEnvelope(raw)
	if !errors.Is(err, domain.ErrUnknownEnvelopeType) {
		t.Fatalf("expected ErrUnknownEnvelopeType on DecodeEnvelope, got %v", err)
	}
}

func TestEnvelope_MalformedJSON(t *testing.T) {
	raw := []byte(`{not valid json}`)
	_, err := domain.DecodeEnvelope(raw)
	if err == nil {
		t.Fatal("expected error on malformed json, got nil")
	}
}
