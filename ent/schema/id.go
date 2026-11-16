package schema

import "github.com/google/uuid"

// newID generates a new creation-ordered UUID v7 for entity IDs.
func newID() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}
