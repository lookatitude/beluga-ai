// Package contracts defines the interface contracts that must be upheld by the schema package.
// These contracts serve as specifications for testing and validation.

package contracts

import (
	"context"
	"time"
)

// BenchmarkContract defines the expected performance characteristics for schema operations.
type BenchmarkContract struct {
	// MessageCreationMaxTime defines the maximum time allowed for message creation operations
	MessageCreationMaxTime time.Duration // Must be < 1ms

	// FactoryFunctionMaxTime defines the maximum time allowed for factory function calls
	FactoryFunctionMaxTime time.Duration // Must be < 100μs

	// ValidationMaxTime defines the maximum time allowed for validation operations
	ValidationMaxTime time.Duration // Must be < 500μs

	// MaxMemoryAllocationsPerMessage defines the maximum heap allocations per message
	MaxMemoryAllocationsPerMessage int // Should be minimized

	// ConcurrentOperationsScalingFactor defines expected performance under concurrent load
	ConcurrentOperationsScalingFactor float64 // Should be close to 1.0 (linear scaling)
}

// MockBehaviorContract defines the expected behavior of mock implementations.
type MockBehaviorContract struct {
	// InterfaceCompliance requires all mocks to fully implement their target interfaces
	InterfaceCompliance bool

	// ThreadSafety requires mocks to match thread-safety characteristics of original
	ThreadSafety bool

	// ConfigurableBehavior requires mocks to support functional options for configuration
	ConfigurableBehavior bool

	// CallTracking requires mocks to track invocation counts and parameters
	CallTracking bool

	// AutoGeneration requires generated mocks to stay in sync with interface changes
	AutoGeneration bool
}

// HealthCheckContract defines the expected behavior of health check components.
type HealthCheckContract struct {
	// MaxCheckDuration defines maximum time allowed for health checks
	MaxCheckDuration time.Duration // Must be < 100ms

	// MinCheckInterval defines minimum time between health checks
	MinCheckInterval time.Duration // Should be > 1s to avoid overhead

	// AccurateReporting requires health status to accurately reflect component state
	AccurateReporting bool

	// ActionableErrors requires error messages to be informative and actionable
	ActionableErrors bool

	// NoPerformanceImpact requires health checks to not impact normal operations
	NoPerformanceImpact bool
}

// TestCoverageContract defines the required test coverage characteristics.
type TestCoverageContract struct {
	// MinimumCoverage defines the minimum acceptable code coverage percentage
	MinimumCoverage float64 // Must be 100% for public methods

	// EdgeCaseCoverage requires comprehensive coverage of boundary conditions
	EdgeCaseCoverage bool

	// ErrorScenarioCoverage requires all error conditions to be tested
	ErrorScenarioCoverage bool

	// TableDrivenTests requires use of table-driven test patterns
	TableDrivenTests bool

	// IntegrationTests requires cross-package interaction testing
	IntegrationTests bool

	// ConcurrencyTests requires thread-safety validation
	ConcurrencyTests bool
}

// TracingContract defines the required OTEL tracing behavior.
type TracingContract struct {
	// FactoryFunctionTracing requires all factory functions to create spans
	FactoryFunctionTracing bool

	// ValidationOperationTracing requires validation operations to be traced
	ValidationOperationTracing bool

	// ContextPropagation requires proper context propagation through call chains
	ContextPropagation bool

	// RelevantAttributes requires spans to include meaningful attributes
	RelevantAttributes bool

	// ErrorRecording requires failed operations to be recorded in spans
	ErrorRecording bool

	// MinimalOverhead requires tracing to have minimal performance impact
	MinimalOverhead bool
}

// DocumentationContract defines the required documentation characteristics.
type DocumentationContract struct {
	// ExecutableExamples requires all code examples to be executable and tested
	ExecutableExamples bool

	// CurrentWithImplementation requires docs to stay current with code changes
	CurrentWithImplementation bool

	// ComprehensiveUsageCoverage requires examples covering common and edge cases
	ComprehensiveUsageCoverage bool

	// ActionableMigrationGuides requires complete migration instructions
	ActionableMigrationGuides bool

	// ClearTargetAudience requires documentation to be tailored to specific audiences
	ClearTargetAudience bool
}

// OverallComplianceContract aggregates all individual contract requirements.
type OverallComplianceContract struct {
	Benchmark     BenchmarkContract
	MockBehavior  MockBehaviorContract
	HealthCheck   HealthCheckContract
	TestCoverage  TestCoverageContract
	Tracing       TracingContract
	Documentation DocumentationContract

	// BackwardCompatibility requires zero breaking changes to existing API
	BackwardCompatibility bool

	// ConstitutionalCompliance requires adherence to all constitutional requirements
	ConstitutionalCompliance bool

	// ExtensibilityPreservation requires existing extension patterns to be maintained
	ExtensibilityPreservation bool
}

// DefaultSchemaPackageContracts returns the standard contract requirements for the schema package.
func DefaultSchemaPackageContracts() OverallComplianceContract {
	return OverallComplianceContract{
		Benchmark: BenchmarkContract{
			MessageCreationMaxTime:            1 * time.Millisecond,
			FactoryFunctionMaxTime:            100 * time.Microsecond,
			ValidationMaxTime:                 500 * time.Microsecond,
			MaxMemoryAllocationsPerMessage:    5,   // Minimize heap allocations
			ConcurrentOperationsScalingFactor: 0.9, // Allow 10% degradation under load
		},
		MockBehavior: MockBehaviorContract{
			InterfaceCompliance:  true,
			ThreadSafety:         true,
			ConfigurableBehavior: true,
			CallTracking:         true,
			AutoGeneration:       true,
		},
		HealthCheck: HealthCheckContract{
			MaxCheckDuration:    100 * time.Millisecond,
			MinCheckInterval:    1 * time.Second,
			AccurateReporting:   true,
			ActionableErrors:    true,
			NoPerformanceImpact: true,
		},
		TestCoverage: TestCoverageContract{
			MinimumCoverage:       100.0, // 100% for public methods
			EdgeCaseCoverage:      true,
			ErrorScenarioCoverage: true,
			TableDrivenTests:      true,
			IntegrationTests:      true,
			ConcurrencyTests:      true,
		},
		Tracing: TracingContract{
			FactoryFunctionTracing:     true,
			ValidationOperationTracing: true,
			ContextPropagation:         true,
			RelevantAttributes:         true,
			ErrorRecording:             true,
			MinimalOverhead:            true,
		},
		Documentation: DocumentationContract{
			ExecutableExamples:         true,
			CurrentWithImplementation:  true,
			ComprehensiveUsageCoverage: true,
			ActionableMigrationGuides:  true,
			ClearTargetAudience:        true,
		},
		BackwardCompatibility:     true,
		ConstitutionalCompliance:  true,
		ExtensibilityPreservation: true,
	}
}

// ValidateContract provides a framework for validating contract compliance during testing.
type ContractValidator interface {
	ValidateBenchmarkContract(ctx context.Context, contract BenchmarkContract) error
	ValidateMockBehaviorContract(ctx context.Context, contract MockBehaviorContract) error
	ValidateHealthCheckContract(ctx context.Context, contract HealthCheckContract) error
	ValidateTestCoverageContract(ctx context.Context, contract TestCoverageContract) error
	ValidateTracingContract(ctx context.Context, contract TracingContract) error
	ValidateDocumentationContract(ctx context.Context, contract DocumentationContract) error
	ValidateOverallCompliance(ctx context.Context, contract OverallComplianceContract) error
}
