# Data Model: V2 Framework Alignment

**Feature**: V2 Framework Alignment  
**Date**: 2025-01-27  
**Status**: Complete

## Overview

This document defines the key entities and data structures for v2 framework alignment. These entities represent the compliance status, provider extensions, observability integration, and multimodal capabilities across packages.

---

## Entity 1: PackageCompliance

**Purpose**: Represents the compliance status of a package with v2 framework standards.

### Attributes

- **PackageName** (string, required): Name of the package (e.g., "llms", "embeddings", "voice")
- **ComplianceStatus** (enum, required): Overall compliance status
  - Values: `FULL`, `PARTIAL`, `NON_COMPLIANT`
- **OTELObservability** (struct, required): OTEL integration status
  - **HasMetrics** (bool): Whether metrics.go exists and is complete
  - **HasTracing** (bool): Whether tracing is implemented in public methods
  - **HasLogging** (bool): Whether structured logging with OTEL is present
  - **Status** (enum): `COMPLETE`, `PARTIAL`, `MISSING`
- **PackageStructure** (struct, required): Structure compliance status
  - **HasIface** (bool): Whether iface/ directory exists
  - **HasInternal** (bool): Whether internal/ directory exists (if needed)
  - **HasProviders** (bool): Whether providers/ directory exists (if multi-provider)
  - **HasConfig** (bool): Whether config.go exists
  - **HasMetrics** (bool): Whether metrics.go exists
  - **HasErrors** (bool): Whether errors.go exists
  - **HasTestUtils** (bool): Whether test_utils.go exists
  - **HasAdvancedTest** (bool): Whether advanced_test.go exists
  - **HasREADME** (bool): Whether README.md exists
  - **Status** (enum): `COMPLETE`, `PARTIAL`, `NON_COMPLIANT`
- **TestingCoverage** (struct, required): Testing compliance
  - **HasTestUtils** (bool): Whether test_utils.go exists
  - **HasAdvancedTest** (bool): Whether advanced_test.go exists
  - **HasBenchmarks** (bool): Whether benchmarks exist (for performance-critical packages)
  - **CoveragePercentage** (float): Test coverage percentage
  - **Status** (enum): `COMPLETE`, `PARTIAL`, `INSUFFICIENT`
- **ProviderSupport** (struct, optional): Provider support status (for multi-provider packages)
  - **HasRegistry** (bool): Whether global registry exists
  - **ProviderCount** (int): Number of providers available
  - **MissingProviders** ([]string): List of high-demand providers not yet supported
  - **Status** (enum): `COMPLETE`, `PARTIAL`, `INSUFFICIENT`
- **MultimodalSupport** (struct, optional): Multimodal capability status
  - **HasImageSupport** (bool): Whether image processing is supported
  - **HasVoiceSupport** (bool): Whether voice/audio processing is supported
  - **HasVideoSupport** (bool): Whether video processing is supported
  - **Status** (enum): `COMPLETE`, `PARTIAL`, `NOT_SUPPORTED`
- **AlignmentPriorities** ([]string, required): List of alignment priorities for this package
  - Values: `OTEL_OBSERVABILITY`, `PROVIDER_EXPANSION`, `STRUCTURE_STANDARDIZATION`, `MULTIMODAL_CAPABILITIES`, `TESTING_ENHANCEMENT`
- **CompletionStatus** (enum, required): Overall completion status
  - Values: `NOT_STARTED`, `IN_PROGRESS`, `COMPLETE`, `VERIFIED`

### Relationships

- **RelatedTo**: PackageStructure, ObservabilityIntegration, ProviderRegistry
- **Contains**: Multiple ProviderExtension entities (for multi-provider packages)
- **Has**: One ObservabilityIntegration entity

### Validation Rules

- PackageName must match existing package name
- ComplianceStatus must be consistent with component statuses
- If PackageStructure.Status is COMPLETE, all required files must exist
- If OTELObservability.Status is COMPLETE, all three components (metrics, tracing, logging) must be present

---

## Entity 2: ProviderExtension

**Purpose**: Represents a new provider addition to an existing multi-provider package.

### Attributes

- **ProviderName** (string, required): Name of the provider (e.g., "grok", "gemini", "openai-multimodal")
- **PackageName** (string, required): Package where provider is added (e.g., "llms", "embeddings")
- **ProviderType** (enum, required): Type of provider
  - Values: `LLM`, `EMBEDDING`, `VECTOR_STORE`, `STT`, `TTS`, `S2S`
- **Capabilities** ([]string, required): List of provider capabilities
  - Values: `TEXT`, `MULTIMODAL`, `STREAMING`, `FUNCTION_CALLING`, `VISION`, `AUDIO`
- **SDKInfo** (struct, required): SDK and API information
  - **SDKName** (string): Name of the SDK/library used
  - **SDKVersion** (string): Version of the SDK
  - **APIEndpoint** (string, optional): API endpoint URL
  - **AuthenticationType** (enum): `API_KEY`, `OAUTH`, `AWS_SIGV4`, `NONE`
