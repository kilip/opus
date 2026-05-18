package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/kilip/opus/server/ent"
	"github.com/kilip/opus/server/ent/enttest"
	_ "github.com/mattn/go-sqlite3"
)

// NewTestEntClient returns a new test Ent client with a unique memory database.
func NewTestEntClient(t *testing.T) *ent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared&_fk=1", t.Name(), time.Now().UnixNano())
	return enttest.Open(t, "sqlite3", dsn)
}
