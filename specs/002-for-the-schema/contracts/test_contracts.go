// Package contracts defines test contracts that must be implemented for schema package compliance.
// These contracts specify the exact testing requirements from the constitutional standards.

package contracts

import (
	"time"
)

// BenchmarkTestContract defines the required benchmark tests for the schema package.
type BenchmarkTestContract struct {
	// Required benchmark functions that must be implemented
	RequiredBenchmarks []BenchmarkSpec

	// Performance targets that benchmarks must validate
	PerformanceTargets map[string]PerformanceTarget

	// Memory allocation limits for operations
	AllocationLimits map[string]int
}

// BenchmarkSpec defines a specific benchmark test that must be implemented.
type BenchmarkSpec struct {
	FunctionName  string        // Name of benchmark function (BenchmarkXxx)
	OperationType string        // Type of operation being benchmarked
	Target        time.Duration // Performance target
	Description   string        // Description of what is being measured
}

// PerformanceTarget defines performance expectations for operations.
type PerformanceTarget struct {
	MaxDuration       time.Duration // Maximum allowed execution time
	MaxAllocations    int           // Maximum heap allocations
	MaxBytesAllocated int64         // Maximum bytes allocated
	MinOpsPerSecond   int64         // Minimum operations per second
}

// MockTestContract defines the required mock implementations and tests.
type MockTestContract struct {
	// Required mock interfaces that must be implemented
	RequiredMocks []MockSpec

	// Mock generation requirements
	GenerationRequirements MockGenerationSpec

	// Mock behavior validation requirements
	BehaviorValidation []MockBehaviorSpec
}

// MockSpec defines a specific mock that must be implemented.
type MockSpec struct {
	InterfaceName    string   // Name of interface to mock
	PackagePath      string   // Package path where interface is defined
	MockName         string   // Name of mock implementation
	RequiredMethods  []string // Methods that must be mocked
	IsGenerated      bool     // Whether mock should be auto-generated
	GenerationSource string   // Source file for generation
}

// MockGenerationSpec defines requirements for automated mock generation.
type MockGenerationSpec struct {
	UseCodeGeneration  bool     // Whether to use automated generation
	GenerationTool     string   // Tool to use (e.g., "mockery")
	OutputDirectory    string   // Where to place generated mocks
	ConfigFile         string   // Configuration file for generation
	GenerateDirectives []string // Go generate directives to include
}

// MockBehaviorSpec defines behavioral requirements for mocks.
type MockBehaviorSpec struct {
	MockName             string                 // Name of mock
	RequiredBehaviors    map[string]interface{} // Expected behaviors
	ConfigurableOptions  []string               // Options that must be configurable
	ThreadSafetyRequired bool                   // Whether mock must be thread-safe
}

// IntegrationTestContract defines the required integration tests.
type IntegrationTestContract struct {
	// Required integration test categories
	RequiredTestCategories []IntegrationTestCategory

	// Cross-package interaction tests
	CrossPackageTests []CrossPackageTestSpec

	// End-to-end workflow tests
	WorkflowTests []WorkflowTestSpec
}

// IntegrationTestCategory defines a category of integration tests.
type IntegrationTestCategory struct {
	CategoryName  string   // Name of test category
	Description   string   // Description of what is being tested
	RequiredTests []string // Specific tests that must be implemented
	TestDirectory string   // Directory where tests should be located
	MinCoverage   float64  // Minimum coverage required for category
}

// CrossPackageTestSpec defines tests for cross-package interactions.
type CrossPackageTestSpec struct {
	TestName         string   // Name of integration test
	PackagesInvolved []string // Packages that are tested together
	TestScenario     string   // Description of scenario being tested
	ExpectedBehavior string   // Expected behavior description
}

// WorkflowTestSpec defines end-to-end workflow tests.
type WorkflowTestSpec struct {
	WorkflowName     string            // Name of workflow being tested
	Steps            []WorkflowStep    // Individual steps in the workflow
	ValidationPoints []ValidationPoint // Points where validation occurs
	ErrorScenarios   []ErrorScenario   // Error scenarios to test
}