- **Configuration** (struct, required): Provider configuration structure
  - **RequiredFields** ([]string): List of required configuration fields
  - **OptionalFields** ([]string): List of optional configuration fields
  - **DefaultValues** (map[string]interface{}): Default configuration values
- **RegistrationInfo** (struct, required): Provider registration details
  - **RegistryName** (string): Name used in global registry
  - **FactoryFunction** (string): Name of factory function
  - **AutoRegister** (bool): Whether provider auto-registers via init()
- **TestCoverage** (struct, required): Test coverage information
  - **HasUnitTests** (bool): Whether unit tests exist
  - **HasIntegrationTests** (bool): Whether integration tests exist
  - **HasMock** (bool): Whether mock implementation exists in test_utils.go
  - **CoveragePercentage** (float): Test coverage percentage
- **Status** (enum, required): Implementation status
  - Values: `PLANNED`, `IN_PROGRESS`, `COMPLETE`, `VERIFIED`

### Relationships

- **BelongsTo**: One PackageCompliance entity
- **RegisteredIn**: One ProviderRegistry entity
- **Implements**: Provider interface from package

### Validation Rules

- ProviderName must be unique within PackageName
- RegistryName must be unique across all providers in package
- If Status is COMPLETE, all test coverage requirements must be met
- Configuration must include all required fields for provider type

---

## Entity 3: ObservabilityIntegration

**Purpose**: Represents OTEL metrics, tracing, and logging integration for a package.

### Attributes

- **PackageName** (string, required): Name of the package
- **MetricsIntegration** (struct, required): Metrics integration details
  - **HasMetricsFile** (bool): Whether metrics.go exists
  - **MetricDefinitions** ([]MetricDefinition): List of metric definitions
  - **MetricTypes** ([]string): Types of metrics (counter, histogram, gauge)
  - **Status** (enum): `COMPLETE`, `PARTIAL`, `MISSING`
- **TracingIntegration** (struct, required): Tracing integration details
  - **HasTracing** (bool): Whether tracing is implemented
  - **PublicMethodsTraced** (int): Number of public methods with tracing
  - **TotalPublicMethods** (int): Total number of public methods
  - **CoveragePercentage** (float): Percentage of public methods traced
  - **Status** (enum): `COMPLETE`, `PARTIAL`, `MISSING`
- **LoggingIntegration** (struct, required): Logging integration details
  - **HasStructuredLogging** (bool): Whether structured logging is used
  - **HasOTELContext** (bool): Whether logs include OTEL context (trace IDs)
  - **LogLevels** ([]string): Supported log levels
  - **Status** (enum): `COMPLETE`, `PARTIAL`, `MISSING`
- **PatternCompliance** (struct, required): Compliance with framework patterns
  - **UsesStandardPatterns** (bool): Whether standard OTEL patterns are used
  - **MetricNamingConvention** (string): Metric naming convention used
  - **TracingPattern** (string): Tracing pattern used
  - **Status** (enum): `COMPLIANT`, `NON_COMPLIANT`
- **PerformanceImpact** (struct, optional): Performance impact assessment
  - **BenchmarkResults** (map[string]float64): Benchmark results (operation -> latency)
  - **OverheadPercentage** (float): Estimated overhead percentage
  - **Status** (enum): `ACCEPTABLE`, `NEEDS_OPTIMIZATION`, `NOT_MEASURED`

### Relationships

- **BelongsTo**: One PackageCompliance entity
- **Uses**: OTEL infrastructure from pkg/monitoring

### Validation Rules

- If Status is COMPLETE, all three components (metrics, tracing, logging) must be COMPLETE
- PatternCompliance.Status must be COMPLIANT for v2 alignment
- PerformanceImpact must be measured for performance-critical packages

---

## Entity 4: MultimodalCapability

**Purpose**: Represents multimodal support added to a package (images, audio, video).

### Attributes

- **PackageName** (string, required): Name of the package
- **SupportedDataTypes** ([]string, required): List of supported data types
  - Values: `TEXT`, `IMAGE`, `AUDIO`, `VIDEO`
- **SchemaExtensions** ([]SchemaExtension, optional): Schema type extensions
  - **TypeName** (string): Name of new type (e.g., "ImageMessage")
  - **BaseType** (string): Base type extended (e.g., "Message")
  - **Fields** ([]Field): Additional fields for multimodal data
- **EmbeddingSupport** (struct, optional): Multimodal embedding support (for embeddings package)
  - **SupportsImageEmbedding** (bool): Whether image embedding is supported
  - **SupportsVideoEmbedding** (bool): Whether video embedding is supported
  - **EmbeddingDimensions** (map[string]int): Dimensions for each data type
- **VectorStoreSupport** (struct, optional): Multimodal vector storage (for vectorstores package)
  - **SupportsMultimodalVectors** (bool): Whether multimodal vectors can be stored
  - **SupportsMultimodalSearch** (bool): Whether multimodal search is supported
  - **IndexTypes** ([]string): Types of indexes supported
