package lakera

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
)

// Guard implements guard.Guard for Lakera Guard.
type Guard struct {
	client *httpclient.Client
}

// Option configures a Guard.
type Option func(*config)

type config struct {
	baseURL string
	apiKey  string
	timeout time.Duration
}

// WithBaseURL sets the Lakera Guard API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the Lakera Guard API key.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Lakera Guard guard.
func New(opts ...Option) (*Guard, error) {
	cfg := &config{
		baseURL: "https://api.lakera.ai",
		timeout: 15 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiKey == "" {
		return nil, fmt.Errorf("lakera: API key is required")
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithTimeout(cfg.timeout),
		httpclient.WithBearerToken(cfg.apiKey),
	}

	return &Guard{
		client: httpclient.New(clientOpts...),
	}, nil
}

// guardRequest is the Lakera Guard /v1/guard request.
type guardRequest struct {
	Input    string         `json:"input"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// guardResponse is the Lakera Guard /v1/guard response.
type guardResponse struct {
	Flagged    bool             `json:"flagged"`
	Categories []categoryResult `json:"categories"`
	Model      string           `json:"model"`
}

// categoryResult holds an individual category's result.
type categoryResult struct {
	Category string  `json:"category"`
	Flagged  bool    `json:"flagged"`
	Score    float64 `json:"score"`
}

// Name returns the guard name.
func (g *Guard) Name() string {
	return "lakera_guard"
}

// Validate sends the content to Lakera Guard for validation.
func (g *Guard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	req := guardRequest{
		Input:    input.Content,
		Metadata: input.Metadata,
	}

	resp, err := httpclient.DoJSON[guardResponse](ctx, g.client, "POST", "/v1/guard", req)
	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("lakera: validate: %w", err)
	}

	result := guard.GuardResult{
		Allowed:   !resp.Flagged,
		GuardName: g.Name(),
	}

	if resp.Flagged {
		var flagged []string
		for _, c := range resp.Categories {
			if c.Flagged {
				flagged = append(flagged, c.Category)
			}
		}
		if len(flagged) > 0 {
			result.Reason = "flagged categories: " + strings.Join(flagged, ", ")
		} else {
			result.Reason = "blocked by Lakera Guard"
		}
	}

	return result, nil
}

// compile-time interface check
var _ guard.Guard = (*Guard)(nil)
