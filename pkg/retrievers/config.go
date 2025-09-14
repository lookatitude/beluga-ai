// Package retrievers provides configuration structs for retriever components.
package retrievers

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/retrievers/iface"
)

// Config holds the configuration for retrievers.
type Config struct {
	// Search configuration
	DefaultK       int           `mapstructure:"default_k" yaml:"default_k" env:"RETRIEVERS_DEFAULT_K" default:"4" validate:"min=1,max=100"`
	ScoreThreshold float32       `mapstructure:"score_threshold" yaml:"score_threshold" env:"RETRIEVERS_SCORE_THRESHOLD" default:"0.0" validate:"min=0,max=1"`
	MaxRetries     int           `mapstructure:"max_retries" yaml:"max_retries" env:"RETRIEVERS_MAX_RETRIES" default:"3" validate:"min=0,max=10"`
	Timeout        time.Duration `mapstructure:"timeout" yaml:"timeout" env:"RETRIEVERS_TIMEOUT" default:"30s" validate:"min=1s,max=5m"`
	EnableTracing  bool          `mapstructure:"enable_tracing" yaml:"enable_tracing" env:"RETRIEVERS_ENABLE_TRACING" default:"true"`
	EnableMetrics  bool          `mapstructure:"enable_metrics" yaml:"enable_metrics" env:"RETRIEVERS_ENABLE_METRICS" default:"true"`

	// Vector store specific configuration
	VectorStoreConfig VectorStoreConfig `mapstructure:"vector_store" yaml:"vector_store"`
}

// VectorStoreConfig holds vector store specific configuration.
type VectorStoreConfig struct {
	// Common vector store settings
	MaxBatchSize int `mapstructure:"max_batch_size" yaml:"max_batch_size" env:"VECTOR_STORE_MAX_BATCH_SIZE" default:"100" validate:"min=1,max=1000"`

	// Search optimization settings
	EnableMMR          bool    `mapstructure:"enable_mmr" yaml:"enable_mmr" env:"VECTOR_STORE_ENABLE_MMR" default:"false"`
	MMRLambda          float32 `mapstructure:"mmr_lambda" yaml:"mmr_lambda" env:"VECTOR_STORE_MMR_LAMBDA" default:"0.5" validate:"min=0,max=1"`
	DiversityThreshold float32 `mapstructure:"diversity_threshold" yaml:"diversity_threshold" env:"VECTOR_STORE_DIVERSITY_THRESHOLD" default:"0.7" validate:"min=0,max=1"`
}

// VectorStoreRetrieverConfig holds configuration specific to VectorStoreRetriever.
type VectorStoreRetrieverConfig struct {
	// Embedder configuration
	Embedder iface.Embedder `mapstructure:"-" yaml:"-"`

	// Search parameters
	K              int     `mapstructure:"k" yaml:"k" env:"VECTOR_STORE_RETRIEVER_K" default:"4" validate:"min=1,max=100"`
	ScoreThreshold float32 `mapstructure:"score_threshold" yaml:"score_threshold" env:"VECTOR_STORE_RETRIEVER_SCORE_THRESHOLD" default:"0.0" validate:"min=0,max=1"`

	// Filtering options
	MetadataFilter map[string]any `mapstructure:"metadata_filter" yaml:"metadata_filter"`

	// Performance settings
	BatchSize int           `mapstructure:"batch_size" yaml:"batch_size" env:"VECTOR_STORE_RETRIEVER_BATCH_SIZE" default:"10" validate:"min=1,max=100"`
	Timeout   time.Duration `mapstructure:"timeout" yaml:"timeout" env:"VECTOR_STORE_RETRIEVER_TIMEOUT" default:"30s" validate:"min=1s,max=5m"`
}

// DefaultConfig returns a default configuration for retrievers.
func DefaultConfig() Config {
	return Config{
		DefaultK:       4,
		ScoreThreshold: 0.0,
		MaxRetries:     3,
		Timeout:        30 * time.Second,
		EnableTracing:  true,
		EnableMetrics:  true,
		VectorStoreConfig: VectorStoreConfig{
			MaxBatchSize:       100,
			EnableMMR:          false,
			MMRLambda:          0.5,
			DiversityThreshold: 0.7,
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.DefaultK < 1 || c.DefaultK > 100 {
		return &ValidationError{
			Field: "DefaultK",
			Value: c.DefaultK,
			Msg:   "must be between 1 and 100",
		}
	}
	if c.ScoreThreshold < 0 || c.ScoreThreshold > 1 {
		return &ValidationError{
			Field: "ScoreThreshold",
			Value: c.ScoreThreshold,
			Msg:   "must be between 0 and 1",
		}
	}
	if c.Timeout < time.Second || c.Timeout > 5*time.Minute {
		return &ValidationError{
			Field: "Timeout",
			Value: c.Timeout,
			Msg:   "must be between 1s and 5m",
		}
	}
	return nil
}

// Validate validates the VectorStoreRetrieverConfig.
func (c *VectorStoreRetrieverConfig) Validate() error {
	if c.K < 1 || c.K > 100 {
		return &ValidationError{
			Field: "K",
			Value: c.K,
			Msg:   "must be between 1 and 100",
		}
	}
	if c.ScoreThreshold < 0 || c.ScoreThreshold > 1 {
		return &ValidationError{
			Field: "ScoreThreshold",
			Value: c.ScoreThreshold,
			Msg:   "must be between 0 and 1",
		}
	}
	if c.BatchSize < 1 || c.BatchSize > 100 {
		return &ValidationError{
			Field: "BatchSize",
			Value: c.BatchSize,
			Msg:   "must be between 1 and 100",
		}
	}
	if c.Timeout < time.Second || c.Timeout > 5*time.Minute {
		return &ValidationError{
			Field: "Timeout",
			Value: c.Timeout,
			Msg:   "must be between 1s and 5m",
		}
	}
	return nil
}
