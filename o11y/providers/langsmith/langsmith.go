// Package langsmith provides a LangSmith trace exporter for the Beluga AI
// observability system. It implements the o11y.TraceExporter interface and
// sends LLM call data to LangSmith via its HTTP runs API.
//
// Usage:
//
//	exporter, err := langsmith.New(
//	    langsmith.WithAPIKey("lsv2_..."),
//	    langsmith.WithProject("my-project"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = exporter.ExportLLMCall(ctx, data)
package langsmith

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/o11y"
)

// Exporter implements o11y.TraceExporter for LangSmith.
type Exporter struct {
	client  *httpclient.Client
	project string
}

// Option configures an Exporter.
type Option func(*config)

type config struct {
	baseURL string
	apiKey  string
	project string
	timeout time.Duration
}

// WithBaseURL sets the LangSmith API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the LangSmith API key.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithProject sets the LangSmith project name.
func WithProject(name string) Option {
	return func(c *config) { c.project = name }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new LangSmith trace exporter.
func New(opts ...Option) (*Exporter, error) {
	cfg := &config{
		baseURL: "https://api.smith.langchain.com",
		project: "default",
		timeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiKey == "" {
		return nil, fmt.Errorf("langsmith: API key is required")
	}

	client := httpclient.New(
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithHeader("x-api-key", cfg.apiKey),
		httpclient.WithTimeout(cfg.timeout),
	)

	return &Exporter{
		client:  client,
		project: cfg.project,
	}, nil
}

// runCreate is the LangSmith run creation request.
type runCreate struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	RunType     string         `json:"run_type"`
	StartTime   time.Time      `json:"start_time"`
	EndTime     time.Time      `json:"end_time"`
	Inputs      map[string]any `json:"inputs,omitempty"`
	Outputs     map[string]any `json:"outputs,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
	Error       string         `json:"error,omitempty"`
	SessionName string         `json:"session_name,omitempty"`
}

// batchRequest is the LangSmith batch runs API request.
type batchRequest struct {
	Post []runCreate `json:"post"`
}

// batchResponse is the LangSmith batch runs API response.
type batchResponse struct {
	// LangSmith returns empty response on success.
}

// ExportLLMCall sends an LLM call record to LangSmith via the runs API.
func (e *Exporter) ExportLLMCall(ctx context.Context, data o11y.LLMCallData) error {
	now := time.Now().UTC()
	runID := uuid.New().String()

	name := data.Model
	if data.Provider != "" {
		name = data.Provider + "/" + data.Model
	}

	inputs := map[string]any{
		"messages": data.Messages,
	}

	outputs := map[string]any{
		"response": data.Response,
	}

	extra := map[string]any{
		"model":         data.Model,
		"provider":      data.Provider,
		"input_tokens":  data.InputTokens,
		"output_tokens": data.OutputTokens,
		"total_tokens":  data.InputTokens + data.OutputTokens,
		"cost":          data.Cost,
	}
	for k, v := range data.Metadata {
		extra[k] = v
	}

	run := runCreate{
		ID:          runID,
		Name:        name,
		RunType:     "llm",
		StartTime:   now.Add(-data.Duration),
		EndTime:     now,
		Inputs:      inputs,
		Outputs:     outputs,
		Extra:       extra,
		Error:       data.Error,
		SessionName: e.project,
	}

	req := batchRequest{
		Post: []runCreate{run},
	}

	_, err := httpclient.DoJSON[batchResponse](ctx, e.client, "POST", "/runs/batch", req)
	if err != nil {
		return fmt.Errorf("langsmith: export: %w", err)
	}

	return nil
}

// Flush is a no-op for the HTTP-based exporter. Each ExportLLMCall sends
// data synchronously.
func (e *Exporter) Flush(ctx context.Context) error {
	return nil
}

// compile-time interface check
var _ o11y.TraceExporter = (*Exporter)(nil)
