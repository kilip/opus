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

// RunReader looks up runs for caller context verification.
type RunReader interface {
	Get(ctx context.Context, id uuid.UUID) (run.Run, error)
}

// Handler serves the Streamable HTTP MCP endpoint at POST /mcp.
type Handler struct {
	svc         *service.Service
	nodes       NodeAuthenticator
	auth        SessionValidator
	runs        RunReader
	httpHandler http.Handler
}

// Config configures the MCP HTTP handler.
type Config struct {
	Service *service.Service
	Nodes   NodeAuthenticator
	Auth    SessionValidator
	Runs    RunReader
}

// New constructs a new MCP HTTP Handler.
func New(cfg Config) *Handler {
	h := &Handler{
		svc:   cfg.Service,
		nodes: cfg.Nodes,
		auth:  cfg.Auth,
		runs:  cfg.Runs,
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
	var (
		caller       domain.CallerContext
		authedNodeID string
		authedUserID uuid.UUID
	)

	authz := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		cred := strings.TrimSpace(authz[len("bearer "):])
		if cred == "" || h.nodes == nil {
			return domain.CallerContext{}, httpx.Unauthorized()
		}
		n, err := h.nodes.AuthenticateWorker(c.Context(), cred)
		if err != nil || n == nil {
			return domain.CallerContext{}, httpx.Unauthorized()
		}
		authedNodeID = n.ID
		caller.NodeID = n.ID
	} else {
		cookie := c.Cookies(auth.SessionCookie)
		if cookie == "" || h.auth == nil {
			return domain.CallerContext{}, httpx.Unauthorized()
		}
		uid, err := h.auth.ValidateSession(c.Context(), cookie)
		if err != nil || uid == uuid.Nil {
			return domain.CallerContext{}, httpx.Unauthorized()
		}
		authedUserID = uid
	}

	runHeader := strings.TrimSpace(c.Get("X-Opus-Run-ID"))
	sessionHeader := strings.TrimSpace(c.Get("X-Opus-Session-ID"))
	projectHeader := strings.TrimSpace(c.Get("X-Opus-Project-ID"))

	if runHeader != "" {
		if h.runs == nil {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "run validation unavailable")
		}
		runUUID, err := uuid.Parse(runHeader)
		if err != nil {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "invalid X-Opus-Run-ID")
		}
		r, err := h.runs.Get(c.Context(), runUUID)
		if err != nil {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "run not found")
		}
		if authedNodeID != "" && (r.NodeID == nil || r.NodeID.String() != authedNodeID) {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "run does not belong to authenticated node")
		}
		if sessionHeader != "" && (r.SessionID == nil || r.SessionID.String() != sessionHeader) {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "session does not match run")
		}
		if projectHeader != "" && r.ProjectID.String() != projectHeader {
			return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "project does not match run")
		}
		caller.RunID = r.ID.String()
		caller.TaskID = r.TaskID.String()
		caller.ProjectID = r.ProjectID.String()
		if r.SessionID != nil {
			caller.SessionID = r.SessionID.String()
		}
	} else {
		if projectHeader != "" {
			if _, err := uuid.Parse(projectHeader); err != nil {
				return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "invalid X-Opus-Project-ID")
			}
			caller.ProjectID = projectHeader
		}
		if sessionHeader != "" {
			if _, err := uuid.Parse(sessionHeader); err != nil {
				return domain.CallerContext{}, fiber.NewError(fiber.StatusForbidden, "invalid X-Opus-Session-ID")
			}
			caller.SessionID = sessionHeader
		}
		_ = authedUserID
	}

	return caller, nil
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
