package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/agent/domain"
	"github.com/kilip/opus/internal/agent/service"
	"github.com/kilip/opus/internal/auth"
	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/project"
	"github.com/kilip/opus/internal/run"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type callerContextKey struct{}

// NodeAuthenticator validates worker node Bearer credentials.
type NodeAuthenticator interface {
	AuthenticateWorker(ctx context.Context, cred string) (*node.NodeRecord, error)
}

// SessionValidator validates user session cookie tokens.
type SessionValidator interface {
	ValidateSession(ctx context.Context, token string) (uuid.UUID, error)
}

// RunReader looks up runs and verifies run-scoped MCP tokens.
type RunReader interface {
	Get(ctx context.Context, id uuid.UUID) (run.Run, error)
	AuthenticateMCP(ctx context.Context, runID uuid.UUID, rawToken string) (run.Run, error)
}

// ProjectLister lists projects accessible to cookie-authenticated callers.
type ProjectLister interface {
	List(ctx context.Context) ([]project.Record, error)
}

// Handler serves the Streamable HTTP MCP endpoint at POST /mcp.
type Handler struct {
	svc         *service.Service
	nodes       NodeAuthenticator
	auth        SessionValidator
	runs        RunReader
	projects    ProjectLister
	httpHandler http.Handler
}

// Config configures the MCP HTTP handler.
type Config struct {
	Service  *service.Service
	Nodes    NodeAuthenticator
	Auth     SessionValidator
	Runs     RunReader
	Projects ProjectLister
}

// New constructs a new MCP HTTP Handler.
func New(cfg Config) *Handler {
	h := &Handler{
		svc:      cfg.Service,
		nodes:    cfg.Nodes,
		auth:     cfg.Auth,
		runs:     cfg.runsOrNil(),
		projects: cfg.Projects,
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "opus",
		Version: "1.0.0",
	}, nil)
	h.registerTools(server)

	h.httpHandler = mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:                  true,
		JSONResponse:               true,
		DisableLocalhostProtection: true,
	})
	return h
}

func (c Config) runsOrNil() RunReader {
	return c.Runs
}

// Register mounts POST /mcp outside /api.
//
// ponytail: /mcp speaks JSON-RPC 2.0 Streamable HTTP MCP rather than REST JSON,
// so it is intentionally excluded from swaggo/OpenAPI swagger.json generation.
func (h *Handler) Register(r fiber.Router) {
	r.Post("/mcp", h.handleMCP)
}

func (h *Handler) handleMCP(c fiber.Ctx) error {
	caller, err := h.authenticateAndBuildCaller(c)
	if err != nil {
		return err
	}
	return adaptor.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), callerContextKey{}, caller)
		h.httpHandler.ServeHTTP(w, r.WithContext(ctx))
	}))(c)
}

func (h *Handler) authenticateAndBuildCaller(c fiber.Ctx) (domain.CallerContext, error) {
	runHeader := strings.TrimSpace(c.Get("X-Opus-Run-ID"))
	sessionHeader := strings.TrimSpace(c.Get("X-Opus-Session-ID"))
	projectHeader := strings.TrimSpace(c.Get("X-Opus-Project-ID"))

	authz := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		cred := strings.TrimSpace(authz[len("bearer "):])
		if cred == "" {
			return domain.CallerContext{}, httpx.Unauthorized()
		}

		// 1. Worker node credential: requires X-Opus-Run-ID and node ownership of the run.
		if h.nodes != nil {
			if n, err := h.nodes.AuthenticateWorker(c.Context(), cred); err == nil && n != nil {
				if runHeader == "" {
					return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "run binding required")
				}
				r, err := h.lookupRun(c.Context(), runHeader)
				if err != nil {
					return domain.CallerContext{}, err
				}
				if r.NodeID == nil || r.NodeID.String() != n.ID {
					return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "run does not belong to authenticated node")
				}
				if err := verifyRunHeaders(r, sessionHeader, projectHeader); err != nil {
					return domain.CallerContext{}, err
				}
				return callerFromRun(r, n.ID), nil
			}
		}

		// 2. Run-scoped MCP token: requires X-Opus-Run-ID and valid active run token.
		if runHeader == "" || h.runs == nil {
			return domain.CallerContext{}, httpx.Unauthorized()
		}
		runUUID, err := uuid.Parse(runHeader)
		if err != nil {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "invalid X-Opus-Run-ID")
		}
		r, err := h.runs.AuthenticateMCP(c.Context(), runUUID, cred)
		if err != nil {
			return domain.CallerContext{}, httpx.Unauthorized()
		}
		if err := verifyRunHeaders(r, sessionHeader, projectHeader); err != nil {
			return domain.CallerContext{}, err
		}
		return callerFromRun(r, ""), nil
	}

	// 3. Cookie session caller: owner project scope (headers only filter, never grant).
	cookie := c.Cookies(auth.SessionCookie)
	if cookie == "" || h.auth == nil {
		return domain.CallerContext{}, httpx.Unauthorized()
	}
	uid, err := h.auth.ValidateSession(c.Context(), cookie)
	if err != nil || uid == uuid.Nil {
		return domain.CallerContext{}, httpx.Unauthorized()
	}

	caller := domain.CallerContext{
		UserID: uid.String(),
	}
	if h.projects != nil {
		projects, err := h.projects.List(c.Context())
		if err != nil {
			return domain.CallerContext{}, err
		}
		allowed := make([]string, 0, len(projects))
		for _, p := range projects {
			if p.ID != "" {
				allowed = append(allowed, p.ID)
			}
		}
		caller.AllowedProjectIDs = allowed
	}

	if runHeader != "" {
		r, err := h.lookupRun(c.Context(), runHeader)
		if err != nil {
			return domain.CallerContext{}, err
		}
		if err := verifyRunHeaders(r, sessionHeader, projectHeader); err != nil {
			return domain.CallerContext{}, err
		}
		if h.projects != nil && !caller.AllowsProject(r.ProjectID.String()) {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "project outside caller scope")
		}
		bound := callerFromRun(r, "")
		bound.UserID = caller.UserID
		return bound, nil
	}

	if projectHeader != "" {
		if _, err := uuid.Parse(projectHeader); err != nil {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "invalid X-Opus-Project-ID")
		}
		if h.projects != nil && !caller.AllowsProject(projectHeader) {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "project outside caller scope")
		}
		caller.ProjectID = projectHeader
		caller.AllowedProjectIDs = []string{projectHeader}
	}
	return caller, nil
}

