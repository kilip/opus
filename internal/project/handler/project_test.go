package handler_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/enttest"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/project/domain"
	"github.com/kilip/opus/internal/project/handler"
	"github.com/kilip/opus/internal/project/repo"
	"github.com/kilip/opus/internal/project/service"
	_ "modernc.org/sqlite"
)

func setupApp(t *testing.T, tc ...service.TaskCounter) (*fiber.App, *ent.Client) {
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

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{
		ErrorHandler: httpx.ErrorHandler(logger),
	})

	var taskCounter service.TaskCounter
	if len(tc) > 0 {
		taskCounter = tc[0]
	}
	r := repo.New(client)
	nodeSvc := node.New(client, node.Config{})
	s := service.New(r, service.Config{TaskCounter: taskCounter, NodeReader: nodeSvc})
	h := handler.New(s)
	h.Register(app.Group("/api"))
	return app, client
}

func req(t *testing.T, app *fiber.App, method, path string, body any) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		switch b := body.(type) {
		case string:
			bodyReader = bytes.NewReader([]byte(b))
		case []byte:
			bodyReader = bytes.NewReader(b)
		default:
			buf, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			bodyReader = bytes.NewReader(buf)
		}
	}

	r := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.Test(r)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return v
}

func decodeError(t *testing.T, resp *http.Response) httpx.ErrorResponse {
	t.Helper()
	var v httpx.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return v
}

func TestHandler_CRUD(t *testing.T) {
	app, _ := setupApp(t)

	// 1. Initial list should be empty with total = 0
	respList := req(t, app, http.MethodGet, "/api/projects", nil)
	if respList.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", respList.StatusCode)
	}
	listEnvelope := decode[httpx.Response[[]domain.Project]](t, respList)
	if listEnvelope.Meta == nil || listEnvelope.Meta.Total == nil || *listEnvelope.Meta.Total != 0 {
		t.Fatalf("expected meta.total = 0, got %v", listEnvelope.Meta)
	}
	if len(listEnvelope.Data) != 0 {
		t.Fatalf("expected 0 projects, got %d", len(listEnvelope.Data))
	}

	// 2. Create project returns 201
	createBody := domain.CreateProjectRequest{
		Slug:         "core-api",
		Name:         "Core API",
		RepoProvider: "github",
		RepoRef:      "kilip/opus-api",
	}
	respCreate := req(t, app, http.MethodPost, "/api/projects", createBody)
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", respCreate.StatusCode)
	}
	createEnvelope := decode[httpx.Response[domain.Project]](t, respCreate)
	p := createEnvelope.Data
	if p.ID == "" || p.Slug != "core-api" || p.Name != "Core API" {
		t.Fatalf("unexpected created project: %+v", p)
	}
	// Defaulting checks
	if p.TrackerProvider != "github" || p.TrackerRef != "kilip/opus-api" || p.DefaultBranch != "main" {
		t.Fatalf("defaults not set properly: %+v", p)
	}

	// 3. Get created project returns 200
	respGet := req(t, app, http.MethodGet, "/api/projects/"+p.ID, nil)
	if respGet.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", respGet.StatusCode)
	}
	getEnvelope := decode[httpx.Response[domain.Project]](t, respGet)
	if getEnvelope.Data.ID != p.ID {
		t.Fatalf("expected project ID %s, got %s", p.ID, getEnvelope.Data.ID)
	}

	// 4. Update project returns 200
	newName := "Core API Updated"
	newBranch := "dev"
	newTrackerProvider := "jira"
	newTrackerRef := "PROJ-1"
	respUpdate := req(t, app, http.MethodPatch, "/api/projects/"+p.ID, domain.UpdateProjectRequest{
		Name:            &newName,
		DefaultBranch:   &newBranch,
		TrackerProvider: &newTrackerProvider,
		TrackerRef:      &newTrackerRef,
	})
	if respUpdate.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", respUpdate.StatusCode)
	}
	updateEnvelope := decode[httpx.Response[domain.Project]](t, respUpdate)
	if updateEnvelope.Data.Name != newName || updateEnvelope.Data.DefaultBranch != newBranch {
		t.Fatalf("unexpected updated project: %+v", updateEnvelope.Data)
	}

	// 5. Delete project returns 204
	respDelete := req(t, app, http.MethodDelete, "/api/projects/"+p.ID, nil)
	if respDelete.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", respDelete.StatusCode)
	}

	// 6. Project is now gone (404)
	respGetDeleted := req(t, app, http.MethodGet, "/api/projects/"+p.ID, nil)
	if respGetDeleted.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", respGetDeleted.StatusCode)
	}
}

