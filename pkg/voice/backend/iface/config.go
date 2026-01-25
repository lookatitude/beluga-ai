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
	ChatModel                 chatmodelsiface.ChatModel       `mapstructure:"-" yaml:"-"`
	TelephonyHook             TelephonyHook                   `mapstructure:"-" yaml:"-"`
	DataRetentionHook         DataRetentionHook               `mapstructure:"-" yaml:"-"`
	RateLimiter               RateLimiter                     `mapstructure:"-" yaml:"-"`
	AuthHook                  AuthHook                        `mapstructure:"-" yaml:"-"`
	Memory                    memoryiface.Memory              `mapstructure:"-" yaml:"-"`
	PromptTemplate            any                             `mapstructure:"-" yaml:"-"`
	MultimodalModel           multimodaliface.MultimodalModel `mapstructure:"-" yaml:"-"`
	Embedder                  embeddingsiface.Embedder        `mapstructure:"-" yaml:"-"`
	VectorStore               vectorstoresiface.VectorStore   `mapstructure:"-" yaml:"-"`
	Retriever                 any                             `mapstructure:"-" yaml:"-"`
	Orchestrator              orchestrationiface.Orchestrator `mapstructure:"-" yaml:"-"`
	ProviderConfig            map[string]any                  `mapstructure:"provider_config" yaml:"provider_config"`
	TurnDetectionProvider     string                          `mapstructure:"turn_detection_provider" yaml:"turn_detection_provider"`
	S2SProvider               string                          `mapstructure:"s2s_provider" yaml:"s2s_provider" validate:"required_if=PipelineType s2s"`
	PipelineType              PipelineType                    `mapstructure:"pipeline_type" yaml:"pipeline_type" validate:"required,oneof=stt_tts s2s"`
	STTProvider               string                          `mapstructure:"stt_provider" yaml:"stt_provider" validate:"required_if=PipelineType stt_tts"`
	TTSProvider               string                          `mapstructure:"tts_provider" yaml:"tts_provider" validate:"required_if=PipelineType stt_tts"`
	VADProvider               string                          `mapstructure:"vad_provider" yaml:"vad_provider"`
	Provider                  string                          `mapstructure:"provider" yaml:"provider" validate:"required"`
	NoiseCancellationProvider string                          `mapstructure:"noise_cancellation_provider" yaml:"noise_cancellation_provider"`
	CustomProcessors          []CustomProcessor               `mapstructure:"-" yaml:"-"`
	ServerConfig              serveriface.Config              `mapstructure:"-" yaml:"-"`
	LatencyTarget             time.Duration                   `mapstructure:"latency_target" yaml:"latency_target" validate:"min=100ms,max=5s" default:"500ms"`
	Timeout                   time.Duration                   `mapstructure:"timeout" yaml:"timeout" validate:"min=1s,max=5m" default:"30s"`
	MaxRetries                int                             `mapstructure:"max_retries" yaml:"max_retries" validate:"gte=0,lte=10" default:"3"`
	RetryDelay                time.Duration                   `mapstructure:"retry_delay" yaml:"retry_delay" validate:"min=100ms,max=30s" default:"1s"`
	MaxConcurrentSessions     int                             `mapstructure:"max_concurrent_sessions" yaml:"max_concurrent_sessions" validate:"gte=0" default:"100"`
	EnableTracing             bool                            `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableStructuredLogging   bool                            `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
	EnableMetrics             bool                            `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
}
