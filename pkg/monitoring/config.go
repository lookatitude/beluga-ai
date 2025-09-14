package monitoring

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

// Config contains comprehensive configuration for the monitoring system
type Config struct {
	// Service identification
	ServiceName string `mapstructure:"service_name" yaml:"service_name" env:"SERVICE_NAME" validate:"required,min=1,max=100"`

	// OpenTelemetry configuration
	OpenTelemetry OpenTelemetryConfig `mapstructure:"opentelemetry" yaml:"opentelemetry"`

	// Logging configuration
	Logging LoggingConfig `mapstructure:"logging" yaml:"logging"`

	// Tracing configuration
	Tracing TracingConfig `mapstructure:"tracing" yaml:"tracing"`

	// Metrics configuration
	Metrics MetricsConfig `mapstructure:"metrics" yaml:"metrics"`

	// Safety and ethics configuration
	Safety SafetyConfig `mapstructure:"safety" yaml:"safety"`
	Ethics EthicsConfig `mapstructure:"ethics" yaml:"ethics"`

	// Health check configuration
	Health HealthConfig `mapstructure:"health" yaml:"health"`

	// Best practices configuration
	BestPractices BestPracticesConfig `mapstructure:"best_practices" yaml:"best_practices"`
}

// OpenTelemetryConfig configures OpenTelemetry integration
type OpenTelemetryConfig struct {
	Enabled        bool              `mapstructure:"enabled" yaml:"enabled" env:"OTEL_ENABLED" default:"false"`
	Endpoint       string            `mapstructure:"endpoint" yaml:"endpoint" env:"OTEL_ENDPOINT" default:"localhost:4317"`
	ServiceName    string            `mapstructure:"service_name" yaml:"service_name" env:"OTEL_SERVICE_NAME"`
	ServiceVersion string            `mapstructure:"service_version" yaml:"service_version" env:"OTEL_SERVICE_VERSION" default:"1.0.0"`
	Environment    string            `mapstructure:"environment" yaml:"environment" env:"OTEL_ENVIRONMENT" default:"development"`
	ResourceAttrs  map[string]string `mapstructure:"resource_attrs" yaml:"resource_attrs"`
	SampleRate     float64           `mapstructure:"sample_rate" yaml:"sample_rate" env:"OTEL_SAMPLE_RATE" default:"1.0" validate:"min=0,max=1"`
}

// LoggingConfig configures structured logging
type LoggingConfig struct {
	Enabled       bool   `mapstructure:"enabled" yaml:"enabled" env:"LOG_ENABLED" default:"true"`
	Level         string `mapstructure:"level" yaml:"level" env:"LOG_LEVEL" default:"info" validate:"oneof=debug info warning error fatal"`
	Format        string `mapstructure:"format" yaml:"format" env:"LOG_FORMAT" default:"json" validate:"oneof=json text"`
	OutputFile    string `mapstructure:"output_file" yaml:"output_file" env:"LOG_OUTPUT_FILE"`
	UseColors     bool   `mapstructure:"use_colors" yaml:"use_colors" env:"LOG_USE_COLORS" default:"true"`
	MaxFileSize   int64  `mapstructure:"max_file_size" yaml:"max_file_size" env:"LOG_MAX_FILE_SIZE" default:"10485760" validate:"min=1024"` // 10MB
	MaxBackups    int    `mapstructure:"max_backups" yaml:"max_backups" env:"LOG_MAX_BACKUPS" default:"5" validate:"min=1,max=100"`
	Compress      bool   `mapstructure:"compress" yaml:"compress" env:"LOG_COMPRESS" default:"true"`
	IncludeCaller bool   `mapstructure:"include_caller" yaml:"include_caller" env:"LOG_INCLUDE_CALLER" default:"true"`
	IncludeTrace  bool   `mapstructure:"include_trace" yaml:"include_trace" env:"LOG_INCLUDE_TRACE" default:"true"`
}

