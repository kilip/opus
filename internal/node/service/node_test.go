package service_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	entnode "github.com/kilip/opus/ent/node"
	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/repo"
	"github.com/kilip/opus/internal/node/service"
	_ "modernc.org/sqlite"
)

func setupTestClient(t *testing.T) *ent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	db.SetMaxOpenConns(1)
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}

// AC2: On app startup exactly one node with role master exists; its name is the hostname
// and its ID is generated once and persisted (restarting the app twice leaves one master row with the same ID).
func TestBootstrap_MasterSelfRegistrationIdempotency(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	svc := service.New(r)
	ctx := context.Background()

	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "master"
	}

	// 1st startup bootstrap
	if err := svc.Bootstrap(ctx); err != nil {
		t.Fatalf("first bootstrap failed: %v", err)
	}

	masters, err := client.Node.Query().
		Where(entnode.RoleEQ(entnode.RoleMaster)).
		All(ctx)
	if err != nil {
		t.Fatalf("query masters: %v", err)
	}
	if len(masters) != 1 {
		t.Fatalf("expected 1 master node, got %d", len(masters))
	}
	m1 := masters[0]
	if m1.Name != hostname {
		t.Errorf("expected master name %q, got %q", hostname, m1.Name)
	}
	if m1.Status != entnode.StatusOnline {
		t.Errorf("expected master status online, got %q", m1.Status)
	}
	if m1.CredentialHash != nil {
		t.Errorf("expected master credential_hash to be nil, got %v", m1.CredentialHash)
	}

	// 2nd startup bootstrap (simulating restart)
	if err := svc.Bootstrap(ctx); err != nil {
		t.Fatalf("second bootstrap failed: %v", err)
	}

	mastersAfter, err := client.Node.Query().
		Where(entnode.RoleEQ(entnode.RoleMaster)).
		All(ctx)
	if err != nil {
		t.Fatalf("query masters after restart: %v", err)
	}
	if len(mastersAfter) != 1 {
		t.Fatalf("expected 1 master node after restart, got %d", len(mastersAfter))
	}
	m2 := mastersAfter[0]
	if m2.ID != m1.ID {
		t.Errorf("expected master ID to persist across restarts: got %v, want %v", m2.ID, m1.ID)
	}
	if m2.Name != hostname {
		t.Errorf("expected master name %q, got %q", hostname, m2.Name)
	}
	if m2.Status != entnode.StatusOnline {
		t.Errorf("expected master status online, got %q", m2.Status)
	}
}

// AC3: On startup every node with role worker is set to offline.
func TestBootstrap_WorkerOfflineReset(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	svc := service.New(r)
	ctx := context.Background()

	// Pre-populate worker nodes: one online, one offline
	_, err := client.Node.Create().
		SetName("worker-1").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOnline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker-1: %v", err)
	}

	_, err = client.Node.Create().
		SetName("worker-2").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker-2: %v", err)
	}

	// Run bootstrap
	if err := svc.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	// Verify all workers are now offline
	workers, err := client.Node.Query().
		Where(entnode.RoleEQ(entnode.RoleWorker)).
		All(ctx)
	if err != nil {
		t.Fatalf("query workers: %v", err)
	}
	if len(workers) != 2 {
		t.Fatalf("expected 2 workers, got %d", len(workers))
	}
	for _, w := range workers {
		if w.Status != entnode.StatusOffline {
			t.Errorf("expected worker %s status to be offline, got %q", w.Name, w.Status)
		}
	}

	// Verify master is still online
	master, err := client.Node.Query().
		Where(entnode.RoleEQ(entnode.RoleMaster)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query master: %v", err)
	}
	if master.Status != entnode.StatusOnline {
		t.Errorf("expected master status to be online, got %q", master.Status)
	}
}

func TestList_EmptyAndNonEmpty(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	svc := service.New(r)
	ctx := context.Background()

	// Initial empty list
	nodes, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("expected empty nodes, got %d", len(nodes))
	}

	// Bootstrap creates master
	if err := svc.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	nodes, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("list nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Role != domain.RoleMaster {
		t.Errorf("expected master role, got %v", nodes[0].Role)
	}
}
