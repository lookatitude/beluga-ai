package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/eval"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// defaultModelID identifies the mock's "model" in OTel attributes and logs.
const defaultModelID = "mock-fixture"

// envFixturesPath is the environment variable consulted when no fixture
// source is provided via config or programmatic options.
const envFixturesPath = "BELUGA_MOCK_FIXTURES"

// Fixture is one canned response in the mock's queue. When Error is
// non-empty the mock returns it as an error for that call; otherwise it
// builds an AIMessage from Content and ToolCalls.
type Fixture struct {
	// Content is the text content of the AIMessage.
	Content string `json:"content,omitempty"`
	// ToolCalls are tool invocations the mock should emit. When empty and
	// Content is non-empty the response is a final answer.
	ToolCalls []schema.ToolCall `json:"tool_calls,omitempty"`
	// Error, if non-empty, causes the mock to return core.ErrInternal with
	// this message instead of an AIMessage.
	Error string `json:"error,omitempty"`
}

// Option configures a ChatModel at construction time.
type Option func(*options)

type options struct {
	fixtures  []Fixture
	modelID   string
	fallback  Fixture
	setBacked bool
}

// WithFixtures sets the initial fixture queue, taking precedence over any
// file-based source in ProviderConfig.
func WithFixtures(fs []Fixture) Option {
	return func(o *options) {
		o.fixtures = append([]Fixture(nil), fs...)
		o.setBacked = true
	}
}

// WithModelID overrides the model identifier reported by ModelID.
func WithModelID(id string) Option {
	return func(o *options) { o.modelID = id }
}

// WithFallback overrides the response returned when the fixture queue is
// exhausted. The fallback must satisfy the "final answer" contract — it
// should carry text Content and no ToolCalls so planners interpret it as
// a finish signal.
func WithFallback(f Fixture) Option {
	return func(o *options) { o.fallback = f }
}

// state holds the mutable portion of a mock ChatModel. It is shared
// across all ChatModel values produced by BindTools so that a bound model
// draws from the same fixture queue as its parent.
type state struct {
	mu       sync.Mutex
	fixtures []Fixture
	fallback Fixture
	cursor   int
	calls    int
}

// ChatModel is a fixture-driven implementation of llm.ChatModel. It is
// safe for concurrent use; callers may invoke Generate, Stream, and
// BindTools from multiple goroutines.
type ChatModel struct {
	st         *state
	modelID    string
	boundTools []schema.ToolDefinition
}

var _ llm.ChatModel = (*ChatModel)(nil)

// New constructs a ChatModel from a ProviderConfig plus programmatic
// options. The fixture source priority is: programmatic WithFixtures >
// cfg.Options["fixtures"] / ["fixtures_file"] > BELUGA_MOCK_FIXTURES env
// var. If no source supplies any fixtures the mock still works — every
// call returns the fallback response.
func New(cfg config.ProviderConfig, opts ...Option) (*ChatModel, error) {
	o := options{
		modelID:  defaultModelID,
		fallback: Fixture{Content: "mock: fixture queue exhausted"},
	}
	for _, apply := range opts {
		apply(&o)
	}
	if cfg.Model != "" {
		o.modelID = cfg.Model
	}

	fixtures := o.fixtures
	if !o.setBacked {
		loaded, err := loadFixturesFromConfig(&cfg)
		if err != nil {
			return nil, err
		}
		fixtures = loaded
	}

	return &ChatModel{
		st: &state{
			fixtures: fixtures,
			fallback: o.fallback,
		},
		modelID: o.modelID,
	}, nil
}

// loadFixturesFromConfig resolves the fixture source from ProviderConfig
// options and the environment. Missing sources return an empty slice,
// not an error — this lets the mock stand up with no configuration at all.
func loadFixturesFromConfig(cfg *config.ProviderConfig) ([]Fixture, error) {
	if raw, ok := cfg.Options["fixtures"]; ok {
		fs, err := coerceFixtures(raw)
		if err != nil {
			return nil, core.Errorf(core.ErrInvalidInput, "mock: invalid fixtures option: %v", err)
		}
		return fs, nil
	}
	if path, ok := config.GetOption[string](*cfg, "fixtures_file"); ok && path != "" {
		return loadFixturesFile(path)
	}
	if path := os.Getenv(envFixturesPath); path != "" {
		return loadFixturesFile(path)
	}
	return nil, nil
}

// coerceFixtures accepts either a []Fixture directly or a JSON-ish
// []any / []map[string]any (as produced by decoding cfg.Options from
// JSON) and normalises to []Fixture.
func coerceFixtures(raw any) ([]Fixture, error) {
	if fs, ok := raw.([]Fixture); ok {
		return append([]Fixture(nil), fs...), nil
	}
	blob, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var out []Fixture
	if err := json.Unmarshal(blob, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// loadFixturesFile reads and decodes a JSON fixtures file. The path is
// passed through filepath.Clean; it is the caller's responsibility to
// scope the file to a trusted location.
func loadFixturesFile(path string) ([]Fixture, error) {
	cleaned := filepath.Clean(path)
	// #nosec G304 -- path is supplied by the developer via config or env;
	// the mock provider is test-only and never sees untrusted input.
	data, err := os.ReadFile(cleaned)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "mock: read fixtures file %q: %w", cleaned, err)
	}
	var out []Fixture
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "mock: parse fixtures file %q: %w", cleaned, err)
	}
	return out, nil
}

