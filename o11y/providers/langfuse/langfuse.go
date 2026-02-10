package langfuse

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/o11y"
)

// Exporter implements o11y.TraceExporter for Langfuse.
type Exporter struct {
	client    *httpclient.Client
	publicKey string
	secretKey string
}

// Option configures an Exporter.
type Option func(*config)

type config struct {
	baseURL   string
	publicKey string
	secretKey string
	timeout   time.Duration
}

// WithBaseURL sets the Langfuse API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithPublicKey sets the Langfuse public key.
func WithPublicKey(key string) Option {
	return func(c *config) { c.publicKey = key }
}

// WithSecretKey sets the Langfuse secret key.
func WithSecretKey(key string) Option {
	return func(c *config) { c.secretKey = key }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Langfuse trace exporter.
func New(opts ...Option) (*Exporter, error) {
	cfg := &config{
		baseURL: "https://cloud.langfuse.com",
		timeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.publicKey == "" {
		return nil, fmt.Errorf("langfuse: public key is required")
	}
	if cfg.secretKey == "" {
		return nil, fmt.Errorf("langfuse: secret key is required")
	}

	auth := base64.StdEncoding.EncodeToString(
		[]byte(cfg.publicKey + ":" + cfg.secretKey),
	)

	client := httpclient.New(
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithHeader("Authorization", "Basic "+auth),
		httpclient.WithTimeout(cfg.timeout),
	)

	return &Exporter{
		client:    client,
		publicKey: cfg.publicKey,
		secretKey: cfg.secretKey,
	}, nil
}

// ingestionEvent is the Langfuse ingestion API event format.
type ingestionEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Body      any       `json:"body"`
}

// ingestionRequest is the Langfuse batch ingestion API request.
type ingestionRequest struct {
	Batch []ingestionEvent `json:"batch"`
}

// ingestionResponse is the Langfuse batch ingestion API response.
type ingestionResponse struct {
	Successes []struct {
		ID     string `json:"id"`
		Status int    `json:"status"`
	} `json:"successes"`
	Errors []struct {
		ID      string `json:"id"`
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"errors"`
}

// traceBody is the Langfuse trace creation event body.
type traceBody struct {
	ID       string         `json:"id"`
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// generationBody is the Langfuse generation event body.
type generationBody struct {
	ID             string         `json:"id"`
	TraceID        string         `json:"traceId"`
	Name           string         `json:"name,omitempty"`
	Model          string         `json:"model,omitempty"`
	Input          any            `json:"input,omitempty"`
	Output         any            `json:"output,omitempty"`
	StartTime      time.Time      `json:"startTime"`
	EndTime        time.Time      `json:"endTime"`
	Usage          usageBody      `json:"usage,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Level          string         `json:"level,omitempty"`
	StatusMessage  string         `json:"statusMessage,omitempty"`
	ModelParameter map[string]any `json:"modelParameters,omitempty"`
}

// usageBody is the Langfuse token usage format.
type usageBody struct {
	Input  int `json:"input,omitempty"`
	Output int `json:"output,omitempty"`
	Total  int `json:"total,omitempty"`
}

// ExportLLMCall sends an LLM call record to Langfuse via the ingestion API.
func (e *Exporter) ExportLLMCall(ctx context.Context, data o11y.LLMCallData) error {
	now := time.Now().UTC()
	traceID := uuid.New().String()
	genID := uuid.New().String()

	name := data.Model
	if data.Provider != "" {
		name = data.Provider + "/" + data.Model
	}

	level := "DEFAULT"
	var statusMsg string
	if data.Error != "" {
		level = "ERROR"
		statusMsg = data.Error
	}

	batch := ingestionRequest{
		Batch: []ingestionEvent{
			{
				ID:        traceID,
				Type:      "trace-create",
				Timestamp: now,
				Body: traceBody{
					ID:       traceID,
					Name:     name,
					Metadata: data.Metadata,
				},
			},
			{
				ID:        genID,
				Type:      "generation-create",
				Timestamp: now,
				Body: generationBody{
					ID:      genID,
					TraceID: traceID,
					Name:    name,
					Model:   data.Model,
					Input:   data.Messages,
					Output:  data.Response,
					StartTime: now.Add(-data.Duration),
					EndTime:   now,
					Usage: usageBody{
						Input:  data.InputTokens,
						Output: data.OutputTokens,
						Total:  data.InputTokens + data.OutputTokens,
					},
					Metadata:      data.Metadata,
					Level:         level,
					StatusMessage: statusMsg,
				},
			},
		},
	}

	resp, err := httpclient.DoJSON[ingestionResponse](ctx, e.client, "POST", "/api/public/ingestion", batch)
	if err != nil {
		return fmt.Errorf("langfuse: ingest: %w", err)
	}

	if len(resp.Errors) > 0 {
		return fmt.Errorf("langfuse: ingestion errors: %s (status %d)", resp.Errors[0].Message, resp.Errors[0].Status)
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
