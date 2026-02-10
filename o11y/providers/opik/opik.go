package opik

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/o11y"
)

// Exporter implements o11y.TraceExporter for Opik.
type Exporter struct {
	client    *httpclient.Client
	workspace string
}

// Option configures an Exporter.
type Option func(*config)

type config struct {
	baseURL   string
	apiKey    string
	workspace string
	timeout   time.Duration
}

// WithBaseURL sets the Opik API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the Opik API key.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithWorkspace sets the Opik workspace name.
func WithWorkspace(name string) Option {
	return func(c *config) { c.workspace = name }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Opik trace exporter.
func New(opts ...Option) (*Exporter, error) {
	cfg := &config{
		baseURL:   "https://www.comet.com/opik/api",
		workspace: "default",
		timeout:   10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiKey == "" {
		return nil, fmt.Errorf("opik: API key is required")
	}

	client := httpclient.New(
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithBearerToken(cfg.apiKey),
		httpclient.WithHeader("Comet-Workspace", cfg.workspace),
		httpclient.WithTimeout(cfg.timeout),
	)

	return &Exporter{
		client:    client,
		workspace: cfg.workspace,
	}, nil
}

// traceCreate is the Opik trace creation request.
type traceCreate struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	StartTime time.Time      `json:"start_time"`
	EndTime   time.Time      `json:"end_time"`
	Input     map[string]any `json:"input,omitempty"`
	Output    map[string]any `json:"output,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Error     string         `json:"error_info,omitempty"`
}

// traceResponse is the Opik trace creation response.
type traceResponse struct {
	ID string `json:"id"`
}

// ExportLLMCall sends an LLM call record to Opik via the traces API.
func (e *Exporter) ExportLLMCall(ctx context.Context, data o11y.LLMCallData) error {
	now := time.Now().UTC()
	traceID := uuid.New().String()

	name := data.Model
	if data.Provider != "" {
		name = data.Provider + "/" + data.Model
	}

	metadata := map[string]any{
		"model":         data.Model,
		"provider":      data.Provider,
		"input_tokens":  data.InputTokens,
		"output_tokens": data.OutputTokens,
		"total_tokens":  data.InputTokens + data.OutputTokens,
		"cost":          data.Cost,
	}
	for k, v := range data.Metadata {
		metadata[k] = v
	}

	trace := traceCreate{
		ID:        traceID,
		Name:      name,
		StartTime: now.Add(-data.Duration),
		EndTime:   now,
		Input:     map[string]any{"messages": data.Messages},
		Output:    map[string]any{"response": data.Response},
		Metadata:  metadata,
		Error:     data.Error,
	}

	_, err := httpclient.DoJSON[traceResponse](ctx, e.client, "POST", "/v1/private/traces", trace)
	if err != nil {
		return fmt.Errorf("opik: export: %w", err)
	}

	return nil
}

// Flush is a no-op for the HTTP-based exporter.
func (e *Exporter) Flush(ctx context.Context) error {
	return nil
}

// compile-time interface check
var _ o11y.TraceExporter = (*Exporter)(nil)
