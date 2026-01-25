package backend

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	chatmodelsiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	multimodaliface "github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	orchestrationiface "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// Config is now in iface package - use iface.Config
// This file contains ConfigOption functions and validation helpers

// Config is an alias for iface.Config for backward compatibility.
type Config = vbiface.Config

// ConfigOption is a functional option for configuring voice backend instances.
type ConfigOption func(*vbiface.Config)

// WithProvider sets the backend provider.
func WithProvider(provider string) ConfigOption {
	return func(c *vbiface.Config) {
		c.Provider = provider
	}
}

// WithPipelineType sets the pipeline type.
func WithPipelineType(pipelineType vbiface.PipelineType) ConfigOption {
	return func(c *vbiface.Config) {
		c.PipelineType = pipelineType
	}
}

// WithLatencyTarget sets the latency target.
func WithLatencyTarget(target time.Duration) ConfigOption {
	return func(c *vbiface.Config) {
		c.LatencyTarget = target
	}
}

// WithMaxConcurrentSessions sets the maximum concurrent sessions.
func WithMaxConcurrentSessions(max int) ConfigOption {
	return func(c *vbiface.Config) {
		c.MaxConcurrentSessions = max
	}
}

// WithMemory sets the memory integration.
func WithMemory(memory memoryiface.Memory) ConfigOption {
	return func(c *vbiface.Config) {
		c.Memory = memory
	}
}

// WithOrchestrator sets the orchestrator integration.
func WithOrchestrator(orchestrator orchestrationiface.Orchestrator) ConfigOption {
	return func(c *vbiface.Config) {
		c.Orchestrator = orchestrator
	}
}

// WithRetriever sets the retriever integration.
func WithRetriever(retriever any) ConfigOption {
	return func(c *vbiface.Config) {
		c.Retriever = retriever
	}
}

// WithVectorStore sets the vector store integration.
func WithVectorStore(vectorStore vectorstoresiface.VectorStore) ConfigOption {
	return func(c *vbiface.Config) {
		c.VectorStore = vectorStore
	}
}

// WithEmbedder sets the embedder integration.
func WithEmbedder(embedder embeddingsiface.Embedder) ConfigOption {
	return func(c *vbiface.Config) {
		c.Embedder = embedder
	}
}

// WithMultimodalModel sets the multimodal model integration.
func WithMultimodalModel(model multimodaliface.MultimodalModel) ConfigOption {
	return func(c *vbiface.Config) {
		c.MultimodalModel = model
	}
}

// WithPromptTemplate sets the prompt template integration.
func WithPromptTemplate(template any) ConfigOption {
	return func(c *vbiface.Config) {
		c.PromptTemplate = template
	}
}

// WithChatModel sets the chat model integration.
func WithChatModel(chatModel chatmodelsiface.ChatModel) ConfigOption {
	return func(c *vbiface.Config) {
		c.ChatModel = chatModel
	}
}

// WithAuthHook sets the authentication hook.
func WithAuthHook(hook vbiface.AuthHook) ConfigOption {
	return func(c *vbiface.Config) {
		c.AuthHook = hook
	}
}

// WithRateLimiter sets the rate limiter hook.
func WithRateLimiter(limiter vbiface.RateLimiter) ConfigOption {
	return func(c *vbiface.Config) {
		c.RateLimiter = limiter
	}
}

// WithDataRetentionHook sets the data retention hook.
func WithDataRetentionHook(hook vbiface.DataRetentionHook) ConfigOption {
	return func(c *vbiface.Config) {
		c.DataRetentionHook = hook
	}
}

// WithCustomProcessor adds a custom audio processor to the pipeline.
func WithCustomProcessor(processor vbiface.CustomProcessor) ConfigOption {
	return func(c *vbiface.Config) {
		c.CustomProcessors = append(c.CustomProcessors, processor)
	}
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *vbiface.Config {
	return &vbiface.Config{
		Provider:                  "",
		ProviderConfig:            make(map[string]any),
		PipelineType:              vbiface.PipelineTypeSTTTTS,
		STTProvider:               "",
		TTSProvider:               "",
		S2SProvider:               "",
		VADProvider:               "",
		TurnDetectionProvider:     "",
		NoiseCancellationProvider: "",
		LatencyTarget:             500 * time.Millisecond,
		Timeout:                   30 * time.Second,
		MaxRetries:                3,
		RetryDelay:                time.Second,
		MaxConcurrentSessions:     100,
		EnableTracing:             true,
		EnableMetrics:             true,
		EnableStructuredLogging:   true,
		CustomProcessors:          []vbiface.CustomProcessor{},
	}
}

// ValidateConfig validates the configuration.
func ValidateConfig(config *vbiface.Config) error {
	if config == nil {
		return errors.New("configuration cannot be nil")
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Additional validation based on pipeline type
	switch config.PipelineType {
	case vbiface.PipelineTypeSTTTTS:
		if config.STTProvider == "" {
			return errors.New("stt_provider is required for STT_TTS pipeline")
		}
		if config.TTSProvider == "" {
			return errors.New("tts_provider is required for STT_TTS pipeline")
		}
	case vbiface.PipelineTypeS2S:
		if config.S2SProvider == "" {
			return errors.New("s2s_provider is required for S2S pipeline")
		}
	}

	// Validate latency target
	if config.LatencyTarget < 100*time.Millisecond {
		return fmt.Errorf("latency_target must be at least 100ms, got %v", config.LatencyTarget)
	}
	if config.LatencyTarget > 5*time.Second {
		return fmt.Errorf("latency_target must be at most 5s, got %v", config.LatencyTarget)
	}

	// Validate timeout
	if config.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1s, got %v", config.Timeout)
	}
	if config.Timeout > 5*time.Minute {
		return fmt.Errorf("timeout must be at most 5m, got %v", config.Timeout)
	}

	return nil
}

// ValidateSessionConfig validates the session configuration.
func ValidateSessionConfig(config *vbiface.SessionConfig) error {
	if config == nil {
		return errors.New("session configuration cannot be nil")
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return fmt.Errorf("session configuration validation failed: %w", err)
	}

	// Validate that either AgentCallback or AgentInstance is set
	if config.AgentCallback == nil && config.AgentInstance == nil {
		return errors.New("either agent_callback or agent_instance must be set")
	}

	return nil
}
