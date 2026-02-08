// Package braintrust provides a Braintrust evaluation metric for the Beluga AI
// eval framework. It implements the eval.Metric interface and sends evaluation
// requests to the Braintrust API.
//
// Braintrust provides evaluation scoring for LLM outputs including
// factuality, relevance, and custom scoring functions.
//
// Usage:
//
//	metric, err := braintrust.New(
//	    braintrust.WithAPIKey("bt-..."),
//	    braintrust.WithProjectName("my-project"),
//	    braintrust.WithMetricName("factuality"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	score, err := metric.Score(ctx, sample)
package braintrust

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
)

// Metric implements eval.Metric for Braintrust.
type Metric struct {
	client      *httpclient.Client
	metricName  string
	projectName string
}

// Option configures a Metric.
type Option func(*config)

type config struct {
	baseURL     string
	apiKey      string
	metricName  string
	projectName string
	timeout     time.Duration
}

// WithBaseURL sets the Braintrust API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the Braintrust API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithMetricName sets the Braintrust metric to evaluate.
func WithMetricName(name string) Option {
	return func(c *config) { c.metricName = name }
}

// WithProjectName sets the Braintrust project name.
func WithProjectName(name string) Option {
	return func(c *config) { c.projectName = name }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Braintrust metric evaluator.
func New(opts ...Option) (*Metric, error) {
	cfg := &config{
		baseURL:     "https://api.braintrust.dev",
		metricName:  "factuality",
		projectName: "default",
		timeout:     30 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.metricName == "" {
		return nil, fmt.Errorf("braintrust: metric name is required")
	}
	if cfg.apiKey == "" {
		return nil, fmt.Errorf("braintrust: API key is required")
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithTimeout(cfg.timeout),
		httpclient.WithBearerToken(cfg.apiKey),
	}

	return &Metric{
		client:      httpclient.New(clientOpts...),
		metricName:  cfg.metricName,
		projectName: cfg.projectName,
	}, nil
}

// scoreRequest is the Braintrust score API request.
type scoreRequest struct {
	ProjectName string    `json:"project_name"`
	Scores      []scoring `json:"scores"`
}

// scoring represents a single score request.
type scoring struct {
	Name           string   `json:"name"`
	Input          string   `json:"input"`
	Output         string   `json:"output"`
	Expected       string   `json:"expected,omitempty"`
	Context        []string `json:"context,omitempty"`
}

// scoreResponse is the Braintrust score API response.
type scoreResponse struct {
	Results []scoreResult `json:"results"`
}

// scoreResult holds a single metric score from Braintrust.
type scoreResult struct {
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

// Name returns the metric name prefixed with "braintrust_".
func (m *Metric) Name() string {
	return "braintrust_" + m.metricName
}

// Score evaluates a single sample using the Braintrust API and returns a score
// in [0.0, 1.0].
func (m *Metric) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	var contexts []string
	for _, doc := range sample.RetrievedDocs {
		contexts = append(contexts, doc.Content)
	}

	req := scoreRequest{
		ProjectName: m.projectName,
		Scores: []scoring{
			{
				Name:     m.metricName,
				Input:    sample.Input,
				Output:   sample.Output,
				Expected: sample.ExpectedOutput,
				Context:  contexts,
			},
		},
	}

	resp, err := httpclient.DoJSON[scoreResponse](ctx, m.client, "POST", "/v1/score", req)
	if err != nil {
		return 0, fmt.Errorf("braintrust: score: %w", err)
	}

	if len(resp.Results) == 0 {
		return 0, fmt.Errorf("braintrust: no results returned")
	}

	score := resp.Results[0].Score
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