// WorkflowStep defines a single step in a workflow test.
type WorkflowStep struct {
	StepName  string                 // Name of the step
	Action    string                 // Action to perform
	InputData map[string]interface{} // Input data for the step
	Expected  map[string]interface{} // Expected outputs/state
}

// ValidationPoint defines a point in workflow where validation occurs.
type ValidationPoint struct {
	PointName string                 // Name of validation point
	CheckType string                 // Type of check to perform
	Criteria  map[string]interface{} // Validation criteria
}

// ErrorScenario defines an error scenario to test in workflows.
type ErrorScenario struct {
	ScenarioName    string                 // Name of error scenario
	TriggerAction   string                 // Action that triggers the error
	ExpectedError   string                 // Expected error type/message
	RecoveryAction  string                 // How system should recover
	ValidationCheck map[string]interface{} // How to validate error handling
}

// AdvancedTestContract defines requirements for advanced testing patterns.
type AdvancedTestContract struct {
	// Table-driven test requirements
	TableDrivenTests []TableTestSpec

	// Concurrency test requirements
	ConcurrencyTests []ConcurrencyTestSpec

	// Error handling test requirements
	ErrorHandlingTests []ErrorTestSpec

	// Edge case test requirements
	EdgeCaseTests []EdgeCaseSpec
}

// TableTestSpec defines requirements for table-driven tests.
type TableTestSpec struct {
	TestFunctionName string          // Name of test function
	TableName        string          // Name of test table/cases
	TestCases        []TableTestCase // Individual test cases
	ValidationLogic  string          // Description of validation logic
}

// TableTestCase defines a single test case in a table-driven test.
type TableTestCase struct {
	CaseName     string                 // Name of test case
	Input        map[string]interface{} // Input parameters
	Expected     map[string]interface{} // Expected results
	ExpectError  bool                   // Whether error is expected
	ErrorPattern string                 // Expected error pattern (if applicable)
}

// ConcurrencyTestSpec defines requirements for concurrency testing.
type ConcurrencyTestSpec struct {
	TestName         string   // Name of concurrency test
	ConcurrentOps    int      // Number of concurrent operations
	OperationType    string   // Type of operation being tested concurrently
	DurationSeconds  int      // How long to run concurrent test
	ValidationChecks []string // What to validate after concurrent execution
}

// ErrorTestSpec defines requirements for error handling tests.
type ErrorTestSpec struct {
	TestName      string   // Name of error test
	ErrorType     string   // Type of error to test
	TriggerMethod string   // How to trigger the error
	ErrorCode     string   // Expected error code (Op/Err/Code pattern)
	ErrorMessage  string   // Expected error message pattern
	Recovery      []string // Steps for error recovery testing
}

// EdgeCaseSpec defines requirements for edge case testing.
type EdgeCaseSpec struct {
	CaseName       string                 // Name of edge case
	Scenario       string                 // Description of edge case scenario
	InputData      map[string]interface{} // Edge case input data
	ExpectedResult string                 // Expected behavior
	BoundaryType   string                 // Type of boundary (min, max, null, empty, etc.)
}