func (h *Handler) lookupRun(ctx context.Context, runHeader string) (run.Run, error) {
	if h.runs == nil {
		return run.Run{}, fiber.NewError(fiber.StatusForbidden, "run validation unavailable")
	}
	runUUID, err := uuid.Parse(runHeader)
	if err != nil {
		return run.Run{}, fiber.NewError(fiber.StatusForbidden, "invalid X-Opus-Run-ID")
	}
	r, err := h.runs.Get(ctx, runUUID)
	if err != nil {
		return run.Run{}, fiber.NewError(fiber.StatusForbidden, "run not found")
	}
	return r, nil
}

func verifyRunHeaders(r run.Run, sessionHeader, projectHeader string) error {
	if sessionHeader != "" && (r.SessionID == nil || r.SessionID.String() != sessionHeader) {
		return fiber.NewError(fiber.StatusForbidden, "session does not match run")
	}
	if projectHeader != "" && r.ProjectID.String() != projectHeader {
		return fiber.NewError(fiber.StatusForbidden, "project does not match run")
	}
	return nil
}

func callerFromRun(r run.Run, nodeID string) domain.CallerContext {
	if nodeID == "" && r.NodeID != nil {
		nodeID = r.NodeID.String()
	}
	var sessionID string
	if r.SessionID != nil {
		sessionID = r.SessionID.String()
	}
	return domain.CallerContext{
		NodeID:            nodeID,
		RunID:             r.ID.String(),
		SessionID:         sessionID,
		TaskID:            r.TaskID.String(),
		ProjectID:         r.ProjectID.String(),
		AllowedProjectIDs: []string{r.ProjectID.String()},
	}
}

func (h *Handler) registerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task",
		Description: "Fetch task details and activity timeline within allowed project scope.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in domain.GetTaskInput) (*mcp.CallToolResult, domain.GetTaskOutput, error) {
		out, err := h.svc.GetTask(ctx, callerFromContext(ctx), in)
		return nil, out, formatToolError(err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List tasks within allowed project scope (max 50 items).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in domain.ListTasksInput) (*mcp.CallToolResult, domain.ListTasksOutput, error) {
		out, err := h.svc.ListTasks(ctx, callerFromContext(ctx), in)
		return nil, out, formatToolError(err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "set_status",
		Description: "Transition a task status (backlog, todo, in-progress, review, blocked, done).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in domain.SetStatusInput) (*mcp.CallToolResult, domain.SetStatusOutput, error) {
		out, err := h.svc.SetStatus(ctx, callerFromContext(ctx), in)
		return nil, out, formatToolError(err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_activity",
		Description: "Append a vibe-authored activity entry to a task.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in domain.AddActivityInput) (*mcp.CallToolResult, domain.AddActivityOutput, error) {
		out, err := h.svc.AddActivity(ctx, callerFromContext(ctx), in)
		return nil, out, formatToolError(err)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "submit_report",
		Description: "Submit and persist the structured completion report for the bound run.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in domain.SubmitReportInput) (*mcp.CallToolResult, domain.SubmitReportOutput, error) {
		out, err := h.svc.SubmitReport(ctx, callerFromContext(ctx), in)
		return nil, out, formatToolError(err)
	})
}

func callerFromContext(ctx context.Context) domain.CallerContext {
	if caller, ok := ctx.Value(callerContextKey{}).(domain.CallerContext); ok {
		return caller
	}
	return domain.CallerContext{}
}

func formatToolError(err error) error {
	if err == nil {
		return nil
	}
	if code := domain.Code(err); code != "" {
		return fmt.Errorf("%s: %w", code, err)
	}
	return err
}
