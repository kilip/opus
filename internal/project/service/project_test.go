package service_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/project/domain"
	"github.com/kilip/opus/internal/project/repo"
	"github.com/kilip/opus/internal/project/service"
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

func newService(t *testing.T, cfg ...service.Config) (*service.Project, context.Context) {
	t.Helper()
	client := setupTestClient(t)
	r := repo.New(client)
	return service.New(r, cfg...), context.Background()
}

func TestService_CreateValidation(t *testing.T) {
	svc, ctx := newService(t)

	tests := []struct {
		name    string
		req     domain.CreateProjectRequest
		wantErr error
	}{
		{
			name: "empty slug",
			req: domain.CreateProjectRequest{
				Slug:         "",
				Name:         "Opus",
				RepoProvider: "github",
				RepoRef:      "owner/repo",
			},
			wantErr: domain.ErrInvalidSlug,
		},
		{
			name: "uppercase in slug",
			req: domain.CreateProjectRequest{
				Slug:         "MyProject",
				Name:         "Opus",
				RepoProvider: "github",
				RepoRef:      "owner/repo",
			},
			wantErr: domain.ErrInvalidSlug,
		},
		{
			name: "slug starts with hyphen",
			req: domain.CreateProjectRequest{
				Slug:         "-project",
				Name:         "Opus",
				RepoProvider: "github",
				RepoRef:      "owner/repo",
			},
			wantErr: domain.ErrInvalidSlug,
		},
		{
			name: "slug with invalid character",
			req: domain.CreateProjectRequest{
				Slug:         "my_project",
				Name:         "Opus",
				RepoProvider: "github",
				RepoRef:      "owner/repo",
			},
			wantErr: domain.ErrInvalidSlug,
		},
		{
			name: "slug too long (64 chars)",
			req: domain.CreateProjectRequest{
				Slug:         strings.Repeat("a", 64),
				Name:         "Opus",
				RepoProvider: "github",
				RepoRef:      "owner/repo",
			},
			wantErr: domain.ErrInvalidSlug,
		},
		{
			name: "invalid repo provider",
			req: domain.CreateProjectRequest{
				Slug:         "my-proj",
				Name:         "Opus",
				RepoProvider: "gitlab",
				RepoRef:      "owner/repo",
			},
			wantErr: domain.ErrInvalidRepo,
		},
		{
			name: "empty repo provider",
			req: domain.CreateProjectRequest{
				Slug:         "my-proj",
				Name:         "Opus",
				RepoProvider: "",
				RepoRef:      "owner/repo",
			},
			wantErr: domain.ErrInvalidRepo,
		},
		{
			name: "invalid repo ref - no slash",
			req: domain.CreateProjectRequest{
				Slug:         "my-proj",
				Name:         "Opus",
				RepoProvider: "github",
				RepoRef:      "ownerrepo",
			},
			wantErr: domain.ErrInvalidRepo,
		},
		{
			name: "invalid repo ref - too many slashes",
			req: domain.CreateProjectRequest{
				Slug:         "my-proj",
				Name:         "Opus",
				RepoProvider: "github",
				RepoRef:      "owner/repo/extra",
			},
			wantErr: domain.ErrInvalidRepo,
		},
		{
			name: "invalid repo ref - empty owner",
			req: domain.CreateProjectRequest{
				Slug:         "my-proj",
				Name:         "Opus",
				RepoProvider: "github",
				RepoRef:      "/repo",
			},
			wantErr: domain.ErrInvalidRepo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Create(ctx, tt.req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestService_CreateDefaulting(t *testing.T) {
	svc, ctx := newService(t)

	p, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "opus-app",
		Name:         "Opus App",
		RepoProvider: "github",
		RepoRef:      "kilip/opus",
	})
	if err != nil {
		t.Fatalf("unexpected error creating project: %v", err)
	}

	if p.TrackerProvider != "github" {
		t.Errorf("expected tracker_provider to default to %q, got %q", "github", p.TrackerProvider)
	}
	if p.TrackerRef != "kilip/opus" {
		t.Errorf("expected tracker_ref to default to %q, got %q", "kilip/opus", p.TrackerRef)
	}
	if p.DefaultBranch != "main" {
		t.Errorf("expected default_branch to default to %q, got %q", "main", p.DefaultBranch)
	}
}

func TestService_DuplicateSlug(t *testing.T) {
	svc, ctx := newService(t)

	_, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "unique-slug",
		Name:         "Project One",
		RepoProvider: "github",
		RepoRef:      "owner/one",
	})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "unique-slug",
		Name:         "Project Two",
		RepoProvider: "github",
		RepoRef:      "owner/two",
	})
	if !errors.Is(err, domain.ErrDuplicateSlug) {
		t.Fatalf("expected ErrDuplicateSlug, got %v", err)
	}
}

