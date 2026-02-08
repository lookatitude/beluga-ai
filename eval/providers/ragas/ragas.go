// Package ragas provides RAGAS (Retrieval Augmented Generation Assessment)
// evaluation metrics for the Beluga AI eval framework. It implements the
// eval.Metric interface and sends evaluation requests to a RAGAS API endpoint.
//
// RAGAS provides metrics for evaluating RAG pipelines including faithfulness,
// answer relevancy, context precision, and context recall.
//
// Usage:
//
//	metric, err := ragas.New(
//	    ragas.WithBaseURL("http://localhost:8080"),
//	    ragas.WithMetricName("faithfulness"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	score, err := metric.Score(ctx, sample)
package ragas

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
)

// Metric implements eval.Metric for RAGAS evaluation.
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

// WithBaseURL sets the RAGAS API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the RAGAS API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithMetricName sets the RAGAS metric to evaluate (e.g., "faithfulness",
// "answer_relevancy", "context_precision", "context_recall").
func WithMetricName(name string) Option {
	return func(c *config) { c.metricName = name }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new RAGAS metric evaluator.
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
		return nil, fmt.Errorf("ragas: metric name is required")
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

// evaluateRequest is the RAGAS evaluation API request.
type evaluateRequest struct {
	MetricName string        `json:"metric_name"`
	Data       []evaluateDatum `json:"data"`
}

// evaluateDatum is a single evaluation datum sent to RAGAS.
type evaluateDatum struct {
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Contexts []string `json:"contexts,omitempty"`
	Ground   string   `json:"ground_truth,omitempty"`
}

// evaluateResponse is the RAGAS evaluation API response.
type evaluateResponse struct {
	Scores []scoreResult `json:"scores"`
}

// scoreResult holds a single metric score from RAGAS.
type scoreResult struct {
	MetricName string  `json:"metric_name"`
	Score      float64 `json:"score"`
}

// Name returns the metric name prefixed with "ragas_".
func (m *Metric) Name() string {
	return "ragas_" + m.metricName
}

// Score evaluates a single sample using the RAGAS API and returns a score
// in [0.0, 1.0].
func (m *Metric) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	var contexts []string
	for _, doc := range sample.RetrievedDocs {
		contexts = append(contexts, doc.Content)
	}

	req := evaluateRequest{
		MetricName: m.metricName,
		Data: []evaluateDatum{
			{
				Question: sample.Input,
				Answer:   sample.Output,
				Contexts: contexts,
				Ground:   sample.ExpectedOutput,
			},
		},
	}

	resp, err := httpclient.DoJSON[evaluateResponse](ctx, m.client, "POST", "/api/v1/evaluate", req)
	if err != nil {
		return 0, fmt.Errorf("ragas: evaluate: %w", err)
	}

	if len(resp.Scores) == 0 {
		return 0, fmt.Errorf("ragas: no scores returned")
	}

	score := resp.Scores[0].Score
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
