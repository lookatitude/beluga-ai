package agentfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AgentFile represents the .af agent file format.
type AgentFile struct {
	// Version is the file format version.
	Version string `json:"version"`
	// Agent contains the agent specification.
	Agent AgentDef `json:"agent"`
	// CreatedAt is when this file was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is when this file was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentDef defines an agent within an AgentFile.
type AgentDef struct {
	// ID is the unique agent identifier.
	ID string `json:"id"`
	// Name is a human-readable name.
	Name string `json:"name"`
	// Description explains the agent's purpose.
	Description string `json:"description,omitempty"`
	// Persona defines the agent's role and behavior.
	Persona PersonaDef `json:"persona"`
	// Model defines the LLM configuration.
	Model ModelDef `json:"model"`
	// Tools lists tool configurations.
	Tools []ToolDef `json:"tools,omitempty"`
	// Settings holds additional configuration.
	Settings map[string]any `json:"settings,omitempty"`
}

// PersonaDef defines persona within an AgentFile.
type PersonaDef struct {
	Role      string `json:"role"`
	Goal      string `json:"goal,omitempty"`
	Backstory string `json:"backstory,omitempty"`
}

// ModelDef defines model configuration within an AgentFile.
type ModelDef struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

// ToolDef defines a tool reference within an AgentFile.
type ToolDef struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config,omitempty"`
}

// Serializer writes AgentFiles to bytes.
type Serializer interface {
	// Serialize converts an AgentFile to bytes.
	Serialize(ctx context.Context, af *AgentFile) ([]byte, error)
}

// Deserializer reads AgentFiles from bytes.
type Deserializer interface {
	// Deserialize converts bytes to an AgentFile.
	Deserialize(ctx context.Context, data []byte) (*AgentFile, error)
}

// VersionMigrator migrates AgentFiles between format versions.
type VersionMigrator interface {
	// Migrate updates an AgentFile to the target version.
	Migrate(ctx context.Context, af *AgentFile, targetVersion string) (*AgentFile, error)
	// SupportedVersions returns the versions this migrator can handle.
	SupportedVersions() []string
}

// CurrentVersion is the current agent file format version.
const CurrentVersion = "1.0"

// maxFileSize is the maximum allowed agent file size (1 MB).
const maxFileSize int64 = 1 << 20

// JSONSerializer implements Serializer for JSON format.
type JSONSerializer struct{}

var _ Serializer = (*JSONSerializer)(nil)

// Serialize writes an AgentFile as JSON.
func (s *JSONSerializer) Serialize(ctx context.Context, af *AgentFile) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if af == nil {
		return nil, fmt.Errorf("agentfile: nil agent file")
	}
	return json.MarshalIndent(af, "", "  ")
}

// JSONDeserializer implements Deserializer for JSON format.
type JSONDeserializer struct {
	maxSize int64
}

var _ Deserializer = (*JSONDeserializer)(nil)

// Option configures a JSONDeserializer.
type Option func(*JSONDeserializer)

// WithMaxSize sets the maximum file size in bytes.
func WithMaxSize(n int64) Option {
	return func(d *JSONDeserializer) { d.maxSize = n }
}

// NewDeserializer creates a JSON deserializer.
func NewDeserializer(opts ...Option) *JSONDeserializer {
	d := &JSONDeserializer{maxSize: maxFileSize} // 1MB default
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Deserialize reads an AgentFile from JSON bytes.
func (d *JSONDeserializer) Deserialize(ctx context.Context, data []byte) (*AgentFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(data)) > d.maxSize {
		return nil, fmt.Errorf("agentfile: data exceeds maximum size of %d bytes", d.maxSize)
	}

	var af AgentFile
	if err := json.Unmarshal(data, &af); err != nil {
		return nil, fmt.Errorf("agentfile: invalid JSON: %w", err)
	}

	if err := validate(&af); err != nil {
		return nil, err
	}

	return &af, nil
}