func TestService_DeleteWithTaskCounterHook(t *testing.T) {
	taskCounts := make(map[string]int)
	tc := func(_ context.Context, projectID string) (int, error) {
		return taskCounts[projectID], nil
	}

	client := setupTestClient(t)
	r := repo.New(client)
	svc := service.New(r, service.Config{TaskCounter: tc})
	ctx := context.Background()

	p, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "test-delete",
		Name:         "Test Delete",
		RepoProvider: "github",
		RepoRef:      "owner/repo",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 1. Refuse delete when tasks > 0 (409 conflict domain sentinel)
	taskCounts[p.ID] = 3
	err = svc.Delete(ctx, p.ID)
	if !errors.Is(err, domain.ErrProjectHasTasks) {
		t.Fatalf("expected ErrProjectHasTasks, got %v", err)
	}

	// Project still exists
	if _, err := svc.Get(ctx, p.ID); err != nil {
		t.Fatalf("expected project to exist, got %v", err)
	}

	// 2. Allow delete when tasks == 0
	taskCounts[p.ID] = 0
	err = svc.Delete(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected error deleting project: %v", err)
	}

	// 3. Subsequent delete or get returns ErrProjectNotFound
	if _, err := svc.Get(ctx, p.ID); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound on get, got %v", err)
	}
	if err := svc.Delete(ctx, p.ID); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound on second delete, got %v", err)
	}
}

func TestService_GetAndUpdate(t *testing.T) {
	svc, ctx := newService(t)

	// Unknown ID returns ErrProjectNotFound
	if _, err := svc.Get(ctx, "01933a3c-1b77-7000-8000-000000000000"); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound for unknown UUID, got %v", err)
	}

	// Malformed ID returns ErrProjectNotFound
	if _, err := svc.Get(ctx, "malformed-id"); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound for malformed ID, got %v", err)
	}

	p, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "update-test",
		Name:         "Old Name",
		RepoProvider: "github",
		RepoRef:      "owner/repo",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	newName := "New Name"
	newBranch := "develop"
	newTrackerProvider := "jira"
	newTrackerRef := "PROJ"
	updated, err := svc.Update(ctx, p.ID, domain.UpdateProjectRequest{
		Name:            &newName,
		DefaultBranch:   &newBranch,
		TrackerProvider: &newTrackerProvider,
		TrackerRef:      &newTrackerRef,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}
	if updated.DefaultBranch != newBranch {
		t.Errorf("expected default_branch %q, got %q", newBranch, updated.DefaultBranch)
	}
	if updated.TrackerProvider != newTrackerProvider {
		t.Errorf("expected tracker_provider %q, got %q", newTrackerProvider, updated.TrackerProvider)
	}
	if updated.TrackerRef != newTrackerRef {
		t.Errorf("expected tracker_ref %q, got %q", newTrackerRef, updated.TrackerRef)
	}
	// Slug is unchanged
	if updated.Slug != "update-test" {
		t.Errorf("expected slug to remain unchanged, got %q", updated.Slug)
	}
}

