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
