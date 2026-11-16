package service_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/ent/jointoken"
	entnode "github.com/kilip/opus/ent/node"

	"github.com/kilip/opus/internal/node/domain"
	"github.com/kilip/opus/internal/node/repo"
	"github.com/kilip/opus/internal/node/service"
	_ "modernc.org/sqlite"
)

func setupTestClient(t *testing.T) *ent.Client {
	t.Helper()
	client, _ := setupTestClientWithDB(t)
	return client
}

func setupTestClientWithDB(t *testing.T) (*ent.Client, *sql.DB) {
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
		_ = db.Close()
	})
	return client, db
}

// AC2: On app startup exactly one node with role master exists; its name is the hostname
// and its ID is generated once and persisted (restarting the app twice leaves one master row with the same ID).
func TestBootstrap_MasterSelfRegistrationIdempotency(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	svc := service.New(r, service.Config{})
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
	svc := service.New(r, service.Config{})
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
	svc := service.New(r, service.Config{})
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

// AC3: Only the hash is stored: raw token appears nowhere in the database.
func TestGenerateJoinToken_RawTokenNeverStored(t *testing.T) {
	client, db := setupTestClientWithDB(t)
	r := repo.New(client)
	svc := service.New(r, service.Config{
		BaseURL: "http://localhost:7880",
	})
	ctx := context.Background()

	res, err := svc.GenerateJoinToken(ctx)
	if err != nil {
		t.Fatalf("GenerateJoinToken failed: %v", err)
	}

	const prefix = "http://localhost:7880/join/"
	if !strings.HasPrefix(res.URL, prefix) {
		t.Fatalf("expected URL prefix %s, got %s", prefix, res.URL)
	}
	rawToken := strings.TrimPrefix(res.URL, prefix)
	if len(rawToken) == 0 {
		t.Fatalf("empty raw token")
	}

	h := sha256.Sum256([]byte(rawToken))
	expectedHash := hex.EncodeToString(h[:])

	tokens, err := client.JoinToken.Query().All(ctx)
	if err != nil {
		t.Fatalf("query join tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 join token, got %d", len(tokens))
	}

	tok := tokens[0]
	if tok.TokenHash != expectedHash {
		t.Errorf("expected token_hash %q, got %q", expectedHash, tok.TokenHash)
	}
	if tok.TokenHash == rawToken {
		t.Errorf("token_hash must not equal raw token")
	}
	if tok.UsedAt != nil {
		t.Errorf("expected used_at to be nil")
	}

	// Assert raw token is nowhere in the join_tokens table
	rows, err := db.QueryContext(ctx, "SELECT id, token_hash FROM join_tokens")
	if err != nil {
		t.Fatalf("query raw join_tokens: %v", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id, hash string
		if err := rows.Scan(&id, &hash); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		if strings.Contains(id, rawToken) || strings.Contains(hash, rawToken) {
			t.Errorf("raw token found in database row: id=%s hash=%s", id, hash)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}
}

// AC4: ConsumeJoinToken validates and consumes a token:
// an expired token, an already used token and an unknown token each return a distinct sentinel error;
// a valid token is accepted once. Table-driven test covers all four cases using an injectable clock.
func TestConsumeJoinToken_Validation(t *testing.T) {
	fixedNow := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		setup         func(ctx context.Context, svc *service.Node, curTime *time.Time) string
		expectedErr   error
		verifyConsume func(t *testing.T, ctx context.Context, svc *service.Node, token string)
	}{
		{
			name: "valid token accepted once",
			setup: func(ctx context.Context, svc *service.Node, curTime *time.Time) string {
				*curTime = fixedNow
				res, err := svc.GenerateJoinToken(ctx)
				if err != nil {
					t.Fatalf("GenerateJoinToken: %v", err)
				}
				*curTime = fixedNow.Add(5 * time.Minute) // 5m later (valid, TTL is 15m)
				return strings.TrimPrefix(res.URL, "http://localhost:7880/join/")
			},
			expectedErr: nil,
			verifyConsume: func(t *testing.T, ctx context.Context, svc *service.Node, token string) {
				// Second consumption must fail with ErrTokenAlreadyUsed
				err := svc.ConsumeJoinToken(ctx, token)
				if !errors.Is(err, domain.ErrTokenAlreadyUsed) {
					t.Errorf("second consume: expected ErrTokenAlreadyUsed, got %v", err)
				}
			},
		},
		{
			name: "expired token returns ErrTokenExpired",
			setup: func(ctx context.Context, svc *service.Node, curTime *time.Time) string {
				*curTime = fixedNow
				res, err := svc.GenerateJoinToken(ctx)
				if err != nil {
					t.Fatalf("GenerateJoinToken: %v", err)
				}
				*curTime = fixedNow.Add(16 * time.Minute) // 16m later (> 15m TTL)
				return strings.TrimPrefix(res.URL, "http://localhost:7880/join/")
			},
			expectedErr: domain.ErrTokenExpired,
		},
		{
			name: "already used token returns ErrTokenAlreadyUsed",
			setup: func(ctx context.Context, svc *service.Node, curTime *time.Time) string {
				*curTime = fixedNow
				res, err := svc.GenerateJoinToken(ctx)
				if err != nil {
					t.Fatalf("GenerateJoinToken: %v", err)
				}
				token := strings.TrimPrefix(res.URL, "http://localhost:7880/join/")
				*curTime = fixedNow.Add(1 * time.Minute)
				if err := svc.ConsumeJoinToken(ctx, token); err != nil {
					t.Fatalf("initial consume failed: %v", err)
				}
				*curTime = fixedNow.Add(2 * time.Minute)
				return token
			},
			expectedErr: domain.ErrTokenAlreadyUsed,
		},
		{
			name: "unknown token returns ErrTokenNotFound",
			setup: func(ctx context.Context, svc *service.Node, curTime *time.Time) string {
				*curTime = fixedNow
				return "unknown-random-token-not-in-db"
			},
			expectedErr: domain.ErrTokenNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := setupTestClient(t)
			r := repo.New(client)
			var curTime time.Time
			svc := service.New(r, service.Config{
				BaseURL: "http://localhost:7880",
				Now: func() time.Time {
					return curTime
				},
			})
			ctx := context.Background()

			token := tc.setup(ctx, svc, &curTime)
			err := svc.ConsumeJoinToken(ctx, token)

			if tc.expectedErr == nil {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
			} else {
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
				}
			}

			if tc.verifyConsume != nil {
				tc.verifyConsume(t, ctx, svc, token)
			}
		})
	}
}