// TracingConfig configures distributed tracing
type TracingConfig struct {
	Enabled           bool              `mapstructure:"enabled" yaml:"enabled" env:"TRACING_ENABLED" default:"true"`
	SampleRate        float64           `mapstructure:"sample_rate" yaml:"sample_rate" env:"TRACING_SAMPLE_RATE" default:"1.0" validate:"min=0,max=1"`
	MaxSpansPerSecond int               `mapstructure:"max_spans_per_second" yaml:"max_spans_per_second" env:"TRACING_MAX_SPANS_PER_SECOND" default:"1000" validate:"min=1"`
	SpanBufferSize    int               `mapstructure:"span_buffer_size" yaml:"span_buffer_size" env:"TRACING_SPAN_BUFFER_SIZE" default:"10000" validate:"min=100"`
	ExportTimeout     time.Duration     `mapstructure:"export_timeout" yaml:"export_timeout" env:"TRACING_EXPORT_TIMEOUT" default:"30s" validate:"min=1s,max=5m"`
	Tags              map[string]string `mapstructure:"tags" yaml:"tags"`
}

// MetricsConfig configures metrics collection
type MetricsConfig struct {
	Enabled              bool               `mapstructure:"enabled" yaml:"enabled" env:"METRICS_ENABLED" default:"true"`
	CollectionInterval   time.Duration      `mapstructure:"collection_interval" yaml:"collection_interval" env:"METRICS_COLLECTION_INTERVAL" default:"60s" validate:"min=1s,max=5m"`
	ExportInterval       time.Duration      `mapstructure:"export_interval" yaml:"export_interval" env:"METRICS_EXPORT_INTERVAL" default:"30s" validate:"min=1s,max=5m"`
	HistogramBuckets     []float64          `mapstructure:"histogram_buckets" yaml:"histogram_buckets"`
	SummaryObjectives    map[string]float64 `mapstructure:"summary_objectives" yaml:"summary_objectives"`
	Tags                 map[string]string  `mapstructure:"tags" yaml:"tags"`
	EnableRuntimeMetrics bool               `mapstructure:"enable_runtime_metrics" yaml:"enable_runtime_metrics" env:"METRICS_RUNTIME_ENABLED" default:"true"`
	EnableGCMetrics      bool               `mapstructure:"enable_gc_metrics" yaml:"enable_gc_metrics" env:"METRICS_GC_ENABLED" default:"true"`
}

// SafetyConfig configures safety validation
type SafetyConfig struct {
	Enabled            bool                `mapstructure:"enabled" yaml:"enabled" env:"SAFETY_ENABLED" default:"true"`
	RiskThreshold      float64             `mapstructure:"risk_threshold" yaml:"risk_threshold" env:"SAFETY_RISK_THRESHOLD" default:"0.7" validate:"min=0,max=1"`
	AutoBlockHighRisk  bool                `mapstructure:"auto_block_high_risk" yaml:"auto_block_high_risk" env:"SAFETY_AUTO_BLOCK" default:"true"`
	ToxicityPatterns   []string            `mapstructure:"toxicity_patterns" yaml:"toxicity_patterns"`
	BiasPatterns       []string            `mapstructure:"bias_patterns" yaml:"bias_patterns"`
	HarmfulPatterns    []string            `mapstructure:"harmful_patterns" yaml:"harmful_patterns"`
	CustomPatterns     map[string][]string `mapstructure:"custom_patterns" yaml:"custom_patterns"`
	EnableHumanReview  bool                `mapstructure:"enable_human_review" yaml:"enable_human_review" env:"SAFETY_HUMAN_REVIEW" default:"false"`
	HumanReviewTimeout time.Duration       `mapstructure:"human_review_timeout" yaml:"human_review_timeout" env:"SAFETY_REVIEW_TIMEOUT" default:"5m" validate:"min=30s,max=1h"`
}

