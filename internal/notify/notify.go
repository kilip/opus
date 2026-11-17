// Package notify delivers external notifications for Opus domain events.
package notify

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/run"
)

const (
	defaultTelegramBaseURL = "https://api.telegram.org"
	defaultHTTPTimeout     = 5 * time.Second
	subscriberName         = "notify-telegram"
)

// Config configures the Telegram notifier.
type Config struct {
	BotToken   string
	ChatID     string
	BaseURL    string
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// Notify subscribes to domain events and sends notifications when configured.
type Notify struct {
	botToken   string
	chatID     string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// New creates a Notify instance. When BotToken or ChatID is empty, the notifier
// is disabled and performs no deliveries.
func New(cfg Config) *Notify {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultTelegramBaseURL
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Notify{
		botToken:   cfg.BotToken,
		chatID:     cfg.ChatID,
		baseURL:    baseURL,
		httpClient: client,
		logger:     logger,
	}
}

// Enabled reports whether Telegram delivery is configured.
func (n *Notify) Enabled() bool {
	return n.botToken != "" && n.chatID != ""
}

// Register subscribes to run.finished and node.offline on bus if enabled.
func (n *Notify) Register(bus *eventbus.Bus) error {
	if bus == nil || !n.Enabled() {
		return nil
	}
	return bus.Subscribe(subscriberName, n.HandleEvent, run.EventFinished, node.EventOffline)
}

// HandleEvent formats and delivers a notification for a subscribed event.
// Delivery errors are logged and never returned so the eventbus is not blocked.
func (n *Notify) HandleEvent(ctx context.Context, e eventbus.Event) error {
	if !n.Enabled() {
		return nil
	}

	msg, err := formatMessage(e)
	if err != nil {
		n.logger.ErrorContext(ctx, "failed to format notification",
			slog.String("event_type", e.Type),
			slog.Any("error", err),
		)
		return nil
	}
	if msg == "" {
		return nil
	}

	if err := n.sendTelegram(ctx, msg); err != nil {
		n.logger.ErrorContext(ctx, "telegram notification failed",
			slog.String("event_type", e.Type),
			slog.Any("error", err),
		)
	}
	return nil
}