func TestJoin_Success(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	svc := service.New(r, service.Config{
		BaseURL: "http://localhost:7880",
		Now:     func() time.Time { return now },
	})
	ctx := context.Background()

	res, err := svc.GenerateJoinToken(ctx)
	if err != nil {
		t.Fatalf("GenerateJoinToken failed: %v", err)
	}
	rawToken := strings.TrimPrefix(res.URL, "http://localhost:7880/join/")

	joinRes, err := svc.Join(ctx, domain.JoinNodeRequest{
		Token: rawToken,
		Name:  "worker-1",
	})
	if err != nil {
		t.Fatalf("Join failed: %v", err)
	}

	if joinRes.NodeID == "" {
		t.Errorf("expected non-empty NodeID")
	}
	if joinRes.Credential == "" {
		t.Errorf("expected non-empty Credential")
	}

	// Verify node in DB
	n, err := client.Node.Query().Where(entnode.NameEQ("worker-1")).Only(ctx)
	if err != nil {
		t.Fatalf("query node failed: %v", err)
	}
	if n.ID.String() != joinRes.NodeID {
		t.Errorf("node ID mismatch: got %s, want %s", n.ID.String(), joinRes.NodeID)
	}
	if n.Name != "worker-1" {
		t.Errorf("node name = %s, want worker-1", n.Name)
	}
	if n.Role != entnode.RoleWorker {
		t.Errorf("node role = %s, want worker", n.Role)
	}
	if n.Status != entnode.StatusOffline {
		t.Errorf("node status = %s, want offline", n.Status)
	}
	if n.CredentialHash == nil {
		t.Fatalf("expected credential hash to be set")
	}

	expectedCredHash := sha256.Sum256([]byte(joinRes.Credential))
	if *n.CredentialHash != hex.EncodeToString(expectedCredHash[:]) {
		t.Errorf("credential hash mismatch: got %s, want %s", *n.CredentialHash, hex.EncodeToString(expectedCredHash[:]))
	}

	// Replaying the same token must fail
	_, err = svc.Join(ctx, domain.JoinNodeRequest{
		Token: rawToken,
		Name:  "worker-2",
	})
	if !errors.Is(err, domain.ErrTokenAlreadyUsed) {
		t.Errorf("expected ErrTokenAlreadyUsed on replay, got %v", err)
	}
}

