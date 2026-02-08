// Package nemo provides an NVIDIA NeMo Guardrails guard implementation for the
// Beluga AI safety pipeline. It implements the guard.Guard interface and sends
// content validation requests to a NeMo Guardrails API endpoint.
//
// NeMo Guardrails can be configured to check for topic safety, jailbreak
// detection, fact-checking, and more via Colang configurations.
//
// Usage:
//
//	g, err := nemo.New(
//	    nemo.WithBaseURL("http://localhost:8080"),
//	    nemo.WithConfigID("my-config"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := g.Validate(ctx, input)
package nemo

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
)

// Guard implements guard.Guard for NVIDIA NeMo Guardrails.
type Guard struct {
	client   *httpclient.Client
	configID string
}

// Option configures a Guard.
type Option func(*config)

type config struct {
	baseURL  string
	apiKey   string
	configID string
	timeout  time.Duration
}

// WithBaseURL sets the NeMo Guardrails API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the NeMo Guardrails API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithConfigID sets the NeMo Guardrails configuration ID.
func WithConfigID(id string) Option {
	return func(c *config) { c.configID = id }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new NeMo Guardrails guard.
func New(opts ...Option) (*Guard, error) {
	cfg := &config{
		baseURL:  "http://localhost:8080",
		configID: "default",
		timeout:  15 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithTimeout(cfg.timeout),
	}
	if cfg.apiKey != "" {
		clientOpts = append(clientOpts, httpclient.WithBearerToken(cfg.apiKey))
	}

	return &Guard{
		client:   httpclient.New(clientOpts...),
		configID: cfg.configID,
	}, nil
}

// chatRequest is the NeMo Guardrails /v1/chat/completions request.
type chatRequest struct {
	ConfigID string        `json:"config_id"`
	Messages []chatMessage `json:"messages"`
}

// chatMessage is a NeMo chat message.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the NeMo Guardrails /v1/chat/completions response.
type chatResponse struct {
	Messages []chatMessage `json:"messages,omitempty"`
	Response []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"response,omitempty"`
	Guardrails guardrailsResult `json:"guardrails"`
}

// guardrailsResult carries the NeMo guardrails evaluation outcome.
type guardrailsResult struct {
	Blocked bool   `json:"blocked"`
	Reason  string `json:"reason,omitempty"`
	Score   float64 `json:"score,omitempty"`
}

// Name returns the guard name.
func (g *Guard) Name() string {
	return "nemo_guardrails"
}

// Validate sends the content to NeMo Guardrails for validation. It maps the
// GuardInput.Role to the NeMo message role and returns a GuardResult based on
// the guardrails evaluation.
func (g *Guard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	role := "user"
	if input.Role == "output" {
		role = "assistant"
	}

	req := chatRequest{
		ConfigID: g.configID,
		Messages: []chatMessage{
			{
				Role:    role,
				Content: input.Content,
			},
		},
	}

	resp, err := httpclient.DoJSON[chatResponse](ctx, g.client, "POST", "/v1/chat/completions", req)
	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("nemo: validate: %w", err)
	}

	result := guard.GuardResult{
		Allowed:   !resp.Guardrails.Blocked,
		GuardName: g.Name(),
	}

	if resp.Guardrails.Blocked {
		result.Reason = resp.Guardrails.Reason
		if result.Reason == "" {
			result.Reason = "blocked by NeMo Guardrails"
		}
	}

	// If NeMo returned a modified response, use it.
	if len(resp.Response) > 0 && resp.Response[0].Content != "" && resp.Response[0].Content != input.Content {
		result.Modified = resp.Response[0].Content
	} else if len(resp.Messages) > 0 {
		lastMsg := resp.Messages[len(resp.Messages)-1]
		if lastMsg.Content != "" && lastMsg.Content != input.Content {
			result.Modified = lastMsg.Content
		}
	}

	return result, nil
}

// compile-time interface check
var _ guard.Guard = (*Guard)(nil)