func TestService_ListNodeMappings(t *testing.T) {
	client := setupTestClient(t)
	ctx := context.Background()
	r := repo.New(client)
	nodeSvc := node.New(client, node.Config{})
	svc := service.New(r, service.Config{NodeReader: nodeSvc})

	// 1. Unknown project returns ErrProjectNotFound
	_, err := svc.ListNodeMappings(ctx, "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}

	// 2. Create project and nodes
	p, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "list-nodes-proj",
		Name:         "List Nodes Proj",
		RepoProvider: "github",
		RepoRef:      "org/list-nodes-proj",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	n1, err := client.Node.Create().
		SetName("worker-1").
		SetRole("worker").
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create node 1: %v", err)
	}

	n2, err := client.Node.Create().
		SetName("worker-2").
		SetRole("worker").
		SetStatus("offline").
		Save(ctx)
	if err != nil {
		t.Fatalf("create node 2: %v", err)
	}

	// Before any mapping is set, all nodes appear with path: nil
	mappings, err := svc.ListNodeMappings(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListNodeMappings: %v", err)
	}
	if len(mappings) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(mappings))
	}
	for _, m := range mappings {
		if m.Path != nil {
			t.Errorf("expected nil path for unmapped node %s, got %v", m.NodeID, *m.Path)
		}
	}

	// Set path for n1
	path1 := "/var/repo/p"
	_, err = svc.SetNodePath(ctx, p.ID, n1.ID.String(), &path1)
	if err != nil {
		t.Fatalf("SetNodePath: %v", err)
	}

	// Check mappings again
	mappings, err = svc.ListNodeMappings(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListNodeMappings: %v", err)
	}
	var foundN1, foundN2 bool
	for _, m := range mappings {
		if m.NodeID == n1.ID.String() {
			foundN1 = true
			if m.Path == nil || *m.Path != path1 {
				t.Errorf("n1 path = %v, want %q", m.Path, path1)
			}
			if m.NodeName != "worker-1" {
				t.Errorf("n1 name = %q, want worker-1", m.NodeName)
			}
		}
		if m.NodeID == n2.ID.String() {
			foundN2 = true
			if m.Path != nil {
				t.Errorf("n2 path = %v, want nil", m.Path)
			}
		}
	}
	if !foundN1 || !foundN2 {
		t.Errorf("expected to find both n1 (%v) and n2 (%v)", foundN1, foundN2)
	}
}