// AC2: The token is marked used in the same database transaction as the node insert;
// replaying the same token returns an error. A test forces the node insert to fail and asserts the token stays unused.
func TestJoin_TransactionRollbackOnConflict(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	svc := service.New(r, service.Config{
		BaseURL: "http://localhost:7880",
		Now:     func() time.Time { return now },
	})
	ctx := context.Background()

	// Pre-create a node with name "conflict-worker"
	_, err := client.Node.Create().
		SetName("conflict-worker").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		Save(ctx)
	if err != nil {
		t.Fatalf("create initial node: %v", err)
	}

	// Generate join token
	res, err := svc.GenerateJoinToken(ctx)
	if err != nil {
		t.Fatalf("GenerateJoinToken failed: %v", err)
	}
	rawToken := strings.TrimPrefix(res.URL, "http://localhost:7880/join/")

	// Force node insert to fail with name conflict
	_, err = svc.Join(ctx, domain.JoinNodeRequest{
		Token: rawToken,
		Name:  "conflict-worker",
	})
	if !errors.Is(err, domain.ErrNodeNameConflict) {
		t.Fatalf("expected ErrNodeNameConflict, got %v", err)
	}

	// Assert the token stays unused in the database
	tokHash := sha256.Sum256([]byte(rawToken))
	tokHex := hex.EncodeToString(tokHash[:])
	tok, err := client.JoinToken.Query().
		Where(jointoken.TokenHashEQ(tokHex)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query join token: %v", err)
	}
	if tok.UsedAt != nil {
		t.Fatalf("expected token to remain unused (used_at == nil), got used_at=%v", tok.UsedAt)
	}

	// Joining with the same token and a valid name must now succeed
	joinRes, err := svc.Join(ctx, domain.JoinNodeRequest{
		Token: rawToken,
		Name:  "valid-worker",
	})
	if err != nil {
		t.Fatalf("join with same token and valid name failed: %v", err)
	}
	if joinRes.NodeID == "" || joinRes.Credential == "" {
		t.Fatalf("invalid join response: %+v", joinRes)
	}

	// Token must now be marked used
	tokAfter, err := client.JoinToken.Query().
		Where(jointoken.TokenHashEQ(tokHex)).
		Only(ctx)
	if err != nil {
		t.Fatalf("query join token after: %v", err)
	}
	if tokAfter.UsedAt == nil {
		t.Fatalf("expected token to be marked used")
	}

	// Replaying the token now fails with ErrTokenAlreadyUsed
	_, err = svc.Join(ctx, domain.JoinNodeRequest{
		Token: rawToken,
		Name:  "worker-replay",
	})
	if !errors.Is(err, domain.ErrTokenAlreadyUsed) {
		t.Fatalf("expected ErrTokenAlreadyUsed on replay, got %v", err)
	}
}

func TestJoin_Validation(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	svc := service.New(r, service.Config{
		BaseURL: "http://localhost:7880",
		Now:     func() time.Time { return now },
	})
	ctx := context.Background()

	tests := []struct {
		name        string
		req         domain.JoinNodeRequest
		expectedErr error
	}{
		{
			name:        "empty name",
			req:         domain.JoinNodeRequest{Token: "some-tok", Name: ""},
			expectedErr: domain.ErrInvalidNodeName,
		},
		{
			name:        "invalid chars in name",
			req:         domain.JoinNodeRequest{Token: "some-tok", Name: "bad name!"},
			expectedErr: domain.ErrInvalidNodeName,
		},
		{
			name:        "name too long",
			req:         domain.JoinNodeRequest{Token: "some-tok", Name: strings.Repeat("a", 64)},
			expectedErr: domain.ErrInvalidNodeName,
		},
		{
			name:        "empty token",
			req:         domain.JoinNodeRequest{Token: "", Name: "valid-name"},
			expectedErr: domain.ErrTokenNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Join(ctx, tc.req)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestAuthenticateWorker(t *testing.T) {
	client := setupTestClient(t)
	r := repo.New(client)
	svc := service.New(r, service.Config{})
	ctx := context.Background()

	// 1. Missing / empty cred
	_, err := svc.AuthenticateWorker(ctx, "")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for empty cred, got %v", err)
	}

	// 2. Unknown cred
	_, err = svc.AuthenticateWorker(ctx, "nonexistent-cred")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for unknown cred, got %v", err)
	}

	// 3. Worker node with credential
	workerCred := "valid-worker-cred"
	h := sha256.Sum256([]byte(workerCred))
	workerNode, err := client.Node.Create().
		SetName("worker-test-auth").
		SetRole(entnode.RoleWorker).
		SetStatus(entnode.StatusOffline).
		SetCredentialHash(hex.EncodeToString(h[:])).
		Save(ctx)
	if err != nil {
		t.Fatalf("create worker: %v", err)
	}

	node, err := svc.AuthenticateWorker(ctx, workerCred)
	if err != nil {
		t.Fatalf("unexpected authenticate error: %v", err)
	}
	if node.ID != workerNode.ID.String() {
		t.Errorf("expected node ID %s, got %s", workerNode.ID.String(), node.ID)
	}

	// 4. Master node with credential attempt -> ErrMasterConnectForbidden
	masterCred := "master-cred-secret"
	mh := sha256.Sum256([]byte(masterCred))
	_, err = client.Node.Create().
		SetName("master-test-auth").
		SetRole(entnode.RoleMaster).
		SetStatus(entnode.StatusOnline).
		SetCredentialHash(hex.EncodeToString(mh[:])).
		Save(ctx)
	if err != nil {
		t.Fatalf("create master: %v", err)
	}

	_, err = svc.AuthenticateWorker(ctx, masterCred)
	if !errors.Is(err, domain.ErrMasterConnectForbidden) {
		t.Fatalf("expected ErrMasterConnectForbidden for master node, got %v", err)
	}
}
