package iface

import (
	"context"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
)

// SessionConfig represents the configuration for a new voice session.
type SessionConfig struct {
	UserID        string                                                       `mapstructure:"user_id" yaml:"user_id" validate:"required"`
	Transport     string                                                       `mapstructure:"transport" yaml:"transport" validate:"required,oneof=webrtc websocket"`
	ConnectionURL string                                                       `mapstructure:"connection_url" yaml:"connection_url" validate:"required,url"`
	AgentCallback func(ctx context.Context, transcript string) (string, error) `mapstructure:"-" yaml:"-"`
	AgentInstance agentsiface.Agent                                            `mapstructure:"-" yaml:"-"`
	PipelineType  PipelineType                                                 `mapstructure:"pipeline_type" yaml:"pipeline_type" validate:"required,oneof=stt_tts s2s"`
	Metadata      map[string]any                                               `mapstructure:"metadata" yaml:"metadata"`

	// Integration-specific configurations
	// Note: Using map[string]any for flexibility since Config types may vary
	MemoryConfig        map[string]any `mapstructure:"memory_config" yaml:"memory_config"`
	OrchestrationConfig map[string]any `mapstructure:"orchestration_config" yaml:"orchestration_config"`
	RAGConfig           map[string]any `mapstructure:"rag_config" yaml:"rag_config"`
}
