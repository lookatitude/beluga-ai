package azuresafety

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/guard"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
)

// Guard implements guard.Guard for Azure Content Safety.
type Guard struct {
	client    *httpclient.Client
	threshold int
}

// Option configures a Guard.
type Option func(*config)

type config struct {
	endpoint  string
	apiKey    string
	threshold int
	timeout   time.Duration
}

// WithEndpoint sets the Azure Content Safety endpoint URL.
func WithEndpoint(url string) Option {
	return func(c *config) { c.endpoint = url }
}

// WithAPIKey sets the Azure Content Safety API key.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithThreshold sets the severity threshold (0-6). Content with any category
// severity at or above this value is blocked.
func WithThreshold(t int) Option {
	return func(c *config) { c.threshold = t }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Azure Content Safety guard.
func New(opts ...Option) (*Guard, error) {
	cfg := &config{
		threshold: 2,
		timeout:   15 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.endpoint == "" {
		return nil, fmt.Errorf("azuresafety: endpoint is required")
	}
	if cfg.apiKey == "" {
		return nil, fmt.Errorf("azuresafety: API key is required")
	}

	client := httpclient.New(
		httpclient.WithBaseURL(cfg.endpoint),
		httpclient.WithHeader("Ocp-Apim-Subscription-Key", cfg.apiKey),
		httpclient.WithTimeout(cfg.timeout),
	)

	return &Guard{
		client:    client,
		threshold: cfg.threshold,
	}, nil
}

// analyzeRequest is the Azure Content Safety text analyze request.
type analyzeRequest struct {
	Text       string   `json:"text"`
	Categories []string `json:"categories,omitempty"`
}

// analyzeResponse is the Azure Content Safety text analyze response.
type analyzeResponse struct {
	CategoriesAnalysis []categoryAnalysis `json:"categoriesAnalysis"`
}

// categoryAnalysis holds a single category's analysis result.
type categoryAnalysis struct {
	Category string `json:"category"`
	Severity int    `json:"severity"`
}

// Name returns the guard name.
func (g *Guard) Name() string {
	return "azure_content_safety"
}

// Validate sends the content to Azure Content Safety for analysis.
func (g *Guard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	req := analyzeRequest{
		Text: input.Content,
	}

	resp, err := httpclient.DoJSON[analyzeResponse](ctx, g.client, "POST",
		"/contentsafety/text:analyze?api-version=2024-09-01", req)
	if err != nil {
		return guard.GuardResult{}, fmt.Errorf("azuresafety: validate: %w", err)
	}

	result := guard.GuardResult{
		Allowed:   true,
		GuardName: g.Name(),
	}

	var flagged []string
	for _, cat := range resp.CategoriesAnalysis {
		if cat.Severity >= g.threshold {
			flagged = append(flagged, fmt.Sprintf("%s(severity=%d)", cat.Category, cat.Severity))
			result.Allowed = false
		}
	}

	if !result.Allowed {
		result.Reason = "flagged: " + strings.Join(flagged, ", ")
	}

	return result, nil
}

// compile-time interface check
var _ guard.Guard = (*Guard)(nil)
