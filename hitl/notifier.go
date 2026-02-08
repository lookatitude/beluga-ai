package hitl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Notifier is the interface for sending interaction notifications to humans.
type Notifier interface {
	// Notify sends a notification about a pending interaction request.
	Notify(ctx context.Context, req InteractionRequest) error
}

// LogNotifier implements Notifier by logging interaction requests via slog.
type LogNotifier struct {
	logger *slog.Logger
}

// NewLogNotifier creates a new LogNotifier with the given slog logger.
// If logger is nil, slog.Default() is used.
func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogNotifier{logger: logger}
}

// Notify logs the interaction request at the Warn level.
func (n *LogNotifier) Notify(_ context.Context, req InteractionRequest) error {
	n.logger.Warn("human interaction required",
		"request_id", req.ID,
		"type", string(req.Type),
		"tool", req.ToolName,
		"description", req.Description,
		"confidence", req.Confidence,
		"risk_level", string(req.RiskLevel),
	)
	return nil
}

// WebhookNotifier implements Notifier by sending HTTP POST requests to a
// webhook URL.
type WebhookNotifier struct {
	url    string
	client *http.Client
}

// NewWebhookNotifier creates a new WebhookNotifier pointing at the given URL.
// It uses http.DefaultClient for sending requests.
func NewWebhookNotifier(url string) *WebhookNotifier {
	return &WebhookNotifier{
		url:    url,
		client: http.DefaultClient,
	}
}

// NewWebhookNotifierWithClient creates a new WebhookNotifier with a custom
// HTTP client.
func NewWebhookNotifierWithClient(url string, client *http.Client) *WebhookNotifier {
	return &WebhookNotifier{
		url:    url,
		client: client,
	}
}

// Notify sends the interaction request to the configured webhook URL as JSON.
func (n *WebhookNotifier) Notify(ctx context.Context, req InteractionRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("hitl/webhook: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("hitl/webhook: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("hitl/webhook: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("hitl/webhook: unexpected status %d", resp.StatusCode)
	}
	return nil
}
