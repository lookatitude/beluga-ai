package lakera

import (
	"context"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/guard"
	"github.com/lookatitude/beluga-ai/v2/internal/httpclient"
)

// Guard implements guard.Guard for Lakera Guard.
type Guard struct {
	client    *httpclient.Client
	projectID string
}

// Option configures a Guard.
type Option func(*config)

type config struct {
	baseURL   string
	apiKey    string
	projectID string
	timeout   time.Duration
}

// WithBaseURL sets the Lakera Guard API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the Lakera Guard API key.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithProjectID selects the Lakera project (and therefore its policy). When
// unset, Lakera applies its default policy.
func WithProjectID(id string) Option {
	return func(c *config) { c.projectID = id }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new Lakera Guard guard.
func New(opts ...Option) (*Guard, error) {
	cfg := &config{
		baseURL: "https://api.lakera.ai",
		timeout: 15 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiKey == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "lakera: API key is required")
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithTimeout(cfg.timeout),
		httpclient.WithBearerToken(cfg.apiKey),
	}

	return &Guard{
		client:    httpclient.New(clientOpts...),
		projectID: cfg.projectID,
	}, nil
}

// guardMessage is an OpenAI-style chat message in a Lakera v2 request.
type guardMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// guardRequest is the Lakera Guard POST /v2/guard request.
type guardRequest struct {
	Messages  []guardMessage `json:"messages"`
	ProjectID string         `json:"project_id,omitempty"`
	Breakdown bool           `json:"breakdown"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// guardResponse is the Lakera Guard POST /v2/guard response. Flagged is a
// pointer so a response without the decisive field is detected and treated
// as an error (fail closed) instead of decoding to false.
type guardResponse struct {
	Flagged   *bool             `json:"flagged"`
	Breakdown []breakdownResult `json:"breakdown"`
}

// breakdownResult is one detector entry of a v2 breakdown.
type breakdownResult struct {
	DetectorType string `json:"detector_type"`
	DetectorID   string `json:"detector_id"`
	Detected     bool   `json:"detected"`
}

// Name returns the guard name.
func (g *Guard) Name() string {
	return "lakera_guard"
}

// Validate sends the content to Lakera Guard (POST /v2/guard) for screening.
// Input content is sent as a "user" message and output content as an
// "assistant" message. A response without a boolean "flagged" field is an
// error, never an allow.
func (g *Guard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	role := "user"
	if input.Role == "output" {
		role = "assistant"
	}
	req := guardRequest{
		Messages:  []guardMessage{{Role: role, Content: input.Content}},
		ProjectID: g.projectID,
		Breakdown: true,
		Metadata:  input.Metadata,
	}

	resp, err := httpclient.DoJSON[guardResponse](ctx, g.client, "POST", "/v2/guard", req)
	if err != nil {
		return guard.GuardResult{}, core.Errorf(core.ErrProviderDown, "lakera: validate: %w", err)
	}
	if resp.Flagged == nil {
		return guard.GuardResult{}, core.Errorf(core.ErrProviderDown,
			"lakera: validate: response has no \"flagged\" field; failing closed")
	}

	result := guard.GuardResult{
		Allowed:   !*resp.Flagged,
		GuardName: g.Name(),
	}

	if *resp.Flagged {
		var flagged []string
		for _, b := range resp.Breakdown {
			if b.Detected {
				name := b.DetectorType
				if name == "" {
					name = b.DetectorID
				}
				flagged = append(flagged, name)
			}
		}
		if len(flagged) > 0 {
			result.Reason = "flagged detectors: " + strings.Join(flagged, ", ")
		} else {
			result.Reason = "blocked by Lakera Guard"
		}
	}

	return result, nil
}

// compile-time interface check
var _ guard.Guard = (*Guard)(nil)
