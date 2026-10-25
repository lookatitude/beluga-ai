package nemo

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/guard"
	"github.com/lookatitude/beluga-ai/v2/internal/httpclient"
)

// Guard implements guard.Guard for NVIDIA NeMo Guardrails.
type Guard struct {
	client   *httpclient.Client
	configID string
	model    string
}

// Option configures a Guard.
type Option func(*config)

type config struct {
	baseURL  string
	apiKey   string
	configID string
	model    string
	timeout  time.Duration
}

// WithBaseURL sets the NeMo Guardrails API base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithAPIKey sets the NeMo Guardrails API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// WithConfigID sets the NeMo Guardrails configuration ID.
func WithConfigID(id string) Option {
	return func(c *config) { c.configID = id }
}

// WithModel sets the "model" field of the chat-completions request. The NeMo
// Guardrails server requires it; it should name the main model of the
// selected configuration. Defaults to "default".
func WithModel(model string) Option {
	return func(c *config) { c.model = model }
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// New creates a new NeMo Guardrails guard.
func New(opts ...Option) (*Guard, error) {
	cfg := &config{
		baseURL:  "http://localhost:8080",
		configID: "default",
		model:    "default",
		timeout:  15 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(cfg.baseURL),
		httpclient.WithTimeout(cfg.timeout),
	}
	if cfg.apiKey != "" {
		clientOpts = append(clientOpts, httpclient.WithBearerToken(cfg.apiKey))
	}

	return &Guard{
		client:   httpclient.New(clientOpts...),
		configID: cfg.configID,
		model:    cfg.model,
	}, nil
}

// chatRequest is the NeMo Guardrails server /v1/chat/completions request.
// The configuration is selected with guardrails.config_id, and the request
// asks for rails-only evaluation (no dialog / main LLM generation) plus the
// activated-rails log, which is how the guard learns whether a rail stopped
// the interaction.
type chatRequest struct {
	Model      string            `json:"model"`
	Messages   []chatMessage     `json:"messages"`
	Guardrails guardrailsRequest `json:"guardrails"`
}

// chatMessage is a NeMo chat message.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type guardrailsRequest struct {
	ConfigID string            `json:"config_id"`
	Options  generationOptions `json:"options"`
}

type generationOptions struct {
	Rails railsOptions `json:"rails"`
	Log   logOptions   `json:"log"`
}

type railsOptions struct {
	Input  bool `json:"input"`
	Output bool `json:"output"`
	Dialog bool `json:"dialog"`
}

type logOptions struct {
	ActivatedRails bool `json:"activated_rails"`
}

// chatResponse is the NeMo Guardrails server /v1/chat/completions response
// (OpenAI-style choices plus a guardrails data block).
type chatResponse struct {
	Choices    []chatChoice            `json:"choices"`
	Guardrails *guardrailsResponseData `json:"guardrails"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type guardrailsResponseData struct {
	Log *generationLog `json:"log"`
}

type generationLog struct {
	ActivatedRails *[]activatedRail `json:"activated_rails"`
}

// activatedRail mirrors NeMo's ActivatedRail log entry. Stop is true when the
// rail stopped the interaction, i.e. the content was blocked.
type activatedRail struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Stop bool   `json:"stop"`
}

// Name returns the guard name.
func (g *Guard) Name() string {
	return "nemo_guardrails"
}

// Validate sends the content to NeMo Guardrails for rails-only evaluation.
// Input content is checked with the input rails; content whose Role is
// "output" is sent as the assistant turn and checked with the output rails.
//
// The guard fails closed: a response without choices or without the
// activated-rails log (for example from a server that does not implement the
// documented contract) is an error, never an allow.
func (g *Guard) Validate(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
	isOutput := input.Role == "output"
	msgs := []chatMessage{{Role: "user", Content: input.Content}}
	if isOutput {
		msgs = []chatMessage{{Role: "user", Content: ""}, {Role: "assistant", Content: input.Content}}
	}
	req := chatRequest{
		Model:    g.model,
		Messages: msgs,
		Guardrails: guardrailsRequest{
			ConfigID: g.configID,
			Options: generationOptions{
				Rails: railsOptions{Input: !isOutput, Output: isOutput},
				Log:   logOptions{ActivatedRails: true},
			},
		},
	}

	resp, err := httpclient.DoJSON[chatResponse](ctx, g.client, "POST", "/v1/chat/completions", req)
	if err != nil {
		return guard.GuardResult{}, core.Errorf(core.ErrProviderDown, "nemo: validate: %w", err)
	}
	if len(resp.Choices) == 0 || resp.Guardrails == nil || resp.Guardrails.Log == nil || resp.Guardrails.Log.ActivatedRails == nil {
		return guard.GuardResult{}, core.Errorf(core.ErrProviderDown,
			"nemo: validate: response does not match the chat-completions contract (missing choices or guardrails.log.activated_rails); failing closed")
	}

	result := guard.GuardResult{Allowed: true, GuardName: g.Name()}
	for _, rail := range *resp.Guardrails.Log.ActivatedRails {
		if rail.Stop {
			result.Allowed = false
			result.Reason = "blocked by NeMo Guardrails"
			if rail.Name != "" {
				result.Reason += " rail " + rail.Name
			}
			return result, nil
		}
	}

	// Output rails may rewrite the bot message (for example masking PII).
	if isOutput {
		if content := resp.Choices[0].Message.Content; content != "" && content != input.Content {
			result.Modified = content
		}
	}
	return result, nil
}

// compile-time interface check
var _ guard.Guard = (*Guard)(nil)