// EthicsConfig configures ethical AI validation
type EthicsConfig struct {
	Enabled              bool          `mapstructure:"enabled" yaml:"enabled" env:"ETHICS_ENABLED" default:"true"`
	BiasDetectionEnabled bool          `mapstructure:"bias_detection_enabled" yaml:"bias_detection_enabled" env:"ETHICS_BIAS_ENABLED" default:"true"`
	PrivacyCheckEnabled  bool          `mapstructure:"privacy_check_enabled" yaml:"privacy_check_enabled" env:"ETHICS_PRIVACY_ENABLED" default:"true"`
	FairnessThreshold    float64       `mapstructure:"fairness_threshold" yaml:"fairness_threshold" env:"ETHICS_FAIRNESS_THRESHOLD" default:"0.7" validate:"min=0,max=1"`
	CulturalContexts     []string      `mapstructure:"cultural_contexts" yaml:"cultural_contexts"`
	SensitiveTopics      []string      `mapstructure:"sensitive_topics" yaml:"sensitive_topics"`
	StakeholderGroups    []string      `mapstructure:"stakeholder_groups" yaml:"stakeholder_groups"`
	RequireHumanApproval bool          `mapstructure:"require_human_approval" yaml:"require_human_approval" env:"ETHICS_HUMAN_APPROVAL" default:"false"`
	ApprovalTimeout      time.Duration `mapstructure:"approval_timeout" yaml:"approval_timeout" env:"ETHICS_APPROVAL_TIMEOUT" default:"10m" validate:"min=1m,max=2h"`
}

// HealthConfig configures health monitoring
type HealthConfig struct {
	Enabled          bool              `mapstructure:"enabled" yaml:"enabled" env:"HEALTH_ENABLED" default:"true"`
	CheckInterval    time.Duration     `mapstructure:"check_interval" yaml:"check_interval" env:"HEALTH_CHECK_INTERVAL" default:"30s" validate:"min=5s,max=5m"`
	Timeout          time.Duration     `mapstructure:"timeout" yaml:"timeout" env:"HEALTH_TIMEOUT" default:"10s" validate:"min=1s,max=1m"`
	MaxRetries       int               `mapstructure:"max_retries" yaml:"max_retries" env:"HEALTH_MAX_RETRIES" default:"3" validate:"min=0,max=10"`
	RetryDelay       time.Duration     `mapstructure:"retry_delay" yaml:"retry_delay" env:"HEALTH_RETRY_DELAY" default:"2s" validate:"min=100ms,max=30s"`
	FailureThreshold int               `mapstructure:"failure_threshold" yaml:"failure_threshold" env:"HEALTH_FAILURE_THRESHOLD" default:"3" validate:"min=1,max=10"`
	SuccessThreshold int               `mapstructure:"success_threshold" yaml:"success_threshold" env:"HEALTH_SUCCESS_THRESHOLD" default:"2" validate:"min=1,max=10"`
	Tags             map[string]string `mapstructure:"tags" yaml:"tags"`
}

// BestPracticesConfig configures best practices validation
type BestPracticesConfig struct {
	Enabled              bool     `mapstructure:"enabled" yaml:"enabled" env:"BEST_PRACTICES_ENABLED" default:"true"`
	Validators           []string `mapstructure:"validators" yaml:"validators"`
	ConcurrencyEnabled   bool     `mapstructure:"concurrency_enabled" yaml:"concurrency_enabled" env:"BEST_PRACTICES_CONCURRENCY" default:"true"`
	ErrorHandlingEnabled bool     `mapstructure:"error_handling_enabled" yaml:"error_handling_enabled" env:"BEST_PRACTICES_ERROR_HANDLING" default:"true"`
	ResourceMgmtEnabled  bool     `mapstructure:"resource_mgmt_enabled" yaml:"resource_mgmt_enabled" env:"BEST_PRACTICES_RESOURCE_MGMT" default:"true"`
	SecurityEnabled      bool     `mapstructure:"security_enabled" yaml:"security_enabled" env:"BEST_PRACTICES_SECURITY" default:"true"`
	CustomValidators     []string `mapstructure:"custom_validators" yaml:"custom_validators"`
}

