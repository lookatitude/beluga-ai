# Data Model: Config Package Constitutional Compliance

## Core Entities

### ConfigProviderRegistry
**Purpose**: Global registry for managing configuration provider registration and discovery
**Context**: Enables framework-compliant provider management with thread-safe operations

**Fields**:
- `mu` (sync.RWMutex): Read-write mutex for thread-safe access to registry
- `creators` (map[string]ProviderCreator): Map of provider names to creator functions
- `providers` (map[string]iface.Provider): Map of instantiated providers for caching
- `metadata` (map[string]ProviderMetadata): Provider capability and configuration metadata

**Validation Rules**:
- Provider names must be unique and non-empty
- Creator functions must be non-nil and return valid Provider implementations
- Metadata must include minimum required capability information
- All registry operations must be protected by mutex for thread safety

**State Transitions**:
- `Unregistered` → `Registered` (via RegisterGlobal)
- `Registered` → `Cached` (via first NewProvider call with caching)
- Thread-safe concurrent registration and provider creation

### ProviderCreator
**Purpose**: Function type for creating configuration providers from options
**Context**: Enables dynamic provider instantiation with configuration validation

**Signature**: `func(options ProviderOptions) (iface.Provider, error)`

**Validation Rules**:
- Must validate provider options before creating provider
- Must return error for invalid or missing required options
- Must create provider implementing full iface.Provider interface
- Must handle provider-specific initialization and setup

**Error Handling**: Returns structured ConfigError with operation context and error codes

### ProviderMetadata
**Purpose**: Describes configuration provider capabilities and requirements
**Context**: Enables provider discovery, compatibility checking, and dynamic selection

**Fields**:
- `Name` (string): Unique provider identifier (e.g., "viper", "composite")
- `Description` (string): Human-readable provider description
- `SupportedFormats` ([]string): Supported configuration formats (yaml, json, toml)
- `Capabilities` ([]string): Provider capabilities (file_loading, env_vars, watching)
- `RequiredOptions` ([]string): Required configuration options for provider creation
- `OptionalOptions` ([]string): Optional configuration options with defaults
- `HealthCheckSupported` (bool): Whether provider supports health checking

**Validation Rules**:
- Name must be unique within registry
- SupportedFormats must include at least one format
- Capabilities must use standardized capability identifiers  
- Required/Optional options must follow naming conventions

**Relationships**: Associated with ProviderCreator in registry for provider instantiation

### ConfigMetrics
**Purpose**: OpenTelemetry metrics collection for configuration operations
**Context**: Provides comprehensive observability for all configuration loading and provider operations

**Fields**:
- `meter` (metric.Meter): OpenTelemetry meter for metrics creation
- `tracer` (trace.Tracer): OpenTelemetry tracer for distributed tracing
- `loadCounter` (metric.Int64Counter): Count of configuration load operations by provider/format
- `loadDuration` (metric.Float64Histogram): Configuration load duration distribution
- `errorCounter` (metric.Int64Counter): Error count by error type and provider
- `validationCounter` (metric.Int64Counter): Configuration validation operations
- `providerHealthGauge` (metric.Int64Gauge): Current health status of providers

**Methods**:
- `RecordOperation(ctx, operation, duration, success)`: Record configuration operation metrics
- `NoOpMetrics()`: Return no-op implementation for testing
- `RecordLoad(ctx, provider, format, duration, success)`: Record configuration loading metrics
- `RecordValidation(ctx, provider, success, errorCount)`: Record validation metrics

**Validation Rules**:
- All metric operations must include proper labels (provider, format, operation)
- Duration measurements must be positive
- Error codes must use standardized constants
- Context must be properly propagated for distributed tracing

**Integration**: Used by all Provider implementations for consistent observability across formats

### ConfigError
**Purpose**: Structured error handling following Op/Err/Code pattern for main package
**Context**: Provides standardized error information for programmatic handling of configuration operations

**Fields**:
- `Op` (string): Operation that failed (e.g., "LoadConfig", "ValidateProvider", "RegisterProvider")
- `Err` (error): Underlying error cause
- `Code` (string): Standardized error code for programmatic handling
- `Provider` (string): Provider name where error occurred (if applicable)
- `Format` (string): Configuration format if applicable (yaml, json, toml)
- `Context` (map[string]interface{}): Additional context information for debugging

