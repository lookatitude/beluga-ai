package agentfile

import (
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
	Serialize(af *AgentFile) ([]byte, error)
}

// Deserializer reads AgentFiles from bytes.
type Deserializer interface {
	// Deserialize converts bytes to an AgentFile.
	Deserialize(data []byte) (*AgentFile, error)
}

// VersionMigrator migrates AgentFiles between format versions.
type VersionMigrator interface {
	// Migrate updates an AgentFile to the target version.
	Migrate(af *AgentFile, targetVersion string) (*AgentFile, error)
	// SupportedVersions returns the versions this migrator can handle.
	SupportedVersions() []string
}

// CurrentVersion is the current agent file format version.
const CurrentVersion = "1.0"

// JSONSerializer implements Serializer for JSON format.
type JSONSerializer struct{}

var _ Serializer = (*JSONSerializer)(nil)

// Serialize writes an AgentFile as JSON.
func (s *JSONSerializer) Serialize(af *AgentFile) ([]byte, error) {
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
	d := &JSONDeserializer{maxSize: 1 << 20} // 1MB default
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Deserialize reads an AgentFile from JSON bytes.
func (d *JSONDeserializer) Deserialize(data []byte) (*AgentFile, error) {
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

// Migrate updates an AgentFile to the target version.
func (m *DefaultMigrator) Migrate(af *AgentFile, targetVersion string) (*AgentFile, error) {
	if af == nil {
		return nil, fmt.Errorf("agentfile: nil agent file")
	}

	if af.Version == targetVersion {
		return af, nil
	}

	result := *af
	result.Version = targetVersion
	result.UpdatedAt = time.Now()

	return &result, nil
}

// SupportedVersions returns the versions supported by this migrator.
func (m *DefaultMigrator) SupportedVersions() []string {
	return []string{"0.1", "1.0"}
}

// Save writes an AgentFile to disk.
func Save(path string, af *AgentFile) error {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("agentfile: path traversal not allowed: %q", path)
	}

	s := &JSONSerializer{}
	data, err := s.Serialize(af)
	if err != nil {
		return err
	}

	return os.WriteFile(cleanPath, data, 0600)
}

// Load reads an AgentFile from disk.
func Load(path string) (*AgentFile, error) {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("agentfile: path traversal not allowed: %q", path)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("agentfile: read file: %w", err)
	}

	d := NewDeserializer()
	return d.Deserialize(data)
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
