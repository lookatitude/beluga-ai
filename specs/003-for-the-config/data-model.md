# Phase 1: Data Model & Entity Design

**Feature**: Config Package Full Compliance  
**Date**: October 5, 2025  

## Core Entities

### 1. Config (Central Configuration Structure)

**Purpose**: Root configuration structure containing all application settings  
**Responsibilities**: Hold provider configurations, validation rules, default values

```go
type Config struct {
    LLMProviders       []schema.LLMProviderConfig       `mapstructure:"llm_providers" yaml:"llm_providers" validate:"dive"`
    EmbeddingProviders []schema.EmbeddingProviderConfig `mapstructure:"embedding_providers" yaml:"embedding_providers" validate:"dive"`
    VectorStores       []schema.VectorStoreConfig       `mapstructure:"vector_stores" yaml:"vector_stores" validate:"dive"`
    Tools              []ToolConfig                     `mapstructure:"tools" yaml:"tools" validate:"dive"`
    Agents             []schema.AgentConfig             `mapstructure:"agents" yaml:"agents" validate:"dive"`
    
    // Enhanced fields for compliance
    HealthChecks       HealthCheckConfig                `mapstructure:"health_checks" yaml:"health_checks"`
    Observability      ObservabilityConfig              `mapstructure:"observability" yaml:"observability"`
    Migration          MigrationConfig                  `mapstructure:"migration" yaml:"migration"`
}
```

**Validation Rules**:
- All provider arrays must have unique names within their type
- Cross-references between entities must be valid (e.g., agent.llm_provider_name must exist)
- Provider-specific validation delegated to schema package

**State Transitions**: Immutable after validation, new instances created for changes

### 2. Provider (Configuration Source Interface)

**Purpose**: Interface for loading configuration from various sources  
**Responsibilities**: Load, validate, provide health checks, support hot-reload

```go
type Provider interface {
    // Core loading operations
    Load(configStruct interface{}) error
    UnmarshalKey(key string, rawVal interface{}) error
    
    // Value retrieval operations
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    GetFloat64(key string) float64
    GetStringMapString(key string) map[string]string
    IsSet(key string) bool
    
    // Configuration-specific operations
    GetLLMProviderConfig(name string) (schema.LLMProviderConfig, error)
    GetLLMProvidersConfig() ([]schema.LLMProviderConfig, error)
    GetEmbeddingProvidersConfig() ([]schema.EmbeddingProviderConfig, error)
    GetVectorStoresConfig() ([]schema.VectorStoreConfig, error)
    GetAgentConfig(name string) (schema.AgentConfig, error)
    GetAgentsConfig() ([]schema.AgentConfig, error)
    GetToolConfig(name string) (ToolConfig, error)
    GetToolsConfig() ([]ToolConfig, error)
    
    // Validation and defaults
    Validate() error
    SetDefaults() error
}
```

**Enhanced Provider Interface** (for compliance):
```go
type EnhancedProvider interface {
    Provider
    HealthChecker
    HotReloader
    MetricsReporter
}

type HealthChecker interface {
    HealthCheck(ctx context.Context) HealthStatus
}

type HotReloader interface {
    EnableHotReload(ctx context.Context, callback func(*Config) error) error
    DisableHotReload() error
}

type MetricsReporter interface {
    GetMetrics() ProviderMetrics
    ResetMetrics()
}
```

### 3. Registry (Provider Management System)

**Purpose**: Centralized registration, discovery, and lifecycle management of providers  
**Responsibilities**: Thread-safe provider registration, creation, lifecycle management

```go
type ProviderRegistry struct {
    mu           sync.RWMutex
    creators     map[string]ProviderCreatorFunc
    instances    map[string]Provider
    healthStates map[string]HealthStatus
    metrics      *RegistryMetrics
}

type ProviderCreatorFunc func(ctx context.Context, config interface{}) (Provider, error)

type RegistryConfig struct {
    HealthCheckInterval time.Duration `mapstructure:"health_check_interval" default:"30s"`
    MaxProviders        int           `mapstructure:"max_providers" default:"100"`
    EnableMetrics       bool          `mapstructure:"enable_metrics" default:"true"`
}
```

