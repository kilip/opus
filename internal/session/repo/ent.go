package repo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/opus/ent"
	"github.com/kilip/opus/ent/session"
	"github.com/kilip/opus/ent/sessionevent"
	"github.com/kilip/opus/internal/session/domain"
)

// Ent implements session persistence backed by an Ent client.
type Ent struct {
	db *ent.Client
}

// New creates a new Ent session repository.
func New(db *ent.Client) *Ent {
	return &Ent{db: db}
}

// EnqueueOrStart atomically creates a session as running if no other session is
// currently running, or as queued otherwise.
func (r *Ent) EnqueueOrStart(ctx context.Context, in domain.CreateInput, now time.Time) (domain.Session, error) {
	account := in.ProviderAccount
	if account == "" {
		account = "default"
	}

	tx, err := r.db.Tx(ctx)
	if err != nil {
		return domain.Session{}, err
	}

	// ponytail: single-account global FIFO slot queue for MVP; shard by provider_account when multi-account pooling lands.
	hasRunning, err := tx.Session.Query().
		Where(session.StatusEQ(session.StatusRunning)).
		Exist(ctx)
	if err != nil {
		_ = tx.Rollback()
		return domain.Session{}, err
	}

	create := tx.Session.Create().
		SetTaskID(in.TaskID).
		SetNodeID(in.NodeID).
		SetProvider(in.Provider).
		SetProviderAccount(account).
		SetCreatedAt(now).
		SetUpdatedAt(now)

	if hasRunning {
		create.SetStatus(session.StatusQueued)
	} else {
		create.SetStatus(session.StatusRunning).SetStartedAt(now)
	}

	row, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return domain.Session{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Session{}, err
	}
	return toDomainSession(row), nil
}

// TransitionTerminalAndPromoteNext transitions sessionID to a terminal status
// and, if no other session is running, promotes the oldest queued session to running.
func (r *Ent) TransitionTerminalAndPromoteNext(
	ctx context.Context,
	sessionID uuid.UUID,
	status domain.SessionStatus,
	failureReason string,
	now time.Time,
) (domain.Session, *domain.Session, error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return domain.Session{}, nil, err
	}

	upd := tx.Session.UpdateOneID(sessionID).
		SetStatus(session.Status(status)).
		SetEndedAt(now).
		SetUpdatedAt(now)
	if failureReason != "" {
		upd.SetFailureReason(failureReason)
	}

	endedRow, err := upd.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return domain.Session{}, nil, domain.ErrSessionNotFound
		}
		return domain.Session{}, nil, err
	}

	hasRunning, err := tx.Session.Query().
		Where(session.StatusEQ(session.StatusRunning)).
		Exist(ctx)
	if err != nil {
		_ = tx.Rollback()
		return domain.Session{}, nil, err
	}

	var promoted *domain.Session
	if !hasRunning {
		// ponytail: single-account global FIFO slot queue for MVP; shard by provider_account when multi-account pooling lands.
		nextRow, err := tx.Session.Query().
			Where(session.StatusEQ(session.StatusQueued)).
			Order(ent.Asc(session.FieldCreatedAt), ent.Asc(session.FieldID)).
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			_ = tx.Rollback()
			return domain.Session{}, nil, err
		}
		if nextRow != nil {
			promotedRow, err := tx.Session.UpdateOneID(nextRow.ID).
				SetStatus(session.StatusRunning).
				SetStartedAt(now).
				SetUpdatedAt(now).
				Save(ctx)
			if err != nil {
				_ = tx.Rollback()
				return domain.Session{}, nil, err
			}
			p := toDomainSession(promotedRow)
			promoted = &p
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Session{}, nil, err
	}
	return toDomainSession(endedRow), promoted, nil
}

// Get returns a session by ID or domain.ErrSessionNotFound.
func (r *Ent) Get(ctx context.Context, id uuid.UUID) (domain.Session, error) {
	row, err := r.db.Session.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Session{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, err
	}
	return toDomainSession(row), nil
}