**Standard Error Codes**:
- `ErrCodeRegistryNotInitialized`: Provider registry not initialized
- `ErrCodeProviderNotFound`: Requested provider not registered
- `ErrCodeProviderCreationFailed`: Provider creation function failed
- `ErrCodeProviderConfigInvalid`: Invalid provider configuration
- `ErrCodeLoadOperationFailed`: Configuration loading operation failed
- `ErrCodeValidationFailed`: Configuration validation failed
- `ErrCodeFormatUnsupported`: Configuration format not supported by provider
- `ErrCodeFileAccessError`: Cannot access configuration file
- `ErrCodeEnvironmentError`: Environment variable processing error

**Validation Rules**:
- Op must specify the failing operation clearly
- Code must use standardized constants
- Error chain must be preserved via Unwrap() method
- Context must not contain sensitive information (API keys, secrets)

**Methods**:
- `Error() string`: Return formatted error message with operation context
- `Unwrap() error`: Return underlying error for chain preservation
- `Is(target error) bool`: Support errors.Is comparison
- `As(target interface{}) bool`: Support errors.As type assertion

### ProviderOptions
**Purpose**: Unified configuration structure for creating configuration providers
**Context**: Supports both shared and provider-specific configuration options

**Fields**:
- `ProviderType` (string): Provider identifier (e.g., "viper", "composite")
- `ConfigName` (string): Base name for configuration files
- `ConfigPaths` ([]string): Search paths for configuration files
- `EnvPrefix` (string): Environment variable prefix for overrides
- `Format` (string): Configuration format preference (yaml, json, toml, auto)
- `EnableWatching` (bool): Enable configuration file watching for hot reload
- `EnableValidation` (bool): Enable configuration validation
- `EnableCaching` (bool): Enable provider result caching
- `CacheTTL` (time.Duration): Cache time-to-live for provider results
- `ProviderSpecific` (map[string]interface{}): Provider-specific options

**Validation Rules**:
- ProviderType must be registered in global registry
- ConfigPaths must contain at least one valid path
- EnvPrefix must follow naming conventions (uppercase, underscore-separated)
- Format must be supported by the specified provider
- Cache TTL must be positive duration if caching enabled

**Functional Options Support**:
- `WithProviderType(string)`: Set provider type
- `WithConfigPaths([]string)`: Set configuration search paths
- `WithEnvPrefix(string)`: Set environment variable prefix
- `WithFormat(string)`: Set preferred configuration format
- `WithValidation(bool)`: Enable/disable validation
- `WithCaching(duration)`: Configure result caching

### ProviderHealthMonitor
**Purpose**: Health monitoring for configuration providers and validation systems
**Context**: Provides operational visibility into configuration system health

**Fields**:
- `providers` (map[string]iface.Provider): Registered providers for health monitoring
- `lastHealthCheck` (map[string]time.Time): Last health check time per provider
- `healthStatus` (map[string]ProviderHealthStatus): Current health status per provider
- `validationStats` (ValidationHealthStats): Validation operation statistics
- `mu` (sync.RWMutex): Thread safety for concurrent health operations

**Methods**:
- `CheckProviderHealth(ctx, providerName)`: Check individual provider health
- `CheckAllProviders(ctx)`: Check health of all registered providers
- `GetHealthSummary()`: Get aggregated health status summary
- `RecordValidationHealth(success, duration, errors)`: Record validation health metrics

**Health Status Fields**:
- `Status` (string): "healthy", "degraded", "unhealthy"
- `LastChecked` (time.Time): Timestamp of last health check
- `ResponseTime` (time.Duration): Provider response time
- `ErrorCount` (int): Recent error count
- `SuccessRate` (float64): Recent success rate percentage
- `LastError` (string): Last error message (if any)

### ConfigValidator
**Purpose**: Enhanced validation system with health monitoring and performance tracking
**Context**: Provides comprehensive validation with operational monitoring

**Fields**:
- `schemaValidator` (schema.Validator): Integration with schema package validation
- `validationRules` (map[string]ValidationRule): Custom validation rules by config type
- `validationStats` (ValidationStats): Statistics for validation operations
- `healthMonitor` (ValidationHealthMonitor): Health monitoring for validation system