// DefaultSchemaTestContracts returns the standard test contracts for the schema package.
func DefaultSchemaTestContracts() map[string]interface{} {
	return map[string]interface{}{
		"benchmarks": BenchmarkTestContract{
			RequiredBenchmarks: []BenchmarkSpec{
				{
					FunctionName:  "BenchmarkNewHumanMessage",
					OperationType: "message_creation",
					Target:        1 * time.Millisecond,
					Description:   "Message creation performance",
				},
				{
					FunctionName:  "BenchmarkMessageValidation",
					OperationType: "validation",
					Target:        500 * time.Microsecond,
					Description:   "Message validation performance",
				},
				{
					FunctionName:  "BenchmarkFactoryFunctions",
					OperationType: "factory_creation",
					Target:        100 * time.Microsecond,
					Description:   "Factory function performance",
				},
				{
					FunctionName:  "BenchmarkConcurrentMessageCreation",
					OperationType: "concurrent_operations",
					Target:        2 * time.Millisecond, // Allow degradation under load
					Description:   "Concurrent message creation performance",
				},
			},
			PerformanceTargets: map[string]PerformanceTarget{
				"message_creation": {
					MaxDuration:       1 * time.Millisecond,
					MaxAllocations:    5,
					MaxBytesAllocated: 1024,
					MinOpsPerSecond:   100000,
				},
				"validation": {
					MaxDuration:       500 * time.Microsecond,
					MaxAllocations:    3,
					MaxBytesAllocated: 512,
					MinOpsPerSecond:   200000,
				},
				"factory_creation": {
					MaxDuration:       100 * time.Microsecond,
					MaxAllocations:    2,
					MaxBytesAllocated: 256,
					MinOpsPerSecond:   1000000,
				},
			},
		},
		"mocks": MockTestContract{
			RequiredMocks: []MockSpec{
				{
					InterfaceName:    "Message",
					PackagePath:      "github.com/lookatitude/beluga-ai/pkg/schema/iface",
					MockName:         "MockMessage",
					RequiredMethods:  []string{"GetType", "GetContent", "ToolCalls", "AdditionalArgs"},
					IsGenerated:      true,
					GenerationSource: "iface/message.go",
				},
				{
					InterfaceName:    "ChatHistory",
					PackagePath:      "github.com/lookatitude/beluga-ai/pkg/schema/iface",
					MockName:         "MockChatHistory",
					RequiredMethods:  []string{"AddMessage", "AddUserMessage", "AddAIMessage", "Messages", "Clear"},
					IsGenerated:      true,
					GenerationSource: "iface/message.go",
				},
			},
			GenerationRequirements: MockGenerationSpec{
				UseCodeGeneration: true,
				GenerationTool:    "mockery",
				OutputDirectory:   "internal/mock",
				ConfigFile:        ".mockery.yaml",
				GenerateDirectives: []string{
					"//go:generate mockery --name=Message --output=internal/mock --outpkg=mock",
					"//go:generate mockery --name=ChatHistory --output=internal/mock --outpkg=mock",
				},
			},
		},
		"integration": IntegrationTestContract{
			RequiredTestCategories: []IntegrationTestCategory{
				{
					CategoryName:  "cross_package_message_flow",
					Description:   "Test message passing between schema and other packages",
					RequiredTests: []string{"TestMessageFlowToLLMPackage", "TestMessageFlowToAgentPackage"},
					TestDirectory: "tests/integration",
					MinCoverage:   90.0,
				},
				{
					CategoryName:  "configuration_integration",
					Description:   "Test configuration loading and validation across packages",
					RequiredTests: []string{"TestConfigurationLoading", "TestConfigurationValidation"},
					TestDirectory: "tests/integration",
					MinCoverage:   95.0,
				},
			},
		},
		"advanced": AdvancedTestContract{
			TableDrivenTests: []TableTestSpec{
				{
					TestFunctionName: "TestMessageCreation",
					TableName:        "messageCreationTests",
					ValidationLogic:  "Validate message type, content, and additional properties",
				},
				{
					TestFunctionName: "TestConfigurationValidation",
					TableName:        "configValidationTests",
					ValidationLogic:  "Validate configuration parsing and error handling",
				},
			},
			ConcurrencyTests: []ConcurrencyTestSpec{
				{
					TestName:         "TestConcurrentMessageCreation",
					ConcurrentOps:    100,
					OperationType:    "message_creation",
					DurationSeconds:  5,
					ValidationChecks: []string{"no_race_conditions", "consistent_results", "no_memory_leaks"},
				},
			},
			ErrorHandlingTests: []ErrorTestSpec{
				{
					TestName:      "TestInvalidMessageValidation",
					ErrorType:     "validation_error",
					TriggerMethod: "create_invalid_message",
					ErrorCode:     "ErrCodeInvalidMessage",
					ErrorMessage:  "invalid message format",
				},
			},
		},
	}
}
