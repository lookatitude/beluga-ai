package iface

import (
	"time"

	chatmodelsiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
	multimodaliface "github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	orchestrationiface "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	serveriface "github.com/lookatitude/beluga-ai/pkg/server/iface"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// Config represents the configuration for voice backend providers.
// It includes common settings that apply to all voice backend providers.
type Config struct {
	// Provider is the backend provider name (e.g., "livekit", "pipecat").
	Provider string `mapstructure:"provider" yaml:"provider" validate:"required"`

	// ProviderConfig is provider-specific configuration.
	ProviderConfig map[string]any `mapstructure:"provider_config" yaml:"provider_config"`

	// PipelineType is the type of audio processing pipeline (STT_TTS or S2S).
	PipelineType PipelineType `mapstructure:"pipeline_type" yaml:"pipeline_type" validate:"required,oneof=stt_tts s2s"`

	// STTProvider is the STT provider name (if STT_TTS pipeline).
	STTProvider string `mapstructure:"stt_provider" yaml:"stt_provider" validate:"required_if=PipelineType stt_tts"`

	// TTSProvider is the TTS provider name (if STT_TTS pipeline).
	TTSProvider string `mapstructure:"tts_provider" yaml:"tts_provider" validate:"required_if=PipelineType stt_tts"`

	// S2SProvider is the S2S provider name (if S2S pipeline).
	S2SProvider string `mapstructure:"s2s_provider" yaml:"s2s_provider" validate:"required_if=PipelineType s2s"`

	// VADProvider is the VAD provider name (optional).
	VADProvider string `mapstructure:"vad_provider" yaml:"vad_provider"`

	// TurnDetectionProvider is the turn detection provider name (optional).
	TurnDetectionProvider string `mapstructure:"turn_detection_provider" yaml:"turn_detection_provider"`

	// NoiseCancellationProvider is the noise cancellation provider name (optional).
	NoiseCancellationProvider string `mapstructure:"noise_cancellation_provider" yaml:"noise_cancellation_provider"`

	// LatencyTarget is the target latency for end-to-end processing.
	LatencyTarget time.Duration `mapstructure:"latency_target" yaml:"latency_target" validate:"min=100ms,max=5s" default:"500ms"`

	// Timeout is the timeout for operations.
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" validate:"min=1s,max=5m" default:"30s"`

	// MaxRetries is the maximum number of retries for failed operations.
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries" validate:"gte=0,lte=10" default:"3"`

	// RetryDelay is the delay between retries.
	RetryDelay time.Duration `mapstructure:"retry_delay" yaml:"retry_delay" validate:"min=100ms,max=30s" default:"1s"`

	// MaxConcurrentSessions is the maximum number of concurrent sessions (0 = unlimited).
	MaxConcurrentSessions int `mapstructure:"max_concurrent_sessions" yaml:"max_concurrent_sessions" validate:"gte=0" default:"100"`

	// EnableTracing enables OTEL tracing.
	EnableTracing bool `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`

	// EnableMetrics enables OTEL metrics.
	EnableMetrics bool `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`

	// EnableStructuredLogging enables structured logging with OTEL context.
	EnableStructuredLogging bool `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`

	// Integration fields for deep package integration (all optional)
	Memory          memoryiface.Memory              `mapstructure:"-" yaml:"-"`
	Orchestrator    orchestrationiface.Orchestrator `mapstructure:"-" yaml:"-"`
	Retriever       interface{}                     `mapstructure:"-" yaml:"-"` // Retriever interface from pkg/retrievers/iface
	VectorStore     vectorstoresiface.VectorStore   `mapstructure:"-" yaml:"-"`
	Embedder        embeddingsiface.Embedder        `mapstructure:"-" yaml:"-"`
	MultimodalModel multimodaliface.MultimodalModel `mapstructure:"-" yaml:"-"`
	PromptTemplate  interface{}                     `mapstructure:"-" yaml:"-"` // PromptTemplate interface from pkg/prompts/iface
	ChatModel       chatmodelsiface.ChatModel       `mapstructure:"-" yaml:"-"`
	ServerConfig    serveriface.Config              `mapstructure:"-" yaml:"-"`

	// Extensibility hooks (optional)
	AuthHook          AuthHook          `mapstructure:"-" yaml:"-"`
	RateLimiter       RateLimiter       `mapstructure:"-" yaml:"-"`
	DataRetentionHook DataRetentionHook `mapstructure:"-" yaml:"-"`
	TelephonyHook     TelephonyHook     `mapstructure:"-" yaml:"-"` // For SIP and telephony protocol integration (T330, FR-015)
	CustomProcessors  []CustomProcessor `mapstructure:"-" yaml:"-"`
}
