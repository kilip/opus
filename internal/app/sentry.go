package app

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

// InitSentry initializes Sentry client if SentryDSN is configured.
// It returns a flush function to be called on shutdown.
func InitSentry(cfg Config, logger *slog.Logger) func() {
	if cfg.SentryDSN == "" {
		return func() {}
	}
	opts := sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.SentryEnvironment,
		TracesSampleRate: cfg.SentryTracesSampleRate,
		EnableTracing:    true,
		AttachStacktrace: true,
	}
	if cfg.AppVersion != "" {
		opts.Release = cfg.AppName + "@" + cfg.AppVersion
	}
	err := sentry.Init(opts)
	if err != nil {
		logger.Error("failed to initialize sentry", "error", err)
		return func() {}
	}
	logger.Info("sentry initialized", "environment", cfg.SentryEnvironment)
	return func() {
		sentry.Flush(2 * time.Second)
	}
}

// ponytail: forwards slog.LevelError+ to Sentry via existing sentry-go without adding sentry-go/slog module.
type sentryAttr struct {
	prefix string
	attr   slog.Attr
}

type sentryErrorHandler struct {
	next   slog.Handler
	attrs  []sentryAttr
	groups []string
}

func newSentryErrorHandler(next slog.Handler) slog.Handler {
	return &sentryErrorHandler{next: next}
}

func (h *sentryErrorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *sentryErrorHandler) Handle(ctx context.Context, r slog.Record) error {
	err := h.next.Handle(ctx, r)
	if r.Level < slog.LevelError {
		return err
	}

	event := sentry.NewEvent()
	event.Level = sentry.LevelError
	event.Message = r.Message
	event.Timestamp = r.Time
	event.Logger = "slog"

	extra := make(map[string]any)
	for _, a := range h.attrs {
		addSentryAttr(extra, event, a.prefix, a.attr)
	}
	prefix := groupPrefix(h.groups)
	r.Attrs(func(a slog.Attr) bool {
		addSentryAttr(extra, event, prefix, a)
		return true
	})
	if len(extra) > 0 {
		event.Contexts = map[string]sentry.Context{"slog": extra}
	}

	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureEvent(event)
	} else {
		sentry.CaptureEvent(event)
	}
	return err
}

func (h *sentryErrorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	prefix := groupPrefix(h.groups)
	nextAttrs := slices.Clone(h.attrs)
	for _, a := range attrs {
		nextAttrs = append(nextAttrs, sentryAttr{prefix: prefix, attr: a})
	}
	return &sentryErrorHandler{
		next:   h.next.WithAttrs(attrs),
		attrs:  nextAttrs,
		groups: h.groups,
	}
}

func (h *sentryErrorHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &sentryErrorHandler{
		next:   h.next.WithGroup(name),
		attrs:  h.attrs,
		groups: append(slices.Clone(h.groups), name),
	}
}

func groupPrefix(groups []string) string {
	if len(groups) == 0 {
		return ""
	}
	return strings.Join(groups, ".") + "."
}

func addSentryAttr(dst map[string]any, event *sentry.Event, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	key := prefix + a.Key
	if a.Value.Kind() == slog.KindGroup {
		subPrefix := key + "."
		for _, sub := range a.Value.Group() {
			addSentryAttr(dst, event, subPrefix, sub)
		}
		return
	}
	if errVal, ok := a.Value.Any().(error); ok {
		dst[key] = errVal.Error()
		if len(event.Exception) == 0 {
			event.SetException(errVal, 5)
		}
		return
	}
	dst[key] = a.Value.Any()
}