**Validation Capabilities**:
- Schema-based validation using schema package
- Custom validation rules for provider-specific configurations
- Cross-field validation for configuration consistency
- Performance monitoring for validation operations

**Health Integration**:
- Validation operation success/failure tracking
- Performance monitoring for validation speed
- Error pattern analysis and reporting
- Integration with overall config health monitoring

## Entity Relationships

```
ConfigProviderRegistry (1) ─── (*) ProviderCreator
│
├── (1) ─── (*) ProviderMetadata
│
├── (manages) ─── (many) iface.Provider
│   ├── (implements) ─── ViperProvider
│   ├── (implements) ─── CompositeProvider  
│   └── (implements) ─── CustomProviders
│
├── (uses) ─── ConfigMetrics
│   ├── (records to) ─── OTEL Metrics
│   └── (traces with) ─── OTEL Tracer
│
├── (monitors) ─── ProviderHealthMonitor
│   └── (tracks) ─── ProviderHealthStatus
│
└── (returns) ─── ConfigError
    └── (follows) ─── Op/Err/Code pattern
```

## Data Flow

### Provider Registration Flow
1. **Registration**: `RegisterGlobal(name, creator)` → stored in registry with metadata
2. **Metadata**: Provider capabilities registered for discovery and validation
3. **Discovery**: `ListProviders()` returns available providers with capabilities and formats

### Configuration Loading Flow
1. **Provider Selection**: Registry lookup or explicit provider specification
2. **Provider Creation**: Creator function invoked with validated options
3. **Configuration Loading**: Provider loads config from source (file, env, etc.)
4. **Validation**: ConfigValidator validates loaded configuration using schema integration
5. **Health Monitoring**: Operation metrics recorded and provider health updated
6. **Result**: Validated configuration returned with comprehensive error handling

### Error Handling Flow
1. **Error Occurrence**: Configuration operation fails at any stage
2. **Context Collection**: Operation context, provider info, format details collected
3. **Error Creation**: ConfigError created with Op/Err/Code pattern
4. **Error Propagation**: Structured error with detailed context returned
5. **Metrics Recording**: Error metrics recorded for monitoring and alerting

## Validation & Constraints

### Cross-Entity Validations
- Provider names in registry must be unique and follow naming conventions
- Provider options validation must use provider-specific metadata requirements
- Error codes must be consistent across all configuration operations
- Health monitoring must integrate with existing OTEL metrics collection

### Performance Constraints
- Registry operations must be optimized for concurrent access (read-heavy workload)
- Configuration loading should target <10ms for typical configurations
- Provider resolution must be efficient <1ms for cached providers
- Health monitoring should add minimal overhead to configuration operations

### Thread Safety Requirements
- ConfigProviderRegistry must support concurrent registration and provider creation
- ConfigMetrics must be safe for concurrent use across multiple goroutines
- ProviderHealthMonitor must handle concurrent health checks gracefully
- All provider implementations must be thread-safe for concurrent configuration loading

## Evolution Considerations

### Backward Compatibility
- All existing provider creation functions maintained as convenience methods
- Existing configuration loading behavior preserved without breaking changes
- Error handling enhanced but not breaking for existing error patterns
- Provider options extended but existing options preserved

### Extensibility Points
- ProviderMetadata extensible for new provider capability types
- ProviderOptions.ProviderSpecific supports custom provider configuration
- ConfigError.Context allows additional error context information
- Health monitoring extensible for new provider types and validation scenarios

### Migration Path
- Existing providers work unchanged, gain registry benefits when updated
- Gradual migration of error handling to constitutional Op/Err/Code pattern
- OTEL integration transparent to existing configuration loading usage
- Provider-specific enhancements possible without breaking registry pattern

## Integration with Existing Architecture

### Schema Package Integration
- Leverage existing schema validation capabilities
- Enhance validation with health monitoring and performance tracking
- Maintain current schema-based validation for configuration structures

### Multi-Provider Architecture Enhancement
- Registry enhances existing Viper and Composite providers
- Provider discovery enables dynamic selection and fallback
- Composite provider integration with registry for enhanced capabilities
- Preserve existing provider-specific functionality and configuration options
