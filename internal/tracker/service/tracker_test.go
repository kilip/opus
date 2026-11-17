package service_test

import (
	"context"
	"database/sql"
	"errors"
	"maps"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/task"
	"github.com/kilip/opus/internal/tracker/domain"
	"github.com/kilip/opus/internal/tracker/repo"
	"github.com/kilip/opus/internal/tracker/service"
	_ "modernc.org/sqlite"
)

func setupTrackerClient(t *testing.T) (*ent.Client, *sql.DB) {
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

func TestTrackerService_GetProjectStatus(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	svc := service.New(r)
	ctx := context.Background()

	// Malformed UUID -> ErrInvalidProjectID
	if _, err := svc.GetProjectStatus(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrInvalidProjectID) {
		t.Fatalf("expected ErrInvalidProjectID, got %v", err)
	}

	// Unknown project -> ErrProjectNotFound
	if _, err := svc.GetProjectStatus(ctx, "01933a3c-1b77-7000-8000-000000000000"); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}

	// Project without sync row returns zero status
	p, err := client.Project.Create().
		SetSlug("status-proj").
		SetName("Status Proj").
		SetRepoProvider("github").
		SetRepoRef("org/status").
		SetTrackerProvider("github").
		SetTrackerRef("org/status").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	st, err := svc.GetProjectStatus(ctx, p.ID.String())
	if err != nil {
		t.Fatalf("GetProjectStatus: %v", err)
	}
	if st.LastSyncAt != nil || st.LastError != "" {
		t.Fatalf("expected empty sync status, got %+v", st)
	}

	// After saving sync state, returns populated fields
	syncTime := time.Date(2026, 11, 17, 1, 0, 0, 0, time.UTC)
	if _, err := r.SaveSyncSuccess(ctx, p.ID.String(), "2026-11-17T01:00:00Z", syncTime); err != nil {
		t.Fatalf("SaveSyncSuccess: %v", err)
	}
	if err := r.UpsertLastError(ctx, p.ID.String(), "rate limit"); err != nil {
		t.Fatalf("UpsertLastError: %v", err)
	}

	st, err = svc.GetProjectStatus(ctx, p.ID.String())
	if err != nil {
		t.Fatalf("GetProjectStatus after sync: %v", err)
	}
	if st.LastSyncAt == nil || !st.LastSyncAt.Equal(syncTime) {
		t.Fatalf("expected LastSyncAt %v, got %v", syncTime, st.LastSyncAt)
	}
	if st.LastError != "rate limit" {
		t.Fatalf("expected LastError %q, got %q", "rate limit", st.LastError)
	}
}

func TestTrackerService_SettingsAndEffectiveLabels(t *testing.T) {
	client, _ := setupTrackerClient(t)
	r := repo.New(client)
	svc := service.New(r)
	ctx := context.Background()

	// 1. GetSettings initially returns defaults
	got, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !maps.Equal(got.StatusLabels, domain.DefaultStatusLabels()) {
		t.Fatalf("expected defaults %v, got %v", domain.DefaultStatusLabels(), got.StatusLabels)
	}

	// 2. Create a project with a project-level override {"todo": "project-queue"}
	p, err := client.Project.Create().
		SetSlug("override-proj").
		SetName("Override Proj").
		SetRepoProvider("github").
		SetRepoRef("org/override").
		SetTrackerProvider("github").
		SetTrackerRef("org/override").
		SetTrackerLabels(map[string]string{"todo": "project-queue"}).
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 3. Updating global settings with a label that collides with project override fails
	_, err = svc.UpdateSettings(ctx, domain.UpdateTrackerSettingsRequest{
		StatusLabels: map[string]string{"blocked": "project-queue"},
	})
	if !errors.Is(err, domain.ErrInvalidLabelConfig) {
		t.Fatalf("expected ErrInvalidLabelConfig on cross-project collision, got %v", err)
	}

	// 4. Valid global update succeeds and merges with defaults
	updated, err := svc.UpdateSettings(ctx, domain.UpdateTrackerSettingsRequest{
		StatusLabels: map[string]string{"backlog": "icebox", "todo": "triage"},
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	wantGlobal := map[string]string{
		"backlog":     "icebox",
		"todo":        "triage",
		"in-progress": "status:in-progress",
		"review":      "status:review",
		"blocked":     "status:blocked",
	}
	if !maps.Equal(updated.StatusLabels, wantGlobal) {
		t.Fatalf("got %v, want %v", updated.StatusLabels, wantGlobal)
	}

	// 5. ValidateProjectLabels rejects colliding with global override "icebox"
	err = svc.ValidateProjectLabels(ctx, map[string]string{"review": "icebox"})
	if !errors.Is(err, domain.ErrInvalidLabelConfig) {
		t.Fatalf("expected ErrInvalidLabelConfig colliding with global, got %v", err)
	}
	if err := svc.ValidateProjectLabels(ctx, map[string]string{"review": "qa-ready"}); err != nil {
		t.Fatalf("expected valid project override, got %v", err)
	}

	// 6. EffectiveLabels resolves project > global > default + reverse map
	eff, err := svc.EffectiveLabels(ctx, project.Record{
		ID:            p.ID.String(),
		TrackerLabels: p.TrackerLabels,
	})
	if err != nil {
		t.Fatalf("EffectiveLabels: %v", err)
	}
	if eff.StatusToLabel["todo"] != "project-queue" {
		t.Fatalf("expected project override todo=project-queue, got %q", eff.StatusToLabel["todo"])
	}
	if eff.StatusToLabel["backlog"] != "icebox" {
		t.Fatalf("expected global override backlog=icebox, got %q", eff.StatusToLabel["backlog"])
	}
	if eff.StatusToLabel["in-progress"] != "status:in-progress" {
		t.Fatalf("expected default in-progress, got %q", eff.StatusToLabel["in-progress"])
	}
	if eff.LabelToStatus["project-queue"] != task.StatusTodo {
		t.Fatalf("expected reverse project-queue -> todo, got %q", eff.LabelToStatus["project-queue"])
	}

	// 7. Empty map {} resets global overrides back to defaults
	reset, err := svc.UpdateSettings(ctx, domain.UpdateTrackerSettingsRequest{
		StatusLabels: map[string]string{},
	})
	if err != nil {
		t.Fatalf("UpdateSettings reset: %v", err)
	}
	if !maps.Equal(reset.StatusLabels, domain.DefaultStatusLabels()) {
		t.Fatalf("expected reset to defaults %v, got %v", domain.DefaultStatusLabels(), reset.StatusLabels)
	}
}
