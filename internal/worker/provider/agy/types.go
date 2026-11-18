package agy

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors returned by the ACP client.
var (
	ErrUnauthenticated = errors.New("agy: authentication required")
	ErrQuotaExhausted  = errors.New("agy: quota exhausted")
	ErrClientClosed    = errors.New("agy: client closed")
)

// RPCError represents a JSON-RPC 2.0 error returned by agy acp.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("acp rpc error (%d): %s", e.Code, e.Message)
}

// ClassifyRPCError maps an ACP JSON-RPC error to ErrUnauthenticated or ErrQuotaExhausted
// when applicable, preserving the original RPCError via wrapping.
func ClassifyRPCError(code int, msg string) error {
	base := &RPCError{Code: code, Message: msg}
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "authentication required") || strings.Contains(lower, "unauthenticated"):
		return fmt.Errorf("%w: %w", ErrUnauthenticated, base)
	case strings.Contains(lower, "resource_exhausted") ||
		strings.Contains(lower, "exhausted your capacity") ||
		strings.Contains(lower, "quota") ||
		strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "429"):
		return fmt.Errorf("%w: %w", ErrQuotaExhausted, base)
	default:
		return base
	}
}

// ClientInfo identifies the ACP client during initialize.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams is sent with the initialize request.
type InitializeParams struct {
	ProtocolVersion    int            `json:"protocolVersion"`
	ClientInfo         ClientInfo     `json:"clientInfo"`
	ClientCapabilities map[string]any `json:"clientCapabilities"`
}

// AgentInfo describes the ACP server.
type AgentInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// AuthMethod describes an authentication method advertised by agy acp.
type AuthMethod struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// InitializeResult is returned by initialize.
type InitializeResult struct {
	ProtocolVersion   int            `json:"protocolVersion"`
	AgentCapabilities map[string]any `json:"agentCapabilities,omitempty"`
	AgentInfo         AgentInfo      `json:"agentInfo"`
	AuthMethods       []AuthMethod   `json:"authMethods,omitempty"`
}

// MCPHeader defines an HTTP header sent to an HTTP MCP server.
type MCPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MCPServer defines a session-scoped MCP server passed to session/new or session/resume.
type MCPServer struct {
	Name    string            `json:"name,omitempty"`
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers []MCPHeader       `json:"headers,omitempty"`
}

// SystemPromptMeta configures per-session system prompt appending under _meta.antigravity.
type SystemPromptMeta struct {
	Append string `json:"append,omitempty"`
}

// AntigravityMeta holds Antigravity-specific session metadata.
type AntigravityMeta struct {
	SystemPrompt *SystemPromptMeta `json:"systemPrompt,omitempty"`
}

// SessionMeta is the _meta envelope for session/new.
type SessionMeta struct {
	Antigravity *AntigravityMeta `json:"antigravity,omitempty"`
}

// SessionNewParams is sent with session/new.
type SessionNewParams struct {
	Cwd        string       `json:"cwd"`
	MCPServers []MCPServer  `json:"mcpServers"`
	Meta       *SessionMeta `json:"_meta,omitempty"`
}

// SessionResumeParams is sent with session/resume.
type SessionResumeParams struct {
	SessionID  string      `json:"sessionId"`
	Cwd        string      `json:"cwd"`
	MCPServers []MCPServer `json:"mcpServers"`
}

// SessionNewResult is returned by session/new and session/resume.
type SessionNewResult struct {
	SessionID     string          `json:"sessionId"`
	Models        json.RawMessage `json:"models,omitempty"`
	Modes         json.RawMessage `json:"modes,omitempty"`
	ConfigOptions json.RawMessage `json:"configOptions,omitempty"`
	Meta          json.RawMessage `json:"_meta,omitempty"`
}

// PromptContent represents a content block in session/prompt.
type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// SessionPromptParams is sent with session/prompt.
type SessionPromptParams struct {
	SessionID string          `json:"sessionId"`
	Prompt    []PromptContent `json:"prompt"`
}

// Usage reports token counts for a completed prompt turn.
type Usage struct {
	InputTokens       int `json:"inputTokens"`
	OutputTokens      int `json:"outputTokens"`
	CachedReadTokens  int `json:"cachedReadTokens"`
	CachedWriteTokens int `json:"cachedWriteTokens"`
	ThoughtTokens     int `json:"thoughtTokens"`
	TotalTokens       int `json:"totalTokens"`
}

// SessionPromptResult is returned when session/prompt completes.
type SessionPromptResult struct {
	StopReason string `json:"stopReason"`
	Usage      *Usage `json:"usage,omitempty"`
}

// SessionUpdateParams is the params payload of a session/update notification.
type SessionUpdateParams struct {
	SessionID string        `json:"sessionId"`
	Update    SessionUpdate `json:"update"`
}

// SessionUpdate represents an incremental update emitted during a turn.
type SessionUpdate struct {
	SessionUpdate string          `json:"sessionUpdate"`
	Title         string          `json:"title,omitempty"`
	ToolCallID    string          `json:"toolCallId,omitempty"`
	Name          string          `json:"name,omitempty"`
	Status        string          `json:"status,omitempty"`
	Kind          string          `json:"kind,omitempty"`
	Args          json.RawMessage `json:"args,omitempty"`
	Content       json.RawMessage `json:"content,omitempty"`
	RawInput      json.RawMessage `json:"rawInput,omitempty"`
	RawOutput     json.RawMessage `json:"rawOutput,omitempty"`
	Used          int             `json:"used,omitempty"`
	Size          int             `json:"size,omitempty"`
}

// PermissionOption describes one selectable choice in session/request_permission.
type PermissionOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

// PermissionRequest is sent by agy acp for session/request_permission.
type PermissionRequest struct {
	SessionID string             `json:"sessionId"`
	ToolCall  json.RawMessage    `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
}

// PermissionOutcome is the selected outcome returned for session/request_permission.
type PermissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}

// PermissionResponse is the JSON-RPC result returned to session/request_permission.
type PermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

// QuestionItem represents a single question in an ask_question form.
type QuestionItem struct {
	Question      string   `json:"question"`
	IsMultiSelect bool     `json:"is_multi_select,omitempty"`
	Options       []string `json:"options,omitempty"`
}

// QuestionRequest represents an ask_question prompt from the agent.
type QuestionRequest struct {
	SessionID  string         `json:"sessionId"`
	ToolCallID string         `json:"toolCallId,omitempty"`
	Questions  []QuestionItem `json:"questions"`
}

// QuestionResponse is returned by the caller's OnQuestion callback.
type QuestionResponse struct {
	Answers map[string]any `json:"answers,omitempty"`
	Skipped bool           `json:"skipped,omitempty"`
}