func validate(af *AgentFile) error {
	if af.Version == "" {
		return fmt.Errorf("agentfile: version is required")
	}
	if af.Agent.ID == "" {
		return fmt.Errorf("agentfile: agent.id is required")
	}
	if af.Agent.Persona.Role == "" {
		return fmt.Errorf("agentfile: agent.persona.role is required")
	}
	if af.Agent.Model.Provider == "" {
		return fmt.Errorf("agentfile: agent.model.provider is required")
	}
	if af.Agent.Model.Model == "" {
		return fmt.Errorf("agentfile: agent.model.model is required")
	}
	return nil
}

// DefaultMigrator handles version migrations for the .af format.
type DefaultMigrator struct{}

var _ VersionMigrator = (*DefaultMigrator)(nil)

// NewMigrator creates a new version migrator.
func NewMigrator() *DefaultMigrator {
	return &DefaultMigrator{}
}

// deepCopy returns a deep copy of an AgentFile via JSON round-trip, ensuring
// that map and slice fields are not aliased between the original and the copy.
func deepCopy(af *AgentFile) (*AgentFile, error) {
	b, err := json.Marshal(af)
	if err != nil {
		return nil, fmt.Errorf("agentfile: deep copy marshal: %w", err)
	}
	var cp AgentFile
	if err := json.Unmarshal(b, &cp); err != nil {
		return nil, fmt.Errorf("agentfile: deep copy unmarshal: %w", err)
	}
	return &cp, nil
}

// Migrate updates an AgentFile to the target version.
func (m *DefaultMigrator) Migrate(ctx context.Context, af *AgentFile, targetVersion string) (*AgentFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if af == nil {
		return nil, fmt.Errorf("agentfile: nil agent file")
	}

	supported := map[string]bool{}
	for _, v := range m.SupportedVersions() {
		supported[v] = true
	}
	if !supported[targetVersion] {
		return nil, fmt.Errorf("agentfile: unsupported target version %q", targetVersion)
	}

	result, err := deepCopy(af)
	if err != nil {
		return nil, err
	}

	if result.Version == targetVersion {
		return result, nil
	}

	result.Version = targetVersion
	result.UpdatedAt = time.Now()

	return result, nil
}

// SupportedVersions returns the versions supported by this migrator.
func (m *DefaultMigrator) SupportedVersions() []string {
	return []string{"0.1", "1.0"}
}

// Save writes an AgentFile to disk.
func Save(ctx context.Context, path string, af *AgentFile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Check the original path before cleaning so that traversal segments are
	// not silently collapsed by filepath.Clean.
	if strings.Contains(path, "..") {
		return fmt.Errorf("agentfile: path traversal not allowed: %q", path)
	}
	cleanPath := filepath.Clean(path)

	s := &JSONSerializer{}
	data, err := s.Serialize(ctx, af)
	if err != nil {
		return err
	}

	return os.WriteFile(cleanPath, data, 0600) //nolint:gosec // G304: path is validated and cleaned above
}

// Load reads an AgentFile from disk.
func Load(ctx context.Context, path string) (*AgentFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Check the original path before cleaning so that traversal segments are
	// not silently collapsed by filepath.Clean.
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("agentfile: path traversal not allowed: %q", path)
	}
	cleanPath := filepath.Clean(path)

	// Enforce the size limit before reading to avoid unbounded allocations
	// from external input.
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("agentfile: stat file: %w", err)
	}
	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("agentfile: file exceeds maximum size of %d bytes", maxFileSize)
	}

	data, err := os.ReadFile(cleanPath) //nolint:gosec // G304: path is validated and cleaned above
	if err != nil {
		return nil, fmt.Errorf("agentfile: read file: %w", err)
	}

	d := NewDeserializer()
	return d.Deserialize(ctx, data)
}

// NewAgentFile creates a new AgentFile with defaults.
func NewAgentFile(id, role, provider, model string) *AgentFile {
	now := time.Now()
	return &AgentFile{
		Version: CurrentVersion,
		Agent: AgentDef{
			ID:      id,
			Name:    id,
			Persona: PersonaDef{Role: role},
			Model:   ModelDef{Provider: provider, Model: model},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
