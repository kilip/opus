// Package agent is the MCP agent tool feature. Other packages depend on this
// package only, never on its subpackages.
package agent

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/opus/internal/agent/domain"
	"github.com/kilip/opus/internal/agent/service"
	"github.com/kilip/opus/internal/run"
	"github.com/kilip/opus/internal/session"
	"github.com/kilip/opus/internal/task"
)

// Limit and error code constants re-exported from domain.
const (
	MaxDescriptionBytes  = domain.MaxDescriptionBytes
	MaxListTasksLimit    = domain.MaxListTasksLimit
	MaxActivityBodyBytes = domain.MaxActivityBodyBytes
	MaxReportBytes       = domain.MaxReportBytes
	RateLimitPerMinute   = domain.RateLimitPerMinute
	RateLimitWindow      = domain.RateLimitWindow

	CodeNotFound       = domain.CodeNotFound
	CodeForbiddenScope = domain.CodeForbiddenScope
	CodeTooLarge       = domain.CodeTooLarge
	CodeInvalidStatus  = domain.CodeInvalidStatus
	CodeRateLimited    = domain.CodeRateLimited
	CodeInvalidInput   = domain.CodeInvalidInput
)

// Sentinel errors re-exported from domain.
var (
	ErrNotFound       = domain.ErrNotFound
	ErrForbiddenScope = domain.ErrForbiddenScope
	ErrTooLarge       = domain.ErrTooLarge
	ErrInvalidStatus  = domain.ErrInvalidStatus
	ErrRateLimited    = domain.ErrRateLimited
	ErrInvalidInput   = domain.ErrInvalidInput
)

// Re-exported domain types.
type (
	CallerContext       = domain.CallerContext
	AcceptanceCriterion = domain.AcceptanceCriterion
	Report              = domain.Report
	GetTaskInput        = domain.GetTaskInput
	GetTaskOutput       = domain.GetTaskOutput
	TaskActivityItem    = domain.TaskActivityItem
	ListTasksInput      = domain.ListTasksInput
	ListTasksOutput     = domain.ListTasksOutput
	TaskSummary         = domain.TaskSummary
	SetStatusInput      = domain.SetStatusInput
	SetStatusOutput     = domain.SetStatusOutput
	AddActivityInput    = domain.AddActivityInput
	AddActivityOutput   = domain.AddActivityOutput
	SubmitReportInput   = domain.SubmitReportInput
	SubmitReportOutput  = domain.SubmitReportOutput
)

// ErrorCode returns the stable error code for an agent domain error.
func ErrorCode(err error) string {
	return domain.Code(err)
}

// Config configures the agent feature.
type Config struct {
	Tasks    *task.Task
	Runs     *run.Feature
	Sessions *session.Session
	Now      func() time.Time
}

// Agent is the public facade for the MCP agent tool feature.
type Agent struct {
	svc *service.Service
}

// New constructs a new Agent feature facade.
func New(cfg Config) *Agent {
	return &Agent{
		svc: service.New(service.Config{
			Tasks:    cfg.Tasks,
			Runs:     cfg.Runs,
			Sessions: cfg.Sessions,
			Now:      cfg.Now,
		}),
	}
}

// Register mounts the agent MCP HTTP routes on r.
// ponytail: HTTP/MCP transport wiring (POST /mcp) lands in issue #134.
func (a *Agent) Register(_ fiber.Router) {}

// GetTask resolves a task within caller scope and returns sanitized/wrapped details.
func (a *Agent) GetTask(ctx context.Context, caller CallerContext, in GetTaskInput) (GetTaskOutput, error) {
	return a.svc.GetTask(ctx, caller, in)
}

// ListTasks lists tasks within the caller's allowed project scope.
func (a *Agent) ListTasks(ctx context.Context, caller CallerContext, in ListTasksInput) (ListTasksOutput, error) {
	return a.svc.ListTasks(ctx, caller, in)
}

// SetStatus transitions a task's status within caller scope.
func (a *Agent) SetStatus(ctx context.Context, caller CallerContext, in SetStatusInput) (SetStatusOutput, error) {
	return a.svc.SetStatus(ctx, caller, in)
}

// AddActivity appends a vibe-authored activity to a task within caller scope.
func (a *Agent) AddActivity(ctx context.Context, caller CallerContext, in AddActivityInput) (AddActivityOutput, error) {
	return a.svc.AddActivity(ctx, caller, in)
}

// SubmitReport validates, sanitizes, and persists the structured report on the bound Run.
func (a *Agent) SubmitReport(ctx context.Context, caller CallerContext, in SubmitReportInput) (SubmitReportOutput, error) {
	return a.svc.SubmitReport(ctx, caller, in)
}