**Operations**:
- RegisterProvider(name string, creator ProviderCreatorFunc) error
- UnregisterProvider(name string) error
- CreateProvider(ctx context.Context, name string, config interface{}) (Provider, error)
- ListProviders() []string
- GetProviderHealth(name string) HealthStatus
- Shutdown(ctx context.Context) error

**State Transitions**:
1. Empty → Registered (providers registered)
2. Registered → Active (providers created and healthy)
3. Active → Degraded (some providers unhealthy)
4. Any State → Shutdown (cleanup initiated)

### 4. Validator (Configuration Validation Engine)

**Purpose**: Comprehensive validation with custom rules and cross-field validation  
**Responsibilities**: Schema validation, custom validators, cross-field checks, error reporting

```go
type ConfigValidator struct {
    validator        *validator.Validate
    customValidators map[string]validator.Func
    crossFieldRules  []CrossFieldRule
    metrics          *ValidationMetrics
}

type ValidationError struct {
    Field    string            `json:"field"`
    Tag      string            `json:"tag"`
    Value    interface{}       `json:"value"`
    Message  string            `json:"message"`
    Context  map[string]string `json:"context,omitempty"`
}

type ValidationErrors []ValidationError

type CrossFieldRule struct {
    Name      string
    Validator func(config *Config) error
    Message   string
}
```

**Custom Validators**:
- API key format validation
- URL format and reachability
- Provider-specific configuration validation
- Resource reference validation (e.g., agent → llm_provider references)

### 5. Metrics (Observability Collection System)

**Purpose**: OTEL-compliant metrics collection for configuration operations  
**Responsibilities**: Record operations, provider performance, system health, custom metrics

```go
type Metrics struct {
    // Core metrics
    configLoadsTotal      metric.Int64Counter
    configLoadDuration    metric.Float64Histogram
    configErrorsTotal     metric.Int64Counter
    validationDuration    metric.Float64Histogram
    validationErrorsTotal metric.Int64Counter
    
    // Enhanced metrics for compliance
    providerHealthChecks    metric.Int64Counter
    registryOperations      metric.Int64Counter
    hotReloadEvents        metric.Int64Counter
    providerOperations     metric.Int64Counter
    cacheMissesTotal       metric.Int64Counter
    
    // Custom metrics support
    customMetrics map[string]metric.Instrument
    metricsMu     sync.RWMutex
}

type ProviderMetrics struct {
    LoadCount       int64         `json:"load_count"`
    LoadDuration    time.Duration `json:"load_duration"`
    ErrorCount      int64         `json:"error_count"`
    LastHealthCheck time.Time     `json:"last_health_check"`
    HealthStatus    string        `json:"health_status"`
}
```

### 6. Loader (Configuration Loading Orchestrator)

**Purpose**: Coordinate providers, validation, and default value application  
**Responsibilities**: Load configuration, apply defaults, validate, handle errors

```go
type ConfigLoader struct {
    options   LoaderOptions
    providers []Provider
    validator *ConfigValidator
    metrics   *Metrics
    tracer    trace.Tracer
}

type LoaderOptions struct {
    ConfigName        string        `mapstructure:"config_name"`
    ConfigPaths       []string      `mapstructure:"config_paths"`
    EnvPrefix         string        `mapstructure:"env_prefix"`
    Validate          bool          `mapstructure:"validate" default:"true"`
    SetDefaults       bool          `mapstructure:"set_defaults" default:"true"`
    EnableHotReload   bool          `mapstructure:"enable_hot_reload" default:"false"`
    LoadTimeout       time.Duration `mapstructure:"load_timeout" default:"30s"`
    ValidationTimeout time.Duration `mapstructure:"validation_timeout" default:"10s"`
    
    // Enhanced options for compliance
    EnableHealthChecks bool          `mapstructure:"enable_health_checks" default:"true"`
    HealthCheckInterval time.Duration `mapstructure:"health_check_interval" default:"30s"`
    RetryAttempts      int           `mapstructure:"retry_attempts" default:"3"`
    RetryDelay         time.Duration `mapstructure:"retry_delay" default:"1s"`
}
```

### 7. HealthChecker (Health Monitoring Interface)

**Purpose**: Health monitoring for configuration providers and system components  
**Responsibilities**: Check provider health, report system status, support recovery