// DefaultConfig returns a default configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		ServiceName: "beluga-ai-service",
		OpenTelemetry: OpenTelemetryConfig{
			Enabled:        false,
			Endpoint:       "localhost:4317",
			ServiceVersion: "1.0.0",
			Environment:    "development",
			ResourceAttrs:  make(map[string]string),
			SampleRate:     1.0,
		},
		Logging: LoggingConfig{
			Enabled:       true,
			Level:         "info",
			Format:        "json",
			UseColors:     true,
			MaxFileSize:   10 * 1024 * 1024, // 10MB
			MaxBackups:    5,
			Compress:      true,
			IncludeCaller: true,
			IncludeTrace:  true,
		},
		Tracing: TracingConfig{
			Enabled:           true,
			SampleRate:        1.0,
			MaxSpansPerSecond: 1000,
			SpanBufferSize:    10000,
			ExportTimeout:     30 * time.Second,
			Tags:              make(map[string]string),
		},
		Metrics: MetricsConfig{
			Enabled:              true,
			CollectionInterval:   60 * time.Second,
			ExportInterval:       30 * time.Second,
			HistogramBuckets:     []float64{.005, .01, .025, .05, .1, .25, .5, 1.0, 2.5, 5.0, 10.0},
			SummaryObjectives:    map[string]float64{"0.5": 0.05, "0.9": 0.01, "0.99": 0.001},
			Tags:                 make(map[string]string),
			EnableRuntimeMetrics: true,
			EnableGCMetrics:      true,
		},
		Safety: SafetyConfig{
			Enabled:            true,
			RiskThreshold:      0.7,
			AutoBlockHighRisk:  true,
			ToxicityPatterns:   getDefaultToxicityPatterns(),
			BiasPatterns:       getDefaultBiasPatterns(),
			HarmfulPatterns:    getDefaultHarmfulPatterns(),
			CustomPatterns:     make(map[string][]string),
			EnableHumanReview:  false,
			HumanReviewTimeout: 5 * time.Minute,
		},
		Ethics: EthicsConfig{
			Enabled:              true,
			BiasDetectionEnabled: true,
			PrivacyCheckEnabled:  true,
			FairnessThreshold:    0.7,
			CulturalContexts:     []string{"global", "western", "eastern"},
			SensitiveTopics:      []string{"politics", "religion", "health", "finance"},
			StakeholderGroups:    []string{"users", "developers", "business", "regulators"},
			RequireHumanApproval: false,
			ApprovalTimeout:      10 * time.Minute,
		},
		Health: HealthConfig{
			Enabled:          true,
			CheckInterval:    30 * time.Second,
			Timeout:          10 * time.Second,
			MaxRetries:       3,
			RetryDelay:       2 * time.Second,
			FailureThreshold: 3,
			SuccessThreshold: 2,
			Tags:             make(map[string]string),
		},
		BestPractices: BestPracticesConfig{
			Enabled:              true,
			Validators:           []string{"concurrency", "error_handling", "resource_management", "security"},
			ConcurrencyEnabled:   true,
			ErrorHandlingEnabled: true,
			ResourceMgmtEnabled:  true,
			SecurityEnabled:      true,
			CustomValidators:     []string{},
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	validate := validator.New()

	// Validate main config
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Cross-field validations
	if err := c.validateCrossFieldRules(); err != nil {
		return fmt.Errorf("cross-field validation failed: %w", err)
	}

	return nil
}

// validateCrossFieldRules performs cross-field validation
func (c *Config) validateCrossFieldRules() error {
	// If OpenTelemetry is enabled, endpoint must be provided
	if c.OpenTelemetry.Enabled && c.OpenTelemetry.Endpoint == "" {
		return fmt.Errorf("opentelemetry endpoint is required when opentelemetry is enabled")
	}

	// If tracing is enabled, sampling rate must be valid
	if c.Tracing.Enabled && (c.Tracing.SampleRate < 0 || c.Tracing.SampleRate > 1) {
		return fmt.Errorf("tracing sample rate must be between 0 and 1")
	}

	// If safety is enabled and human review is enabled, timeout must be reasonable
	if c.Safety.Enabled && c.Safety.EnableHumanReview && c.Safety.HumanReviewTimeout < 30*time.Second {
		return fmt.Errorf("human review timeout must be at least 30 seconds")
	}

	// If ethics is enabled and human approval is required, timeout must be reasonable
	if c.Ethics.Enabled && c.Ethics.RequireHumanApproval && c.Ethics.ApprovalTimeout < 1*time.Minute {
		return fmt.Errorf("ethics approval timeout must be at least 1 minute")
	}

	// Health check intervals must be reasonable
	if c.Health.Enabled && c.Health.CheckInterval < 5*time.Second {
		return fmt.Errorf("health check interval must be at least 5 seconds")
	}

	return nil
}