// next returns the next fixture in the queue plus a call index, or the
// fallback when the queue is exhausted.
func (s *state) next() (Fixture, int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.calls
	s.calls++
	if s.cursor >= len(s.fixtures) {
		return s.fallback, idx, false
	}
	fx := s.fixtures[s.cursor]
	s.cursor++
	return fx, idx, true
}

// Generate returns the next fixture as an AIMessage, or the fallback when
// the queue is exhausted. It never blocks and never touches the network.
func (m *ChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fx, idx, _ := m.st.next()
	if fx.Error != "" {
		return nil, core.Errorf(core.ErrInvalidInput, "mock: fixture %d error: %s", idx, fx.Error)
	}
	return m.buildMessage(fx, idx), nil
}

// Stream yields the next fixture as a single chunk plus a terminal chunk
// carrying the finish reason. Callers consuming an entire stream receive
// exactly one Delta or ToolCalls payload.
func (m *ChatModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {
		if err := ctx.Err(); err != nil {
			yield(schema.StreamChunk{}, err)
			return
		}
		fx, idx, _ := m.st.next()
		if fx.Error != "" {
			yield(schema.StreamChunk{}, core.Errorf(core.ErrInvalidInput, "mock: fixture %d error: %s", idx, fx.Error))
			return
		}
		calls := assignToolCallIDs(fx.ToolCalls, idx)
		finish := "stop"
		if len(calls) > 0 {
			finish = "tool_calls"
		}
		if !yield(schema.StreamChunk{Delta: fx.Content, ToolCalls: calls, FinishReason: finish}, nil) {
			return
		}
	}
}

// BindTools returns a new ChatModel that shares the same fixture queue
// but records the supplied tool definitions. The parent model is unchanged.
func (m *ChatModel) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	return &ChatModel{
		st:         m.st,
		modelID:    m.modelID,
		boundTools: append([]schema.ToolDefinition(nil), tools...),
	}
}

// ModelID returns the configured model identifier.
func (m *ChatModel) ModelID() string { return m.modelID }

// BoundTools returns the tools recorded by BindTools on this ChatModel.
// It is exported for test assertions; non-test callers should not rely
// on it.
func (m *ChatModel) BoundTools() []schema.ToolDefinition {
	return append([]schema.ToolDefinition(nil), m.boundTools...)
}

// Calls returns the number of Generate and Stream invocations observed
// by this ChatModel's shared state. It is exported for tests.
func (m *ChatModel) Calls() int {
	m.st.mu.Lock()
	defer m.st.mu.Unlock()
	return m.st.calls
}

func (m *ChatModel) buildMessage(fx Fixture, idx int) *schema.AIMessage {
	msg := &schema.AIMessage{ModelID: m.modelID}
	if fx.Content != "" {
		msg.Parts = []schema.ContentPart{schema.TextPart{Text: fx.Content}}
	}
	msg.ToolCalls = assignToolCallIDs(fx.ToolCalls, idx)
	return msg
}

// FixturesFromTurns derives a mock-provider fixture queue from an
// eval.Turn trajectory. Only assistant turns are converted; user, tool,
// and system turns are skipped because the mock replays the LLM side of
// a conversation, not the user or the tool outputs. For each assistant
// turn the helper emits one Fixture carrying the turn's Content and a
// defensive copy of its ToolCalls, preserving trajectory order so
// replaying the fixture queue reproduces the original assistant outputs.
//
// Use with WithFixtures to seed a deterministic mock from a recorded
// trajectory in scaffolded eval branches, e.g.:
//
//	mock.New(cfg, mock.WithFixtures(mock.FixturesFromTurns(sample.Turns)))
func FixturesFromTurns(turns []eval.Turn) []Fixture {
	if len(turns) == 0 {
		return nil
	}
	out := make([]Fixture, 0, len(turns))
	for _, t := range turns {
		if t.Role != "assistant" {
			continue
		}
		fx := Fixture{Content: t.Content}
		if len(t.ToolCalls) > 0 {
			fx.ToolCalls = append([]schema.ToolCall(nil), t.ToolCalls...)
		}
		out = append(out, fx)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// assignToolCallIDs returns a defensive copy of calls with missing IDs
// filled in deterministically as "call_<fixtureIndex>_<i>".
func assignToolCallIDs(calls []schema.ToolCall, idx int) []schema.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]schema.ToolCall, len(calls))
	for i, c := range calls {
		if c.ID == "" {
			c.ID = fmt.Sprintf("call_%d_%d", idx, i)
		}
		out[i] = c
	}
	return out
}