```go
type HealthStatus struct {
    Status      HealthStatusType           `json:"status"`
    Timestamp   time.Time                  `json:"timestamp"`
    Details     map[string]interface{}     `json:"details,omitempty"`
    LastError   string                     `json:"last_error,omitempty"`
    CheckCount  int64                      `json:"check_count"`
    Latency     time.Duration              `json:"latency"`
}

type HealthStatusType string

const (
    HealthStatusHealthy   HealthStatusType = "healthy"
    HealthStatusDegraded  HealthStatusType = "degraded"
    HealthStatusUnhealthy HealthStatusType = "unhealthy"
    HealthStatusUnknown   HealthStatusType = "unknown"
)

type SystemHealth struct {
    OverallStatus HealthStatusType           `json:"overall_status"`
    Timestamp     time.Time                  `json:"timestamp"`
    Providers     map[string]HealthStatus    `json:"providers"`
    Registry      HealthStatus               `json:"registry"`
    Loader        HealthStatus               `json:"loader"`
    Validator     HealthStatus               `json:"validator"`
}
```

## Enhanced Configuration Types

### Health Check Configuration
```go
type HealthCheckConfig struct {
    Enabled             bool          `mapstructure:"enabled" default:"true"`
    Interval            time.Duration `mapstructure:"interval" default:"30s"`
    Timeout             time.Duration `mapstructure:"timeout" default:"5s"`
    FailureThreshold    int           `mapstructure:"failure_threshold" default:"3"`
    SuccessThreshold    int           `mapstructure:"success_threshold" default:"1"`
    EnableRecovery      bool          `mapstructure:"enable_recovery" default:"true"`
    RecoveryInterval    time.Duration `mapstructure:"recovery_interval" default:"60s"`
}
```

### Observability Configuration
```go
type ObservabilityConfig struct {
    Metrics MetricsConfig `mapstructure:"metrics"`
    Tracing TracingConfig `mapstructure:"tracing"`
    Logging LoggingConfig `mapstructure:"logging"`
}

type MetricsConfig struct {
    Enabled         bool   `mapstructure:"enabled" default:"true"`
    Provider        string `mapstructure:"provider" default:"otel"`
    ExportInterval  time.Duration `mapstructure:"export_interval" default:"30s"`
    EnableCustom    bool   `mapstructure:"enable_custom" default:"true"`
}

type TracingConfig struct {
    Enabled     bool    `mapstructure:"enabled" default:"true"`
    Provider    string  `mapstructure:"provider" default:"otel"`
    SampleRate  float64 `mapstructure:"sample_rate" default:"0.1"`
    EnableSpans bool    `mapstructure:"enable_spans" default:"true"`
}

type LoggingConfig struct {
    Level           string `mapstructure:"level" default:"info"`
    Format          string `mapstructure:"format" default:"json"`
    EnableContext   bool   `mapstructure:"enable_context" default:"true"`
    EnableTraceID   bool   `mapstructure:"enable_trace_id" default:"true"`
}
```

### Migration Configuration
```go
type MigrationConfig struct {
    Enabled            bool     `mapstructure:"enabled" default:"false"`
    CurrentVersion     string   `mapstructure:"current_version"`
    TargetVersion      string   `mapstructure:"target_version"`
    MigrationPaths     []string `mapstructure:"migration_paths"`
    BackupBeforeMigration bool  `mapstructure:"backup_before_migration" default:"true"`
    ValidateAfterMigration bool `mapstructure:"validate_after_migration" default:"true"`
}
```

## Entity Relationships

```
Config (1) ←→ (1) Validator
Config (1) ←→ (1..*) Provider
Registry (1) ←→ (1..*) Provider
Loader (1) ←→ (1..*) Provider
Loader (1) ←→ (1) Validator
Loader (1) ←→ (1) Metrics
Provider (1) ←→ (1) HealthChecker
Provider (1) ←→ (1) Metrics
```

## Validation Rules Summary

1. **Config Level**: Cross-field validation, reference integrity
2. **Provider Level**: Source-specific validation (file existence, network connectivity)
3. **Registry Level**: Name uniqueness, creator function validity
4. **Loader Level**: Option validation, timeout boundaries
5. **Health Level**: Status consistency, threshold validation

## State Management

All entities follow immutable patterns where possible, with new instances created for changes rather than in-place mutations. State transitions are logged and traced for observability.
