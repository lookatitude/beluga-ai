# Data Model: ChatModels Package Framework Compliance

## Core Entities

### ChatModelRegistry
**Purpose**: Global registry for managing chat model provider registration and discovery
**Context**: Enables framework-compliant provider management with thread-safe operations

**Fields**:
- `mu` (sync.RWMutex): Read-write mutex for thread-safe access
- `creators` (map[string]ProviderCreator): Map of provider names to creator functions
- `metadata` (map[string]ProviderMetadata): Provider capability and configuration metadata
- `initialized` (map[string]bool): Lazy initialization tracking for providers

**Validation Rules**:
- Provider names must be unique and non-empty
- Creator functions must be non-nil and valid
- Metadata must include minimum required capability information
- All map operations must be protected by mutex

**State Transitions**: 
- `Unregistered` → `Registered` (via RegisterGlobal)
- `Registered` → `Initialized` (via first NewProvider call)
- Thread-safe concurrent registration and creation

### ProviderCreator
**Purpose**: Function type for creating chat model provider instances from configuration
**Context**: Enables dynamic provider instantiation with configuration validation

**Signature**: `func(ctx context.Context, config Config) (ChatModel, error)`

**Validation Rules**:
- Must validate configuration before creating provider
- Must return error for invalid configuration
- Must create provider implementing full ChatModel interface
- Must handle context cancellation appropriately

**Error Handling**: Returns structured errors following Op/Err/Code pattern

### ProviderMetadata  
**Purpose**: Describes provider capabilities and configuration requirements
**Context**: Enables provider discovery and compatibility checking

**Fields**:
- `Name` (string): Unique provider identifier
- `Description` (string): Human-readable provider description
- `Capabilities` ([]string): Supported capabilities (streaming, tool-calling, etc.)
- `RequiredConfig` ([]string): Required configuration keys
- `OptionalConfig` ([]string): Optional configuration keys with defaults
- `SupportedModels` ([]string): List of supported model identifiers

**Validation Rules**:
- Name must be unique within registry
- Capabilities must use standardized capability identifiers
- Configuration keys must follow naming conventions
- Supported models list must be non-empty

**Relationships**: Associated with ProviderCreator in registry

### ChatModelMetrics
**Purpose**: OpenTelemetry metrics collection for chat model operations
**Context**: Provides comprehensive observability for all chat model interactions

**Fields**:
- `meter` (metric.Meter): OpenTelemetry meter for metrics creation
- `tracer` (trace.Tracer): OpenTelemetry tracer for distributed tracing
- `requestCounter` (metric.Int64Counter): Count of chat model requests by provider/model
- `durationHistogram` (metric.Float64Histogram): Request duration distribution
- `errorCounter` (metric.Int64Counter): Error count by error type and provider
- `activeRequestsGauge` (metric.Int64UpDownCounter): Current active requests

**Methods**:
- `RecordOperation(ctx, operation, duration, success)`: Record operation metrics
- `StartSpan(ctx, operation, provider, model)`: Create distributed tracing span
- `RecordError(ctx, operation, errorCode, provider)`: Record error metrics
- `NoOpMetrics()`: Return no-op implementation for testing

**Validation Rules**:
- All metric operations must include proper labels (provider, model, operation)
- Duration measurements must be positive
- Error codes must use standardized constants
- Context must be properly propagated

**Integration**: Used by all ChatModel implementations for consistent observability

### ChatModelError
**Purpose**: Structured error handling following Op/Err/Code pattern
**Context**: Provides standardized error information for programmatic handling

**Fields**:
- `Op` (string): Operation that failed (e.g., "GenerateMessages", "StreamMessages")
- `Err` (error): Underlying error cause  
- `Code` (string): Standardized error code for programmatic handling
- `Provider` (string): Provider name where error occurred
- `Model` (string): Model identifier if applicable
- `Context` (map[string]interface{}): Additional context information

**Standard Error Codes**:
- `ErrCodeProviderUnavailable`: Provider service unavailable
- `ErrCodeConfigInvalid`: Invalid configuration provided
- `ErrCodeAuthenticationFailed`: API key or authentication error
- `ErrCodeRateLimit`: Rate limit exceeded
- `ErrCodeModelUnsupported`: Requested model not supported
- `ErrCodeRequestTimeout`: Request exceeded timeout
- `ErrCodeInvalidInput`: Invalid input messages or options

**Validation Rules**:
- Op must specify the failing operation
- Code must use standardized constants
- Error chain must be preserved via Unwrap()
- Context must not contain sensitive information

**Methods**:
- `Error() string`: Return formatted error message
- `Unwrap() error`: Return underlying error for chain preservation
- `Is(target error) bool`: Support errors.Is comparison
- `As(target interface{}) bool`: Support errors.As type assertion

### ProviderConfig
**Purpose**: Unified configuration structure for chat model providers
**Context**: Supports both shared and provider-specific configuration options

**Fields**:
- `Provider` (string): Provider identifier (e.g., "openai", "anthropic")
- `Model` (string): Model identifier (e.g., "gpt-4", "claude-3-sonnet")
- `APIKey` (string): Provider API key or authentication token
- `BaseURL` (string): Custom API endpoint URL if different from default
- `Timeout` (time.Duration): Request timeout duration
- `MaxRetries` (int): Maximum retry attempts for failed requests
- `Temperature` (float32): Model temperature parameter
- `MaxTokens` (int): Maximum tokens in response
- `ProviderSpecific` (map[string]interface{}): Provider-specific options

