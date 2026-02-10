package llmguard

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
)

// Guard implements guard.Guard for LLM Guard.
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

// WithBaseURL sets the LLM Guard API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the LLM Guard API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new LLM Guard guard.
func New(opts ...Option) (*Guard, error) {
	cfg := &config{
		baseURL: "http://localhost:8000",
		timeout: 15 * time.Second,
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
		client: httpclient.New(clientOpts...),
	}, nil
}

// analyzeRequest is the LLM Guard analyze API request.
type analyzeRequest struct {
	Prompt string `json:"prompt"`
}

// analyzeOutputRequest is the LLM Guard analyze/output API request.
type analyzeOutputRequest struct {
	Prompt string `json:"prompt"`
	Output string `json:"output"`
}

// analyzeResponse is the LLM Guard analyze API response.
type analyzeResponse struct {
	IsValid    bool             `json:"is_valid"`
	Scanners   []scannerResult  `json:"scanners"`
	SanitizedPrompt string     `json:"sanitized_prompt,omitempty"`
	SanitizedOutput string     `json:"sanitized_output,omitempty"`
}

// scannerResult holds an individual scanner's outcome.
type scannerResult struct {
	Name      string  `json:"name"`
	Score     float64 `json:"score"`
	IsValid   bool    `json:"is_valid"`
	Threshold float64 `json:"threshold"`
}

// Name returns the guard name.
func (g *Guard) Name() string {
	return "llm_guard"
}

// Validate sends the content to LLM Guard for validation. It uses the
// /analyze/prompt endpoint for input and /analyze/output for output content.
func (g *Guard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	var resp analyzeResponse
	var err error

	if input.Role == "output" {
		req := analyzeOutputRequest{
			Prompt: "",
			Output: input.Content,
		}
		resp, err = httpclient.DoJSON[analyzeResponse](ctx, g.client, "POST", "/analyze/output", req)
	} else {
		req := analyzeRequest{
			Prompt: input.Content,
		}
		resp, err = httpclient.DoJSON[analyzeResponse](ctx, g.client, "POST", "/analyze/prompt", req)
	}

	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("llmguard: validate: %w", err)
	}

	result := guard.GuardResult{
		Allowed:   resp.IsValid,
		GuardName: g.Name(),
	}

	if !resp.IsValid {
		// Find the first failing scanner for the reason.
		for _, s := range resp.Scanners {
			if !s.IsValid {
				result.Reason = fmt.Sprintf("scanner %s failed (score=%.2f, threshold=%.2f)", s.Name, s.Score, s.Threshold)
				break
			}
		}
		if result.Reason == "" {
			result.Reason = "blocked by LLM Guard"
		}
	}

	// Use sanitized content if available and different.
	if input.Role == "output" && resp.SanitizedOutput != "" && resp.SanitizedOutput != input.Content {
		result.Modified = resp.SanitizedOutput
	} else if resp.SanitizedPrompt != "" && resp.SanitizedPrompt != input.Content {
		result.Modified = resp.SanitizedPrompt
	}

	return result, nil
}

// compile-time interface check
var _ guard.Guard = (*Guard)(nil)