func TestService_SetNodePath(t *testing.T) {
	client := setupTestClient(t)
	ctx := context.Background()
	r := repo.New(client)
	nodeSvc := node.New(client, node.Config{})
	svc := service.New(r, service.Config{NodeReader: nodeSvc})

	p, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "set-path-proj",
		Name:         "Set Path Proj",
		RepoProvider: "github",
		RepoRef:      "org/set-path-proj",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	n, err := client.Node.Create().
		SetName("worker-set-path").
		SetRole("worker").
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// 1. Unknown project returns ErrProjectNotFound
	p1 := "/some/path"
	_, err = svc.SetNodePath(ctx, "00000000-0000-0000-0000-000000000000", n.ID.String(), &p1)
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}

	// 2. Unknown node returns ErrNodeNotFound
	_, err = svc.SetNodePath(ctx, p.ID, "00000000-0000-0000-0000-000000000000", &p1)
	if !errors.Is(err, domain.ErrNodeNotFound) {
		t.Fatalf("expected ErrNodeNotFound, got %v", err)
	}

	// 3. Relative path returns ErrInvalidPath
	relPath := "relative/path"
	_, err = svc.SetNodePath(ctx, p.ID, n.ID.String(), &relPath)
	if !errors.Is(err, domain.ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath, got %v", err)
	}

	// 4. Valid absolute path upserts mapping
	absPath := "/var/repo/p"
	resp, err := svc.SetNodePath(ctx, p.ID, n.ID.String(), &absPath)
	if err != nil {
		t.Fatalf("SetNodePath failed: %v", err)
	}
	if resp.Path == nil || *resp.Path != absPath {
		t.Errorf("got path %v, want %q", resp.Path, absPath)
	}
	if resp.NodeID != n.ID.String() {
		t.Errorf("got NodeID %q, want %q", resp.NodeID, n.ID.String())
	}

	// 5. Upsert with new path updates existing mapping
	newPath := "/new/absolute/path"
	resp, err = svc.SetNodePath(ctx, p.ID, n.ID.String(), &newPath)
	if err != nil {
		t.Fatalf("SetNodePath update failed: %v", err)
	}
	if resp.Path == nil || *resp.Path != newPath {
		t.Errorf("got path %v, want %q", resp.Path, newPath)
	}

	// 6. Clear with empty string sets path to NULL (nil)
	emptyPath := ""
	resp, err = svc.SetNodePath(ctx, p.ID, n.ID.String(), &emptyPath)
	if err != nil {
		t.Fatalf("SetNodePath clear with empty failed: %v", err)
	}
	if resp.Path != nil {
		t.Errorf("expected nil path after empty string, got %v", resp.Path)
	}

	// 7. Clear with nil sets path to NULL (nil)
	absPath2 := "/another/path"
	_, err = svc.SetNodePath(ctx, p.ID, n.ID.String(), &absPath2)
	if err != nil {
		t.Fatalf("SetNodePath re-set failed: %v", err)
	}
	resp, err = svc.SetNodePath(ctx, p.ID, n.ID.String(), nil)
	if err != nil {
		t.Fatalf("SetNodePath clear with nil failed: %v", err)
	}
	if resp.Path != nil {
		t.Errorf("expected nil path after nil, got %v", resp.Path)
	}
}

func TestService_ProjectNode_CascadeDelete(t *testing.T) {
	client := setupTestClient(t)
	ctx := context.Background()
	r := repo.New(client)
	nodeSvc := node.New(client, node.Config{})
	svc := service.New(r, service.Config{NodeReader: nodeSvc})

	// Case A: Deleting project via service removes ProjectNode rows
	p1, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "cascade-p1",
		Name:         "Cascade P1",
		RepoProvider: "github",
		RepoRef:      "org/cascade-p1",
	})
	if err != nil {
		t.Fatalf("create p1: %v", err)
	}
	n1, err := client.Node.Create().
		SetName("worker-cascade-1").
		SetRole("worker").
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create n1: %v", err)
	}
	path1 := "/path/one"
	_, err = svc.SetNodePath(ctx, p1.ID, n1.ID.String(), &path1)
	if err != nil {
		t.Fatalf("SetNodePath p1-n1: %v", err)
	}

	count, err := client.ProjectNode.Query().Count(ctx)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 project node, got %d (err: %v)", count, err)
	}

	if err := svc.Delete(ctx, p1.ID); err != nil {
		t.Fatalf("delete project p1: %v", err)
	}
	count, err = client.ProjectNode.Query().Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("expected 0 project nodes after project delete, got %d (err: %v)", count, err)
	}

	// Case B: Deleting node via node service removes ProjectNode rows
	p2, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "cascade-p2",
		Name:         "Cascade P2",
		RepoProvider: "github",
		RepoRef:      "org/cascade-p2",
	})
	if err != nil {
		t.Fatalf("create p2: %v", err)
	}
	n2, err := client.Node.Create().
		SetName("worker-cascade-2").
		SetRole("worker").
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create n2: %v", err)
	}
	path2 := "/path/two"
	_, err = svc.SetNodePath(ctx, p2.ID, n2.ID.String(), &path2)
	if err != nil {
		t.Fatalf("SetNodePath p2-n2: %v", err)
	}

	count, err = client.ProjectNode.Query().Count(ctx)
	if err != nil || count != 1 {
		t.Fatalf("expected 1 project node, got %d (err: %v)", count, err)
	}

	if err := nodeSvc.Delete(ctx, n2.ID.String()); err != nil {
		t.Fatalf("delete node n2: %v", err)
	}
	count, err = client.ProjectNode.Query().Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("expected 0 project nodes after node delete, got %d (err: %v)", count, err)
	}
}

