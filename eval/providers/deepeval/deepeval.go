// Package deepeval provides a DeepEval evaluation metric for the Beluga AI
// eval framework. It implements the eval.Metric interface and sends evaluation
// requests to a DeepEval API endpoint.
//
// DeepEval provides LLM evaluation metrics including faithfulness, answer
// relevancy, contextual precision, hallucination, and bias.
//
// Usage:
//
//	metric, err := deepeval.New(
//	    deepeval.WithBaseURL("http://localhost:8080"),
//	    deepeval.WithMetricName("faithfulness"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	score, err := metric.Score(ctx, sample)
package deepeval

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
)

// Metric implements eval.Metric for DeepEval.
type Metric struct {
	client     *httpclient.Client
	metricName string
}

// Option configures a Metric.
type Option func(*config)

type config struct {
	baseURL    string
	apiKey     string
	metricName string
	timeout    time.Duration
}

// WithBaseURL sets the DeepEval API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the DeepEval API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithMetricName sets the DeepEval metric to evaluate.
func WithMetricName(name string) Option {
	return func(c *config) { c.metricName = name }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new DeepEval metric evaluator.
func New(opts ...Option) (*Metric, error) {
	cfg := &config{
		baseURL:    "http://localhost:8080",
		metricName: "faithfulness",
		timeout:    30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.metricName == "" {
		return nil, fmt.Errorf("deepeval: metric name is required")
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithTimeout(cfg.timeout),
	}
	if cfg.apiKey != "" {
		clientOpts = append(clientOpts, httpclient.WithBearerToken(cfg.apiKey))
	}

	return &Metric{
		client:     httpclient.New(clientOpts...),
		metricName: cfg.metricName,
	}, nil
}

// evaluateRequest is the DeepEval evaluate API request.
type evaluateRequest struct {
	Metric         string   `json:"metric"`
	Input          string   `json:"input"`
	ActualOutput   string   `json:"actual_output"`
	ExpectedOutput string   `json:"expected_output,omitempty"`
	Context        []string `json:"context,omitempty"`
}

// evaluateResponse is the DeepEval evaluate API response.
type evaluateResponse struct {
	Score   float64 `json:"score"`
	Reason  string  `json:"reason,omitempty"`
	Success bool    `json:"success"`
}

// Name returns the metric name prefixed with "deepeval_".
func (m *Metric) Name() string {
	return "deepeval_" + m.metricName
}

// Score evaluates a single sample using the DeepEval API and returns a score
// in [0.0, 1.0].
func (m *Metric) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	var contexts []string
	for _, doc := range sample.RetrievedDocs {
		contexts = append(contexts, doc.Content)
	}

	req := evaluateRequest{
		Metric:         m.metricName,
		Input:          sample.Input,
		ActualOutput:   sample.Output,
		ExpectedOutput: sample.ExpectedOutput,
		Context:        contexts,
	}

	resp, err := httpclient.DoJSON[evaluateResponse](ctx, m.client, "POST", "/api/v1/evaluate", req)
	if err != nil {
		return 0, fmt.Errorf("deepeval: evaluate: %w", err)
	}

	if !resp.Success {
		reason := resp.Reason
		if reason == "" {
			reason = "evaluation failed"
		}
		return 0, fmt.Errorf("deepeval: %s", reason)
	}

	score := resp.Score
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score, nil
}

// compile-time interface check
var _ eval.Metric = (*Metric)(nil)
