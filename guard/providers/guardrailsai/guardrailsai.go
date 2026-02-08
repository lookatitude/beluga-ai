// Package guardrailsai provides a Guardrails AI guard implementation for the
// Beluga AI safety pipeline. It implements the guard.Guard interface and sends
// content validation requests to a Guardrails AI API endpoint.
//
// Guardrails AI provides validators for PII detection, toxicity, hallucination,
// prompt injection, and custom rules defined via RAIL specifications.
//
// Usage:
//
//	g, err := guardrailsai.New(
//	    guardrailsai.WithBaseURL("http://localhost:8000"),
//	    guardrailsai.WithGuardName("my-guard"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := g.Validate(ctx, input)
package guardrailsai

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
)

// Guard implements guard.Guard for Guardrails AI.
type Guard struct {
	client    *httpclient.Client
	guardName string
}

// Option configures a Guard.
type Option func(*config)

type config struct {
	baseURL   string
	apiKey    string
	guardName string
	timeout   time.Duration
}

// WithBaseURL sets the Guardrails AI API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the Guardrails AI API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithGuardName sets the guard name to invoke on the Guardrails AI server.
func WithGuardName(name string) Option {
	return func(c *config) { c.guardName = name }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Guardrails AI guard.
func New(opts ...Option) (*Guard, error) {
	cfg := &config{
		baseURL:   "http://localhost:8000",
		guardName: "default",
		timeout:   15 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.guardName == "" {
		return nil, fmt.Errorf("guardrailsai: guard name is required")
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithTimeout(cfg.timeout),
	}
	if cfg.apiKey != "" {
		clientOpts = append(clientOpts, httpclient.WithBearerToken(cfg.apiKey))
	}

	return &Guard{
		client:    httpclient.New(clientOpts...),
		guardName: cfg.guardName,
	}, nil
}

// validateRequest is the Guardrails AI /guards/{guardName}/validate request.
type validateRequest struct {
	LLMOutput string         `json:"llmOutput,omitempty"`
	Prompt    string         `json:"prompt,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// validateResponse is the Guardrails AI validate response.
type validateResponse struct {
	Result       string           `json:"result"`
	ValidatedOutput string        `json:"validatedOutput,omitempty"`
	RawOutput    string           `json:"rawLlmOutput,omitempty"`
	Validations  []validationItem `json:"validationsPassed,omitempty"`
	Failed       []validationItem `json:"validationsFailed,omitempty"`
}

// validationItem represents a single validator result.
type validationItem struct {
	ValidatorName string `json:"validatorName"`
	Result        string `json:"result"`
	Message       string `json:"message,omitempty"`
}

// Name returns the guard name.
func (g *Guard) Name() string {
	return "guardrails_ai"
}

// Validate sends the content to Guardrails AI for validation.
func (g *Guard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	req := validateRequest{
		Metadata: input.Metadata,
	}

	// Map role to the appropriate field.
	switch input.Role {
	case "input":
		req.Prompt = input.Content
	default:
		req.LLMOutput = input.Content
	}

	path := fmt.Sprintf("/guards/%s/validate", g.guardName)
	resp, err := httpclient.DoJSON[validateResponse](ctx, g.client, "POST", path, req)
	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("guardrailsai: validate: %w", err)
	}

	allowed := resp.Result == "pass"

	result := guard.GuardResult{
		Allowed:   allowed,
		GuardName: g.Name(),
	}

	if !allowed {
		if len(resp.Failed) > 0 {
			result.Reason = resp.Failed[0].Message
			if result.Reason == "" {
				result.Reason = "failed validator: " + resp.Failed[0].ValidatorName
			}
		} else {
			result.Reason = "validation failed"
		}
	}

	if resp.ValidatedOutput != "" && resp.ValidatedOutput != input.Content {
		result.Modified = resp.ValidatedOutput
	}

	return result, nil
}

// compile-time interface check
var _ guard.Guard = (*Guard)(nil)
