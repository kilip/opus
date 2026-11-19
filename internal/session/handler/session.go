package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/opus/internal/platform/httpx"
	"github.com/kilip/opus/internal/session/domain"
	"github.com/kilip/opus/internal/session/service"
)

const defaultSSEKeepaliveInterval = 15 * time.Second

// Handler serves session REST and SSE endpoints.
type Handler struct {
	svc               *service.Service
	keepaliveInterval time.Duration
}

// New creates a new session Handler.
func New(svc *service.Service) *Handler {
	return &Handler{
		svc:               svc,
		keepaliveInterval: defaultSSEKeepaliveInterval,
	}
}

// SetKeepaliveInterval overrides the SSE keepalive interval (for tests).
func (h *Handler) SetKeepaliveInterval(d time.Duration) {
	if d > 0 {
		h.keepaliveInterval = d
	}
}

// Register mounts protected session routes on r.
func (h *Handler) Register(r fiber.Router) {
	g := r.Group("/sessions")
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Get("/:id/events", h.Events)
}

// List godoc
//
//	@Summary		List sessions
//	@Description	Returns sessions filtered by optional taskId, nodeId, and status.
//	@ID				list-sessions
//	@Tags			sessions
//	@Produce		json
//	@Param			taskId	query		string	false	"Filter by task ID"
//	@Param			nodeId	query		string	false	"Filter by node ID"
//	@Param			status	query		string	false	"Filter by session status"
//	@Success		200		{object}	httpx.Response[[]domain.SessionResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/sessions [get]
func (h *Handler) List(c fiber.Ctx) error {
	var filter domain.ListFilter
	if raw := c.Query("taskId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return httpx.BadRequest(err)
		}
		filter.TaskID = &id
	}
	if raw := c.Query("nodeId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return httpx.BadRequest(err)
		}
		filter.NodeID = &id
	}
	if raw := c.Query("status"); raw != "" {
		st := domain.SessionStatus(raw)
		if !domain.IsValidStatus(st) {
			return httpx.BadRequest(domain.ErrInvalidStatus)
		}
		filter.Status = &st
	}

	sessions, err := h.svc.List(c.Context(), filter)
	if err != nil {
		return mapErr(err)
	}

	resp := make([]domain.SessionResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = domain.ToSessionResponse(s)
	}
	total := len(resp)
	return httpx.List(c, resp, &httpx.Meta{Total: &total})
}

// Get godoc
//
//	@Summary		Get a session
//	@Description	Returns a single session by its identifier.
//	@ID				get-session
//	@Tags			sessions
//	@Produce		json
//	@Param			id	path		string	true	"Session ID"
//	@Success		200	{object}	httpx.Response[domain.SessionResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Router			/sessions/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.BadRequest(domain.ErrInvalidSessionID)
	}
	sess, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return mapErr(err)
	}
	return httpx.OK(c, domain.ToSessionResponse(sess))
}

// Events godoc
//
//	@Summary		Stream session events (SSE)
//	@Description	Streams ordered historical events (seq > after_seq) then live events via SSE until terminal status or disconnect.
//	@ID				stream-session-events
//	@Tags			sessions
//	@Produce		text/event-stream
//	@Param			id			path		string	true	"Session ID"
//	@Param			after_seq	query		int		false	"Return events with seq greater than after_seq"
//	@Success		200			{string}	string	"SSE event stream"
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		500			{object}	httpx.ErrorResponse
//	@Router			/sessions/{id}/events [get]
func (h *Handler) Events(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httpx.BadRequest(domain.ErrInvalidSessionID)
	}

	sess, err := h.svc.Get(c.Context(), id)
	if err != nil {
		return mapErr(err)
	}

	afterSeq := 0
	if raw := c.Query("after_seq"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return httpx.BadRequest(errors.New("invalid after_seq"))
		}
		afterSeq = n
	} else if lastID := c.Get("Last-Event-ID"); lastID != "" {
		if n, err := strconv.Atoi(lastID); err == nil && n >= 0 {
			afterSeq = n
		}
	}

	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, "keep-alive")
	if conn := c.RequestCtx().Conn(); conn != nil {
		_ = conn.SetWriteDeadline(time.Time{})
	}

	// Subscribe before reading history so no live events are missed in the gap.
	subCh, unsubscribe := h.svc.Subscribe(id)
	keepalive := h.keepaliveInterval
	if keepalive <= 0 {
		keepalive = defaultSSEKeepaliveInterval
	}

	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer unsubscribe()

		history, err := h.svc.ListEventsAfter(context.Background(), id, afterSeq)
		if err != nil {
			return
		}

		lastSent := afterSeq
		for _, evt := range history {
			if err := writeSSEEvent(w, evt); err != nil {
				return
			}
			lastSent = evt.Seq
		}

		if domain.IsTerminalStatus(sess.Status) {
			return
		}

		ticker := time.NewTicker(keepalive)
		defer ticker.Stop()

		for {
			select {
			case evt, ok := <-subCh:
				if !ok {
					return
				}
				if evt.Seq <= lastSent {
					continue
				}
				if err := writeSSEEvent(w, evt); err != nil {
					return
				}
				lastSent = evt.Seq
				if isTerminalEventKind(evt.Kind) {
					return
				}
			case <-ticker.C:
				if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}

func writeSSEEvent(w *bufio.Writer, evt domain.SessionEvent) error {
	data, err := json.Marshal(domain.ToSessionEventResponse(evt))
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", evt.Seq, evt.Kind, data); err != nil {
		return err
	}
	return w.Flush()
}

func isTerminalEventKind(kind string) bool {
	switch kind {
	case domain.EventCompleted, domain.EventFailed, domain.EventCancelled:
		return true
	default:
		return false
	}
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrSessionNotFound):
		return httpx.NotFound(err)
	case errors.Is(err, domain.ErrInvalidStatus),
		errors.Is(err, domain.ErrProviderRequired),
		errors.Is(err, domain.ErrInvalidSessionID),
		errors.Is(err, domain.ErrEventKindEmpty):
		return httpx.BadRequest(err)
	default:
		return err
	}
}