func TestService_UpdateTrackerLabels(t *testing.T) {
	client := setupTestClient(t)
	ctx := context.Background()
	r := repo.New(client)
	validatorErr := errors.New("collision with global label")
	svc := service.New(r, service.Config{
		TrackerLabelValidator: func(_ context.Context, labels map[string]string) error {
			if labels["todo"] == "invalid" {
				return validatorErr
			}
			return nil
		},
	})

	p, err := svc.Create(ctx, domain.CreateProjectRequest{
		Slug:         "labels-proj",
		Name:         "Labels Project",
		RepoProvider: "github",
		RepoRef:      "owner/labels",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// 1. Set trackerLabels
	updated, err := svc.Update(ctx, p.ID, domain.UpdateProjectRequest{
		TrackerLabels: map[string]string{"todo": "queue", "blocked": "stuck"},
	})
	if err != nil {
		t.Fatalf("set trackerLabels failed: %v", err)
	}
	if updated.TrackerLabels["todo"] != "queue" || updated.TrackerLabels["blocked"] != "stuck" {
		t.Fatalf("unexpected trackerLabels: %v", updated.TrackerLabels)
	}

	// 2. Update trackerLabels
	updated, err = svc.Update(ctx, p.ID, domain.UpdateProjectRequest{
		TrackerLabels: map[string]string{"todo": "next-up"},
	})
	if err != nil {
		t.Fatalf("update trackerLabels failed: %v", err)
	}
	if len(updated.TrackerLabels) != 1 || updated.TrackerLabels["todo"] != "next-up" {
		t.Fatalf("expected replaced trackerLabels {todo: next-up}, got %v", updated.TrackerLabels)
	}

	// 3. Invalid trackerLabels rejected by validator
	_, err = svc.Update(ctx, p.ID, domain.UpdateProjectRequest{
		TrackerLabels: map[string]string{"todo": "invalid"},
	})
	if !errors.Is(err, domain.ErrInvalidTrackerLabels) {
		t.Fatalf("expected ErrInvalidTrackerLabels, got %v", err)
	}

	// Existing trackerLabels preserved after rejected update
	fetched, err := svc.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if fetched.TrackerLabels["todo"] != "next-up" {
		t.Fatalf("expected existing trackerLabels preserved, got %v", fetched.TrackerLabels)
	}

	// 4. Clear with empty map {}
	cleared, err := svc.Update(ctx, p.ID, domain.UpdateProjectRequest{
		TrackerLabels: map[string]string{},
	})
	if err != nil {
		t.Fatalf("clear trackerLabels with empty map failed: %v", err)
	}
	if len(cleared.TrackerLabels) != 0 {
		t.Fatalf("expected cleared trackerLabels, got %v", cleared.TrackerLabels)
	}

	// 5. Re-set then clear with ClearTrackerLabels flag (null in JSON)
	_, err = svc.Update(ctx, p.ID, domain.UpdateProjectRequest{
		TrackerLabels: map[string]string{"review": "qa"},
	})
	if err != nil {
		t.Fatalf("re-set trackerLabels failed: %v", err)
	}
	clearedNull, err := svc.Update(ctx, p.ID, domain.UpdateProjectRequest{
		ClearTrackerLabels: true,
	})
	if err != nil {
		t.Fatalf("clear trackerLabels with flag failed: %v", err)
	}
	if len(clearedNull.TrackerLabels) != 0 {
		t.Fatalf("expected cleared trackerLabels, got %v", clearedNull.TrackerLabels)
	}
}