// List returns sessions matching filter, ordered newest-first.
func (r *Ent) List(ctx context.Context, filter domain.ListFilter) ([]domain.Session, error) {
	q := r.db.Session.Query()
	if filter.TaskID != nil {
		q = q.Where(session.TaskIDEQ(*filter.TaskID))
	}
	if filter.NodeID != nil {
		q = q.Where(session.NodeIDEQ(*filter.NodeID))
	}
	if filter.Status != nil {
		q = q.Where(session.StatusEQ(session.Status(*filter.Status)))
	}

	rows, err := q.Order(ent.Desc(session.FieldCreatedAt), ent.Desc(session.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Session, len(rows))
	for i, row := range rows {
		out[i] = toDomainSession(row)
	}
	return out, nil
}

// AppendEvent persists a new SessionEvent with the next monotonic 1-based seq for sessionID.
func (r *Ent) AppendEvent(ctx context.Context, sessionID uuid.UUID, kind string, payload json.RawMessage, at time.Time) (domain.SessionEvent, error) {
	tx, err := r.db.Tx(ctx)
	if err != nil {
		return domain.SessionEvent{}, err
	}

	exists, err := tx.Session.Query().Where(session.IDEQ(sessionID)).Exist(ctx)
	if err != nil {
		_ = tx.Rollback()
		return domain.SessionEvent{}, err
	}
	if !exists {
		_ = tx.Rollback()
		return domain.SessionEvent{}, domain.ErrSessionNotFound
	}

	nextSeq := 1
	last, err := tx.SessionEvent.Query().
		Where(sessionevent.SessionIDEQ(sessionID)).
		Order(ent.Desc(sessionevent.FieldSeq)).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		_ = tx.Rollback()
		return domain.SessionEvent{}, err
	}
	if last != nil {
		nextSeq = last.Seq + 1
	}

	create := tx.SessionEvent.Create().
		SetSessionID(sessionID).
		SetSeq(nextSeq).
		SetKind(kind).
		SetAt(at)
	if len(payload) > 0 {
		create.SetPayload(payload)
	}

	row, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return domain.SessionEvent{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.SessionEvent{}, err
	}
	return toDomainEvent(row), nil
}

// ListEventsAfter returns all events for sessionID with seq > afterSeq ordered by seq ascending.
func (r *Ent) ListEventsAfter(ctx context.Context, sessionID uuid.UUID, afterSeq int) ([]domain.SessionEvent, error) {
	rows, err := r.db.SessionEvent.Query().
		Where(
			sessionevent.SessionIDEQ(sessionID),
			sessionevent.SeqGT(afterSeq),
		).
		Order(ent.Asc(sessionevent.FieldSeq)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]domain.SessionEvent, len(rows))
	for i, row := range rows {
		out[i] = toDomainEvent(row)
	}
	return out, nil
}

// LatestEventTime returns the CreatedAt timestamp of the most recent event for sessionID, or nil if none exist.
func (r *Ent) LatestEventTime(ctx context.Context, sessionID uuid.UUID) (*time.Time, error) {
	row, err := r.db.SessionEvent.Query().
		Where(sessionevent.SessionIDEQ(sessionID)).
		Order(ent.Desc(sessionevent.FieldSeq)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	t := row.At
	return &t, nil
}

func toDomainSession(s *ent.Session) domain.Session {
	return domain.Session{
		ID:              s.ID,
		TaskID:          s.TaskID,
		NodeID:          s.NodeID,
		Provider:        s.Provider,
		ProviderAccount: s.ProviderAccount,
		Status:          domain.SessionStatus(s.Status),
		FailureReason:   s.FailureReason,
		CreatedAt:       s.CreatedAt,
		StartedAt:       s.StartedAt,
		EndedAt:         s.EndedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

func toDomainEvent(e *ent.SessionEvent) domain.SessionEvent {
	return domain.SessionEvent{
		ID:        e.ID,
		SessionID: e.SessionID,
		Seq:       e.Seq,
		Kind:      e.Kind,
		Payload:   e.Payload,
		At:        e.At,
	}
}