**Validation Rules**:
- Provider must be registered in global registry
- Model must be supported by specified provider
- APIKey must be non-empty for providers requiring authentication
- Timeout must be positive duration
- MaxRetries must be non-negative
- Temperature must be between 0.0 and 2.0
- MaxTokens must be positive

**Functional Options Support**:
- `WithProvider(string)`: Set provider identifier
- `WithModel(string)`: Set model identifier  
- `WithAPIKey(string)`: Set authentication key
- `WithTimeout(duration)`: Set request timeout
- `WithTemperature(float32)`: Set temperature parameter

### ChatModelWrapper  
**Purpose**: Wrapper that adds registry compliance to existing ChatModel implementations
**Context**: Enables existing providers to work with registry pattern without modification

**Fields**:
- `underlying` (ChatModel): Original ChatModel implementation
- `providerName` (string): Provider identifier for registry
- `config` (ProviderConfig): Configuration used to create provider
- `metrics` (ChatModelMetrics): Metrics collection instance

**Methods**: Implements ChatModel interface by delegating to underlying implementation while adding:
- Metrics collection for all operations
- Error wrapping with Op/Err/Code pattern  
- Distributed tracing spans
- Registry-compliant behavior

**Validation Rules**:
- Underlying ChatModel must implement complete interface
- Provider name must match registry registration
- Metrics must be properly initialized
- All operations must preserve original behavior while adding observability

**Relationships**: Created by registry when instantiating providers

### ObservabilityContext
**Purpose**: Context information for metrics and tracing operations
**Context**: Provides structured context for observability data collection

**Fields**:
- `OperationID` (string): Unique identifier for operation tracking
- `ProviderName` (string): Provider being used for operation
- `ModelName` (string): Model being used for operation
- `RequestID` (string): Unique request identifier for correlation
- `UserID` (string): User identifier if available (optional)
- `SessionID` (string): Session identifier for conversation tracking
- `StartTime` (time.Time): Operation start timestamp

**Usage**: Embedded in context.Context for operation tracking

**Validation Rules**:
- OperationID must be unique within reasonable time window
- Provider and model names must match registry entries
- Timestamps must be consistent and monotonic
- Optional fields may be empty but must follow format if provided

## Entity Relationships

```
ChatModelRegistry (1) ─── (*) ProviderCreator
│
├── (1) ─── (*) ProviderMetadata
│
└── (creates) ─── ChatModelWrapper
    │
    ├── (wraps) ─── ChatModel (interface)
    │   ├── MessageGenerator
    │   ├── StreamMessageHandler  
    │   ├── ModelInfoProvider
    │   ├── HealthChecker
    │   └── core.Runnable
    │
    ├── (uses) ─── ChatModelMetrics
    │   ├── (records to) ─── OTEL Metrics
    │   └── (traces with) ─── OTEL Tracer
    │
    └── (returns) ─── ChatModelError
        └── (follows) ─── Op/Err/Code pattern
```

## Data Flow

### Provider Registration Flow
1. **Registration**: `RegisterGlobal(name, creator)` → stored in registry with thread-safe access
2. **Metadata**: Provider metadata registered for discovery and validation  
3. **Discovery**: `ListProviders()` returns available providers with capabilities

### Provider Creation Flow  
1. **Request**: `NewProvider(ctx, name, config)` → registry lookup
2. **Validation**: Configuration validated against provider requirements
3. **Creation**: Creator function invoked with validated configuration
4. **Wrapping**: Provider wrapped with metrics and error handling
5. **Return**: ChatModelWrapper returned implementing full ChatModel interface

### Operation Flow
1. **Start**: ChatModel method invoked (Generate, Stream, etc.)
2. **Metrics**: Operation metrics recorded with OTEL integration  
3. **Tracing**: Distributed tracing span created with proper context
4. **Execution**: Underlying provider method invoked with preserved behavior
5. **Completion**: Metrics updated, span finished, errors wrapped if needed

## Validation & Constraints

### Cross-Entity Validations
- Provider names in registry must match metadata provider names
- Configuration validation must use provider-specific metadata requirements
- Error codes must be consistent across all providers
- Metrics labels must follow standardized naming conventions

### Performance Constraints
- Registry operations must be optimized for concurrent access (read-heavy workload)
- Metrics collection should add <5% overhead to operations
- Provider creation should support lazy initialization to avoid startup delays
- Error handling must not significantly impact performance

### Thread Safety Requirements
- ChatModelRegistry must support concurrent registration and provider creation
- ChatModelMetrics must be safe for concurrent use across goroutines
- ChatModelError instances are immutable after creation
- Provider configuration must be copied to avoid shared state mutations

## Evolution Considerations

### Backward Compatibility  
- All existing ChatModel interface methods preserved unchanged
- Existing provider creation functions maintained as convenience methods
- Error behavior enhanced but not breaking for existing error handling
- Configuration options extended but existing options preserved

### Extensibility Points
- ProviderMetadata extensible for new capability types
- ProviderConfig.ProviderSpecific supports custom provider options  
- ChatModelError.Context allows additional error context
- ObservabilityContext extensible for new tracking requirements

### Migration Path
- Existing providers work unchanged, gain registry benefits when updated
- Gradual migration of error handling to Op/Err/Code pattern
- Metrics integration transparent to existing usage
- Provider-specific enhancements possible without breaking registry pattern
