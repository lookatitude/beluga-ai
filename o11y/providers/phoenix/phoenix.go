// Package phoenix provides an Arize Phoenix trace exporter for the Beluga AI
// observability system. It implements the o11y.TraceExporter interface and
// sends LLM call data to an Arize Phoenix instance via its HTTP API.
//
// Phoenix uses OTel-compatible spans, so this exporter translates LLM call
// data into the Phoenix /v1/traces JSON format.
//
// Usage:
//
//	exporter, err := phoenix.New(
//	    phoenix.WithBaseURL("http://localhost:6006"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = exporter.ExportLLMCall(ctx, data)
package phoenix

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/o11y"
)

// Exporter implements o11y.TraceExporter for Arize Phoenix.
type Exporter struct {
	client *httpclient.Client
}

// Option configures an Exporter.
type Option func(*config)

type config struct {
	baseURL string
	apiKey  string
	timeout time.Duration
}

// WithBaseURL sets the Phoenix API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the Phoenix API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Phoenix trace exporter.
func New(opts ...Option) (*Exporter, error) {
	cfg := &config{
		baseURL: "http://localhost:6006",
		timeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithTimeout(cfg.timeout),
		httpclient.WithHeader("Content-Type", "application/json"),
	}
	if cfg.apiKey != "" {
		clientOpts = append(clientOpts, httpclient.WithBearerToken(cfg.apiKey))
	}

	return &Exporter{
		client: httpclient.New(clientOpts...),
	}, nil
}

// phoenixSpan represents a span in the Phoenix /v1/traces format.
type phoenixSpan struct {
	Name       string              `json:"name"`
	Context    phoenixSpanContext  `json:"context"`
	Kind       string              `json:"kind"`
	StartTime  time.Time           `json:"start_time"`
	EndTime    time.Time           `json:"end_time"`
	Status     phoenixStatus       `json:"status"`
	Attributes map[string]any      `json:"attributes,omitempty"`
	Events     []phoenixSpanEvent  `json:"events,omitempty"`
}

// phoenixSpanContext carries trace and span identifiers.
type phoenixSpanContext struct {
	TraceID string `json:"trace_id"`
	SpanID  string `json:"span_id"`
}

// phoenixStatus represents the span outcome.
type phoenixStatus struct {
	StatusCode string `json:"status_code"`
	Message    string `json:"message,omitempty"`
}

// phoenixSpanEvent represents an event within a span.
type phoenixSpanEvent struct {
	Name       string         `json:"name"`
	Timestamp  time.Time      `json:"timestamp"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

// phoenixTraceRequest is the Phoenix /v1/traces request body.
type phoenixTraceRequest struct {
	Data []phoenixSpan `json:"data"`
}

// phoenixTraceResponse is the Phoenix /v1/traces response body.
type phoenixTraceResponse struct {
	Data []struct {
		TraceID string `json:"trace_id"`
	} `json:"data"`
}

// ExportLLMCall sends an LLM call record to Phoenix via the /v1/traces API.
func (e *Exporter) ExportLLMCall(ctx context.Context, data o11y.LLMCallData) error {
	now := time.Now().UTC()
	traceID := randomHex(16)
	spanID := randomHex(8)

	name := "llm." + data.Model
	if data.Provider != "" {
		name = data.Provider + "." + data.Model
	}

	status := phoenixStatus{StatusCode: "OK"}
	if data.Error != "" {
		status = phoenixStatus{StatusCode: "ERROR", Message: data.Error}
	}

	attrs := map[string]any{
		"llm.model_name":   data.Model,
		"llm.provider":     data.Provider,
		"llm.token_count.prompt":     data.InputTokens,
		"llm.token_count.completion": data.OutputTokens,
		"llm.token_count.total":      data.InputTokens + data.OutputTokens,
	}

	if data.Cost > 0 {
		attrs["llm.cost"] = data.Cost
	}

	if data.Messages != nil {
		attrs["input.value"] = data.Messages
	}
	if data.Response != nil {
		attrs["output.value"] = data.Response
	}

	for k, v := range data.Metadata {
		attrs["metadata."+k] = v
	}

	span := phoenixSpan{
		Name: name,
		Context: phoenixSpanContext{
			TraceID: traceID,
			SpanID:  spanID,
		},
		Kind:       "LLM",
		StartTime:  now.Add(-data.Duration),
		EndTime:    now,
		Status:     status,
		Attributes: attrs,
	}

	req := phoenixTraceRequest{Data: []phoenixSpan{span}}

	_, err := httpclient.DoJSON[phoenixTraceResponse](ctx, e.client, "POST", "/v1/traces", req)
	if err != nil {
		return fmt.Errorf("phoenix: export trace: %w", err)
	}

	return nil
}

// Flush is a no-op for the HTTP-based exporter.
func (e *Exporter) Flush(ctx context.Context) error {
	return nil
}

// randomHex generates a random hex string of n bytes.
func randomHex(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.IntN(256))
	}
	return hex.EncodeToString(b)
}

// compile-time interface check
var _ o11y.TraceExporter = (*Exporter)(nil)