// getDefaultToxicityPatterns returns default toxicity detection patterns
func getDefaultToxicityPatterns() []string {
	return []string{
		"(?i)(hate|kill|murder|violence|terror)",
		"(?i)(racist|sexist|homophobic|transphobic)",
		"(?i)(suicide|self-harm|depression)",
		"(?i)(threat|attack|intimidate)",
	}
}

// getDefaultBiasPatterns returns default bias detection patterns
func getDefaultBiasPatterns() []string {
	return []string{
		"(?i)(all (men|women|people) are|everyone knows)",
		"(?i)(typical|normal|average) (man|woman|person)",
		"(?i)(obviously|clearly|of course)",
		"(?i)(they|those people).*?(don't|can't|won't)",
	}
}

// getDefaultHarmfulPatterns returns default harmful content patterns
func getDefaultHarmfulPatterns() []string {
	return []string{
		"(?i)(how to|tutorial|guide).*?(hack|exploit|crack)",
		"(?i)(illegal|illicit|forbidden).*?(activity|method|technique)",
		"(?i)(manufacture|build|make).*?(weapon|bomb|explosive)",
		"(?i)(produce|create).*?(drug|narcotic|controlled substance)",
	}
}

// ConfigOption represents functional options for configuration
type ConfigOption func(*Config)

// WithServiceName sets the service name
func WithServiceName(name string) ConfigOption {
	return func(c *Config) {
		c.ServiceName = name
	}
}

// WithOpenTelemetry enables OpenTelemetry with endpoint
func WithOpenTelemetry(endpoint string) ConfigOption {
	return func(c *Config) {
		c.OpenTelemetry.Enabled = true
		c.OpenTelemetry.Endpoint = endpoint
	}
}

// WithLogging configures logging
func WithLogging(level string, format string) ConfigOption {
	return func(c *Config) {
		c.Logging.Level = level
		c.Logging.Format = format
	}
}

// WithTracing configures tracing
func WithTracing(sampleRate float64) ConfigOption {
	return func(c *Config) {
		c.Tracing.SampleRate = sampleRate
	}
}

// WithSafety configures safety validation
func WithSafety(riskThreshold float64, humanReview bool) ConfigOption {
	return func(c *Config) {
		c.Safety.RiskThreshold = riskThreshold
		c.Safety.EnableHumanReview = humanReview
	}
}

// WithEthics configures ethical validation
func WithEthics(fairnessThreshold float64, requireApproval bool) ConfigOption {
	return func(c *Config) {
		c.Ethics.FairnessThreshold = fairnessThreshold
		c.Ethics.RequireHumanApproval = requireApproval
	}
}

// WithHealth configures health monitoring
func WithHealth(checkInterval time.Duration) ConfigOption {
	return func(c *Config) {
		c.Health.CheckInterval = checkInterval
	}
}

// LoadConfig loads configuration from various sources (environment, files, etc.)
// This is a placeholder - in a real implementation, this would integrate with
// viper or similar configuration management libraries
func LoadConfig(opts ...ConfigOption) (Config, error) {
	config := DefaultConfig()

	for _, opt := range opts {
		opt(&config)
	}

	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// LoadFromMainConfig loads monitoring configuration from the main config package
func LoadFromMainConfig(mainConfig *iface.Config) (Config, error) {
	// This is a placeholder for integrating with the main config package
	// In a real implementation, this would extract monitoring-specific config
	// from the main config and validate it

	monitoringConfig := DefaultConfig()

	// Extract service name from main config if available
	if mainConfig != nil {
		// Could extract monitoring-specific settings from main config here
		// For now, just use defaults
		monitoringConfig.ServiceName = "beluga-ai-service"
	}

	if err := monitoringConfig.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid monitoring configuration: %w", err)
	}

	return monitoringConfig, nil
}

// ValidateWithMainConfig validates monitoring config against main config
func (c *Config) ValidateWithMainConfig(mainConfig *iface.Config) error {
	// First validate the monitoring config itself
	if err := c.Validate(); err != nil {
		return fmt.Errorf("monitoring config validation failed: %w", err)
	}

	// Then validate integration with main config
	if mainConfig != nil {
		// Add cross-validation logic here
		// For example, check if service names match, etc.
		if c.ServiceName == "" {
			return fmt.Errorf("service name cannot be empty when using main config")
		}
	}

	return nil
}