func TestHandler_ValidationErrors(t *testing.T) {
	app, _ := setupApp(t)

	// Create a base project to test duplicate slug later
	respFirst := req(t, app, http.MethodPost, "/api/projects", domain.CreateProjectRequest{
		Slug:         "existing-slug",
		Name:         "Existing",
		RepoProvider: "github",
		RepoRef:      "owner/repo",
	})
	if respFirst.StatusCode != http.StatusCreated {
		t.Fatalf("first project creation failed: %d", respFirst.StatusCode)
	}

	tests := []struct {
		name       string
		payload    any
		wantStatus int
		wantError  string
	}{
		{
			name: "invalid slug uppercase",
			payload: domain.CreateProjectRequest{
				Slug:         "InvalidSlug",
				Name:         "Test",
				RepoProvider: "github",
				RepoRef:      "owner/repo",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_INVALID_SLUG",
		},
		{
			name: "invalid slug hyphens",
			payload: domain.CreateProjectRequest{
				Slug:         "-invalid",
				Name:         "Test",
				RepoProvider: "github",
				RepoRef:      "owner/repo",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_INVALID_SLUG",
		},
		{
			name: "duplicate slug",
			payload: domain.CreateProjectRequest{
				Slug:         "existing-slug",
				Name:         "Another",
				RepoProvider: "github",
				RepoRef:      "owner/another",
			},
			wantStatus: http.StatusConflict,
			wantError:  "PROJECT_DUPLICATE_SLUG",
		},
		{
			name: "repo_provider not github",
			payload: domain.CreateProjectRequest{
				Slug:         "valid-slug",
				Name:         "GitLab Project",
				RepoProvider: "gitlab",
				RepoRef:      "owner/repo",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_INVALID_REPO",
		},
		{
			name: "repo_ref not owner/name",
			payload: domain.CreateProjectRequest{
				Slug:         "valid-slug-2",
				Name:         "Project",
				RepoProvider: "github",
				RepoRef:      "justrepo",
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_INVALID_REPO",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := req(t, app, http.MethodPost, "/api/projects", tt.payload)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			errResp := decodeError(t, resp)
			if errResp.Error != tt.wantError {
				t.Fatalf("expected error %q, got %q", tt.wantError, errResp.Error)
			}
		})
	}
}

func TestHandler_PatchForbiddenFields(t *testing.T) {
	app, _ := setupApp(t)

	respCreate := req(t, app, http.MethodPost, "/api/projects", domain.CreateProjectRequest{
		Slug:         "patch-test",
		Name:         "Patch Test",
		RepoProvider: "github",
		RepoRef:      "owner/repo",
	})
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("create failed: %d", respCreate.StatusCode)
	}
	p := decode[httpx.Response[domain.Project]](t, respCreate).Data

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "patch body contains slug",
			body:       `{"slug": "new-slug"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_PATCH_FORBIDDEN",
		},
		{
			name:       "patch body contains empty slug",
			body:       `{"slug": ""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_PATCH_FORBIDDEN",
		},
		{
			name:       "patch body contains repo_ref",
			body:       `{"repo_ref": "owner/new-repo"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_PATCH_FORBIDDEN",
		},
		{
			name:       "patch body contains empty repo_ref",
			body:       `{"repo_ref": ""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_PATCH_FORBIDDEN",
		},
		{
			name:       "patch body contains both forbidden fields",
			body:       `{"name": "Valid Name", "slug": "patch-test", "repo_ref": "owner/repo"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "PROJECT_PATCH_FORBIDDEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := req(t, app, http.MethodPatch, "/api/projects/"+p.ID, tt.body)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, resp.StatusCode)
			}
			errResp := decodeError(t, resp)
			if errResp.Error != tt.wantError {
				t.Fatalf("expected error %q, got %q", tt.wantError, errResp.Error)
			}
		})
	}
}

func TestHandler_UnknownAndMalformedID(t *testing.T) {
	app, _ := setupApp(t)

	ids := []string{
		"01933a3c-1b77-7000-8000-000000000000", // unknown UUID
		"not-a-valid-uuid",                     // malformed ID
	}

	for _, id := range ids {
		t.Run("GET "+id, func(t *testing.T) {
			resp := req(t, app, http.MethodGet, "/api/projects/"+id, nil)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 404, got %d", resp.StatusCode)
			}
			errResp := decodeError(t, resp)
			if errResp.Error != "PROJECT_NOT_FOUND" {
				t.Fatalf("expected PROJECT_NOT_FOUND, got %q", errResp.Error)
			}
		})

		t.Run("PATCH "+id, func(t *testing.T) {
			newName := "Test"
			resp := req(t, app, http.MethodPatch, "/api/projects/"+id, domain.UpdateProjectRequest{Name: &newName})
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 404, got %d", resp.StatusCode)
			}
			errResp := decodeError(t, resp)
			if errResp.Error != "PROJECT_NOT_FOUND" {
				t.Fatalf("expected PROJECT_NOT_FOUND, got %q", errResp.Error)
			}
		})

		t.Run("DELETE "+id, func(t *testing.T) {
			resp := req(t, app, http.MethodDelete, "/api/projects/"+id, nil)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("expected 404, got %d", resp.StatusCode)
			}
			errResp := decodeError(t, resp)
			if errResp.Error != "PROJECT_NOT_FOUND" {
				t.Fatalf("expected PROJECT_NOT_FOUND, got %q", errResp.Error)
			}
		})
	}
}

func TestHandler_DeleteRefusalWithTasks(t *testing.T) {
	taskCounts := make(map[string]int)
	tc := func(_ context.Context, projectID string) (int, error) {
		return taskCounts[projectID], nil
	}

	app, _ := setupApp(t, tc)

	respCreate := req(t, app, http.MethodPost, "/api/projects", domain.CreateProjectRequest{
		Slug:         "guarded-project",
		Name:         "Guarded",
		RepoProvider: "github",
		RepoRef:      "owner/repo",
	})
	if respCreate.StatusCode != http.StatusCreated {
		t.Fatalf("create failed: %d", respCreate.StatusCode)
	}
	p := decode[httpx.Response[domain.Project]](t, respCreate).Data

	// Project has 5 tasks -> DELETE must return 409 Conflict
	taskCounts[p.ID] = 5
	respDelete := req(t, app, http.MethodDelete, "/api/projects/"+p.ID, nil)
	if respDelete.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", respDelete.StatusCode)
	}
	errResp := decodeError(t, respDelete)
	if errResp.Error != "PROJECT_HAS_TASKS" {
		t.Fatalf("expected error PROJECT_HAS_TASKS, got %q", errResp.Error)
	}

	// Project has 0 tasks -> DELETE must return 204 No Content
	taskCounts[p.ID] = 0
	respDeleteOk := req(t, app, http.MethodDelete, "/api/projects/"+p.ID, nil)
	if respDeleteOk.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", respDeleteOk.StatusCode)
	}
}

func TestHandler_ListNodeMappings(t *testing.T) {
	app, client := setupApp(t)
	ctx := context.Background()

	// 1. Invalid UUID returns 400
	respBad := req(t, app, http.MethodGet, "/api/projects/not-a-uuid/nodes", nil)
	if respBad.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", respBad.StatusCode)
	}

	// 2. Non-existent UUID returns 404
	respNotFound := req(t, app, http.MethodGet, "/api/projects/00000000-0000-0000-0000-000000000000/nodes", nil)
	if respNotFound.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", respNotFound.StatusCode)
	}

	// 3. Create project and 2 nodes
	p, err := client.Project.Create().
		SetSlug("test-nodes-p").
		SetName("Test Nodes Project").
		SetRepoProvider("github").
		SetRepoRef("owner/repo").
		SetTrackerProvider("github").
		SetTrackerRef("owner/repo").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	n1, err := client.Node.Create().
		SetName("node-1").
		SetRole("master").
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create node 1: %v", err)
	}

	n2, err := client.Node.Create().
		SetName("node-2").
		SetRole("worker").
		SetStatus("offline").
		Save(ctx)
	if err != nil {
		t.Fatalf("create node 2: %v", err)
	}

	// 4. GET before mapping: all nodes have path: null
	respList := req(t, app, http.MethodGet, "/api/projects/"+p.ID.String()+"/nodes", nil)
	if respList.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", respList.StatusCode)
	}
	list := decode[httpx.Response[[]domain.ProjectNodeResponse]](t, respList).Data
	if len(list) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(list))
	}
	for _, item := range list {
		if item.Path != nil {
			t.Errorf("expected path null for unmapped node %s, got %v", item.NodeID, *item.Path)
		}
	}

	// 5. Set mapping for n1
	path1 := "/var/repo/p"
	putBody := domain.UpdateProjectNodePathRequest{Path: &path1}
	respPut := req(t, app, http.MethodPut, "/api/projects/"+p.ID.String()+"/nodes/"+n1.ID.String(), putBody)
	if respPut.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on PUT, got %d", respPut.StatusCode)
	}

	// 6. GET after mapping: n1 has path: "/var/repo/p", n2 has path: null
	respList2 := req(t, app, http.MethodGet, "/api/projects/"+p.ID.String()+"/nodes", nil)
	if respList2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", respList2.StatusCode)
	}
	list2 := decode[httpx.Response[[]domain.ProjectNodeResponse]](t, respList2).Data
	for _, item := range list2 {
		if item.NodeID == n1.ID.String() {
			if item.Path == nil || *item.Path != path1 {
				t.Errorf("expected path %q, got %v", path1, item.Path)
			}
			if item.NodeRole != "master" {
				t.Errorf("expected nodeRole master, got %q", item.NodeRole)
			}
		}
		if item.NodeID == n2.ID.String() {
			if item.Path != nil {
				t.Errorf("expected path null for n2, got %v", item.Path)
			}
		}
	}
}

func TestHandler_SetNodePath(t *testing.T) {
	app, client := setupApp(t)
	ctx := context.Background()

	p, err := client.Project.Create().
		SetSlug("set-node-path-p").
		SetName("Set Node Path Project").
		SetRepoProvider("github").
		SetRepoRef("owner/repo").
		SetTrackerProvider("github").
		SetTrackerRef("owner/repo").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	n, err := client.Node.Create().
		SetName("worker-set").
		SetRole("worker").
		SetStatus("online").
		Save(ctx)
	if err != nil {
		t.Fatalf("create node: %v", err)
	}

	// 1. Unknown project returns 404
	validPath := "/opt/opus"
	respProj404 := req(t, app, http.MethodPut, "/api/projects/00000000-0000-0000-0000-000000000000/nodes/"+n.ID.String(), domain.UpdateProjectNodePathRequest{Path: &validPath})
	if respProj404.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown project, got %d", respProj404.StatusCode)
	}

	// 2. Unknown node returns 404
	respNode404 := req(t, app, http.MethodPut, "/api/projects/"+p.ID.String()+"/nodes/00000000-0000-0000-0000-000000000000", domain.UpdateProjectNodePathRequest{Path: &validPath})
	if respNode404.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown node, got %d", respNode404.StatusCode)
	}

	// 3. Non-absolute path returns 400
	relPath := "relative/path"
	respRel := req(t, app, http.MethodPut, "/api/projects/"+p.ID.String()+"/nodes/"+n.ID.String(), domain.UpdateProjectNodePathRequest{Path: &relPath})
	if respRel.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for relative path, got %d", respRel.StatusCode)
	}
	errResp := decodeError(t, respRel)
	if errResp.Error != "PROJECT_INVALID_PATH" {
		t.Fatalf("expected PROJECT_INVALID_PATH, got %q", errResp.Error)
	}

	// 4. Valid absolute path upserts mapping -> 200
	respOk := req(t, app, http.MethodPut, "/api/projects/"+p.ID.String()+"/nodes/"+n.ID.String(), domain.UpdateProjectNodePathRequest{Path: &validPath})
	if respOk.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", respOk.StatusCode)
	}
	resData := decode[httpx.Response[domain.ProjectNodeResponse]](t, respOk).Data
	if resData.Path == nil || *resData.Path != validPath {
		t.Fatalf("expected path %q, got %v", validPath, resData.Path)
	}
	if resData.NodeID != n.ID.String() {
		t.Errorf("expected NodeID %q, got %q", n.ID.String(), resData.NodeID)
	}
	if resData.NodeName != "worker-set" {
		t.Errorf("expected NodeName worker-set, got %q", resData.NodeName)
	}

	// 5. Update to empty string clears mapping to null -> 200
	emptyPath := ""
	respEmpty := req(t, app, http.MethodPut, "/api/projects/"+p.ID.String()+"/nodes/"+n.ID.String(), domain.UpdateProjectNodePathRequest{Path: &emptyPath})
	if respEmpty.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on empty string, got %d", respEmpty.StatusCode)
	}
	resEmptyData := decode[httpx.Response[domain.ProjectNodeResponse]](t, respEmpty).Data
	if resEmptyData.Path != nil {
		t.Fatalf("expected path null, got %v", resEmptyData.Path)
	}

	// 6. Update to new path -> 200
	newPath := "/new/custom/path"
	respNew := req(t, app, http.MethodPut, "/api/projects/"+p.ID.String()+"/nodes/"+n.ID.String(), domain.UpdateProjectNodePathRequest{Path: &newPath})
	if respNew.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", respNew.StatusCode)
	}
	resNewData := decode[httpx.Response[domain.ProjectNodeResponse]](t, respNew).Data
	if resNewData.Path == nil || *resNewData.Path != newPath {
		t.Fatalf("expected path %q, got %v", newPath, resNewData.Path)
	}

	// 7. Update to null clears mapping to null -> 200
	respNull := req(t, app, http.MethodPut, "/api/projects/"+p.ID.String()+"/nodes/"+n.ID.String(), domain.UpdateProjectNodePathRequest{Path: nil})
	if respNull.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK on null, got %d", respNull.StatusCode)
	}
	resNullData := decode[httpx.Response[domain.ProjectNodeResponse]](t, respNull).Data
	if resNullData.Path != nil {
		t.Fatalf("expected path null, got %v", resNullData.Path)
	}
}
