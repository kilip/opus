package testutil

import (
	"testing"

	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
)

// NewTestEntClient returns a new test Ent client.
func NewTestEntClient(t *testing.T) *ent.Client {
	t.Helper()
	return enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
}