- **AgentSupport** (struct, optional): Multimodal agent support (for agents package)
  - **SupportsMultimodalInput** (bool): Whether agents can process multimodal inputs
  - **SupportsMultimodalOutput** (bool): Whether agents can generate multimodal outputs
  - **SupportedInputTypes** ([]string): Types of inputs agents can process
- **BackwardCompatibility** (struct, required): Backward compatibility information
  - **TextOnlyWorkflowsSupported** (bool): Whether text-only workflows still work
  - **BreakingChanges** ([]string): List of any breaking changes (should be empty)
  - **MigrationRequired** (bool): Whether migration is required for existing code
- **Status** (enum, required): Implementation status
  - Values: `PLANNED`, `IN_PROGRESS`, `COMPLETE`, `VERIFIED`

### Relationships

- **BelongsTo**: One PackageCompliance entity
- **Extends**: Schema types from schema package
- **CompatibleWith**: Text-only workflows (must maintain compatibility)

### Validation Rules

- BackwardCompatibility.BreakingChanges must be empty (no breaking changes allowed)
- BackwardCompatibility.TextOnlyWorkflowsSupported must be true
- If Status is COMPLETE, all supported data types must have implementations

---

## Entity 5: PackageStructure

**Purpose**: Represents the file and directory structure of a package.

### Attributes

- **PackageName** (string, required): Name of the package
- **DirectoryLayout** (struct, required): Directory structure
  - **HasIface** (bool): Whether iface/ directory exists
  - **HasInternal** (bool): Whether internal/ directory exists
  - **HasProviders** (bool): Whether providers/ directory exists (if multi-provider)
  - **IfaceFiles** ([]string): List of files in iface/ directory
  - **InternalFiles** ([]string): List of files in internal/ directory
  - **ProviderDirectories** ([]string): List of provider subdirectories
- **RequiredFiles** (struct, required): Required file presence
  - **HasConfig** (bool): Whether config.go exists
  - **HasMetrics** (bool): Whether metrics.go exists
  - **HasErrors** (bool): Whether errors.go exists
  - **HasTestUtils** (bool): Whether test_utils.go exists
  - **HasAdvancedTest** (bool): Whether advanced_test.go exists
  - **HasREADME** (bool): Whether README.md exists
  - **HasFactory** (bool): Whether factory.go or registry.go exists (if multi-provider)
- **FileLocations** (map[string]string): Mapping of file types to actual file paths
  - Keys: `config`, `metrics`, `errors`, `test_utils`, `advanced_test`, `readme`, `factory`
  - Values: Actual file paths
- **NonStandardFiles** ([]string): List of files in non-standard locations (should be moved)
- **ComplianceStatus** (enum, required): Structure compliance status
  - Values: `COMPLIANT`, `NON_COMPLIANT`, `PARTIAL`
- **AlignmentActions** ([]string): List of actions needed for alignment
  - Values: `MOVE_FILES`, `CREATE_MISSING_FILES`, `REORGANIZE_DIRECTORIES`

### Relationships

- **BelongsTo**: One PackageCompliance entity
- **Contains**: Files and directories

### Validation Rules

- If ComplianceStatus is COMPLIANT, all required files must exist in correct locations
- NonStandardFiles should be empty for compliant packages
- AlignmentActions should be empty for compliant packages

---

## Entity Relationships Diagram

```
PackageCompliance
├── Has: ObservabilityIntegration (1:1)
├── Has: PackageStructure (1:1)
├── Contains: ProviderExtension (1:N, for multi-provider packages)
└── Has: MultimodalCapability (0:1, for packages with multimodal support)

ProviderExtension
├── BelongsTo: PackageCompliance (N:1)
└── RegisteredIn: ProviderRegistry (N:1)

ObservabilityIntegration
└── BelongsTo: PackageCompliance (1:1)

MultimodalCapability
├── BelongsTo: PackageCompliance (1:1)
└── Extends: Schema types (N:M)

PackageStructure
└── BelongsTo: PackageCompliance (1:1)
```

---

## Data Validation Rules

### Global Validation Rules

1. **Backward Compatibility**: All changes must maintain backward compatibility
   - No breaking API changes
   - No breaking configuration changes
   - Existing code must continue to work

2. **Completeness**: All implementations must be complete
   - No placeholders or TODOs
   - All tests pass
   - All builds pass

3. **Framework Compliance**: All changes must follow framework patterns
   - Package design patterns
   - OTEL observability patterns
   - Testing patterns
   - Error handling patterns

---

## State Transitions

### PackageCompliance.Status

```
NOT_STARTED → IN_PROGRESS → COMPLETE → VERIFIED
                ↓
            (if issues found)
                ↓
            IN_PROGRESS (fix issues)
```

### ProviderExtension.Status

```
PLANNED → IN_PROGRESS → COMPLETE → VERIFIED
            ↓
        (if tests fail)
            ↓
        IN_PROGRESS (fix tests)
```

### MultimodalCapability.Status

```
PLANNED → IN_PROGRESS → COMPLETE → VERIFIED
            ↓
        (if compatibility issues)
            ↓
        IN_PROGRESS (fix compatibility)
```

---

**Status**: Data model complete, ready for implementation.
