package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kilip/opus/internal/node"
	"github.com/kilip/opus/internal/platform/eventbus"
	"github.com/kilip/opus/internal/run"
)

type telegramSendMessageRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func formatMessage(e eventbus.Event) (string, error) {
	switch e.Type {
	case run.EventFinished:
		var p run.RunFinishedPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return "", fmt.Errorf("decode %s: %w", e.Type, err)
		}
		if p.Reason != "" {
			return fmt.Sprintf("Run %s (%s) finished: %s (task %s) — %s",
				p.RunID, p.Mode, p.Outcome, p.TaskID, p.Reason), nil
		}
		return fmt.Sprintf("Run %s (%s) finished: %s (task %s)",
			p.RunID, p.Mode, p.Outcome, p.TaskID), nil

	case node.EventOffline:
		var p node.NodeEventPayload
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return "", fmt.Errorf("decode %s: %w", e.Type, err)
		}
		return fmt.Sprintf("Node offline: %s (%s)", p.Name, p.ID), nil

	default:
		return "", nil
	}
}

func (n *Notify) sendTelegram(ctx context.Context, text string) error {
	body, err := json.Marshal(telegramSendMessageRequest{
		ChatID: n.chatID,
		Text:   text,
	})
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/bot%s/sendMessage", strings.TrimRight(n.baseURL, "/"), n.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram api returned status %d", resp.StatusCode)
	}
	return nil
}
