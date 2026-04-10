package declarative

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// AgentSpec is the declarative specification for an agent.
type AgentSpec struct {
	// ID is the unique agent identifier.
	ID string `json:"id"`
	// Persona defines the agent's role, goal, and backstory.
	Persona PersonaSpec `json:"persona"`
	// Model defines the LLM provider and model to use.
	Model ModelSpec `json:"model"`
	// Tools lists the tool names to attach to the agent.
	Tools []string `json:"tools,omitempty"`
	// MaxIterations limits reasoning iterations. Zero means default.
	MaxIterations int `json:"max_iterations,omitempty"`
	// SystemPrompt overrides the default system prompt.
	SystemPrompt string `json:"system_prompt,omitempty"`
	// Metadata holds arbitrary key-value pairs.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// PersonaSpec defines the agent's identity.
type PersonaSpec struct {
	Role      string `json:"role"`
	Goal      string `json:"goal,omitempty"`
	Backstory string `json:"backstory,omitempty"`
}

// ModelSpec defines which LLM to use.
type ModelSpec struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	// Temperature is a pointer so that an explicit zero value
	// (deterministic/greedy mode) is distinguishable from "unset".
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
}

// Parser reads and validates an AgentSpec from various sources.
type Parser interface {
	// Parse reads an AgentSpec from raw bytes.
	Parse(ctx context.Context, data []byte) (*AgentSpec, error)
}

// Builder constructs runtime objects from an AgentSpec.
type Builder interface {
	// Build creates an agent configuration from a spec.
	// The returned AgentBuild contains all resolved components.
	Build(ctx context.Context, spec *AgentSpec) (*AgentBuild, error)
}

// AgentBuild holds the resolved components from building an AgentSpec.
type AgentBuild struct {
	// Spec is the original specification.
	Spec *AgentSpec
	// ProviderName is the resolved LLM provider name.
	ProviderName string
	// ModelName is the resolved model name.
	ModelName string
	// ToolNames are the resolved tool names.
	ToolNames []string
}

// defaultMaxSpecSize is the default maximum spec size in bytes (1 MB).
const defaultMaxSpecSize int64 = 1 << 20

// Option configures a JSONParser.
type Option func(*parserOptions)

type parserOptions struct {
	maxSize int64
}

// WithMaxSize sets the maximum spec size in bytes.
func WithMaxSize(n int64) Option {
	return func(o *parserOptions) { o.maxSize = n }
}

// JSONParser parses AgentSpec from JSON.
type JSONParser struct {
	opts parserOptions
}

var _ Parser = (*JSONParser)(nil)

// NewJSONParser creates a JSON parser with the given options.
func NewJSONParser(opts ...Option) *JSONParser {
	o := parserOptions{maxSize: defaultMaxSpecSize}
	for _, opt := range opts {
		opt(&o)
	}
	return &JSONParser{opts: o}
}

// Parse decodes JSON bytes into an AgentSpec.
func (p *JSONParser) Parse(_ context.Context, data []byte) (*AgentSpec, error) {
	if int64(len(data)) > p.opts.maxSize {
		return nil, fmt.Errorf("declarative: spec exceeds maximum size of %d bytes", p.opts.maxSize)
	}

	var spec AgentSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("declarative: invalid JSON: %w", err)
	}

	if err := validateSpec(&spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

// LoadSpec reads an AgentSpec from a file path.
// The file is bounded at defaultMaxSpecSize bytes at the I/O layer to
// avoid unbounded allocations from very large files or never-ending
// readers (e.g. named pipes). The raw input path is rejected if it
// contains any ".." segment — this must happen before filepath.Clean,
// which would otherwise resolve the traversal away for absolute paths
// such as "/tmp/../etc/passwd".
func LoadSpec(ctx context.Context, path string) (*AgentSpec, error) {
	// Reject path traversal in the raw input before cleaning collapses
	// ".." segments. Split on OS separators so we only catch true parent
	// references, not legitimate file names that contain "..".
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return nil, fmt.Errorf("declarative: path traversal not allowed: %q", path)
		}
	}
	cleanPath := filepath.Clean(path)

	f, err := os.Open(cleanPath) //nolint:gosec // path is validated above; callers supply trusted config paths
	if err != nil {
		return nil, fmt.Errorf("declarative: read file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Read at most defaultMaxSpecSize+1 bytes so we can detect overflow.
	data, err := io.ReadAll(io.LimitReader(f, defaultMaxSpecSize+1))
	if err != nil {
		return nil, fmt.Errorf("declarative: read file: %w", err)
	}
	if int64(len(data)) > defaultMaxSpecSize {
		return nil, fmt.Errorf("declarative: spec exceeds maximum size of %d bytes", defaultMaxSpecSize)
	}

	parser := NewJSONParser()
	return parser.Parse(ctx, data)
}

func validateSpec(spec *AgentSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("declarative: spec.id is required")
	}
	if spec.Persona.Role == "" {
		return fmt.Errorf("declarative: spec.persona.role is required")
	}
	if spec.Model.Provider == "" {
		return fmt.Errorf("declarative: spec.model.provider is required")
	}
	if spec.Model.Model == "" {
		return fmt.Errorf("declarative: spec.model.model is required")
	}
	if spec.MaxIterations < 0 {
		return fmt.Errorf("declarative: spec.max_iterations must be non-negative")
	}
	if spec.Model.Temperature != nil {
		if t := *spec.Model.Temperature; t < 0 || t > 2 {
			return fmt.Errorf("declarative: spec.model.temperature must be in [0, 2]")
		}
	}
	return nil
}

// DefaultBuilder builds AgentBuild from an AgentSpec by copying spec
// fields into the build. It does not perform registry lookups for
// providers or tools — callers are responsible for resolving names.
type DefaultBuilder struct{}

var _ Builder = (*DefaultBuilder)(nil)

// NewBuilder creates a new DefaultBuilder.
func NewBuilder() *DefaultBuilder {
	return &DefaultBuilder{}
}

// Build resolves an AgentSpec into an AgentBuild.
func (b *DefaultBuilder) Build(_ context.Context, spec *AgentSpec) (*AgentBuild, error) {
	if spec == nil {
		return nil, fmt.Errorf("declarative: spec is nil")
	}

	if err := validateSpec(spec); err != nil {
		return nil, err
	}

	return &AgentBuild{
		Spec:         spec,
		ProviderName: spec.Model.Provider,
		ModelName:    spec.Model.Model,
		ToolNames:    spec.Tools,
	}, nil
}

// MarshalSpec serializes an AgentSpec to JSON.
func MarshalSpec(spec *AgentSpec) ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}
