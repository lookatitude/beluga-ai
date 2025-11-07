# Data Model

## Entities

### 1. TestFile
Represents a Go test file (`*_test.go`) containing test functions and test utilities.

**Fields**:
- `Path` (string): Absolute path to the test file
- `Package` (string): Package name
- `Functions` ([]TestFunction): List of test functions in the file
- `HasIntegrationSuffix` (bool): True if file matches `*_integration_test.go` pattern
- `AST` (*ast.File): Parsed AST representation

**Validation Rules**:
- Path must be absolute and exist
- Package name must be valid Go identifier
- Must contain at least one test function

### 2. TestFunction
Represents an individual test function (`Test*`, `Benchmark*`, `Fuzz*`) that may contain long-running operations.

**Fields**:
- `Name` (string): Function name
- `Type` (TestType): Unit, Integration, or Load
- `File` (*TestFile): Reference to containing file
- `LineStart` (int): Starting line number
- `LineEnd` (int): Ending line number
- `Issues` ([]PerformanceIssue): List of identified issues
- `HasTimeout` (bool): Whether function has timeout mechanism
- `TimeoutDuration` (time.Duration): Timeout duration if present
- `ExecutionTime` (time.Duration): Measured execution time (if available)
- `UsesActualImplementation` (bool): Whether function uses real implementations instead of mocks
- `UsesMocks` (bool): Whether function uses mocks
- `MixedUsage` (bool): Whether function mixes mocks and real implementations

**Validation Rules**:
- Name must start with `Test`, `Benchmark`, or `Fuzz`
- Line numbers must be valid (start < end)
- Type must be determined via detection logic

**State Transitions**:
- `Analyzed` → Issues identified
- `Fixed` → Fixes applied and validated
- `Failed` → Fix validation failed, rolled back

### 3. TestType
Enumeration of test function types.

**Values**:
- `Unit`: Standard unit test (`Test*` in `*_test.go`)
- `Integration`: Integration test (`Test*` in `*_integration_test.go` or with long timeout)
- `Load`: Performance/load test (`Benchmark*` or `RunLoadTest` pattern)

**Detection Logic**:
- Unit: `Test*` in `*_test.go` (not `*_integration_test.go`)
- Integration: `Test*` in `*_integration_test.go` OR has `context.WithTimeout` > 10s
- Load: `Benchmark*` OR contains `RunLoadTest` pattern

### 4. PerformanceIssue
Represents an identified problem in a test function, including location, type, severity, and context.

**Fields**:
- `Type` (IssueType): Category of issue
- `Severity` (Severity): Low, Medium, High, Critical
- `Location` (Location): File path, function name, line numbers
- `Description` (string): Human-readable description
- `Context` (map[string]interface{}): Additional context (iteration count, timeout duration, etc.)
- `Fixable` (bool): Whether issue can be automatically fixed
- `Fix` (*Fix): Proposed or applied fix

**Validation Rules**:
- Type must be valid IssueType
- Severity must be valid Severity
- Location must have valid file path and line numbers
- Description must not be empty

### 5. IssueType
Enumeration of performance issue categories.

**Values**:
- `InfiniteLoop`: `for { }` without proper exit conditions
- `MissingTimeout`: No timeout mechanism present
- `LargeIteration`: Hardcoded large iteration count exceeding threshold
- `HighConcurrency`: High concurrency level in load tests
- `SleepDelay`: `time.Sleep` calls accumulating to significant duration
- `ActualImplementationUsage`: Uses real implementations instead of mocks (unit tests)
- `MixedMockRealUsage`: Mixes mocks and real implementations (unit tests)
- `MissingMock`: Missing mock implementation for used interface
- `BenchmarkHelperUsage`: Benchmark helper called during regular test
- `Other`: Other performance-related patterns

### 6. Severity
Enumeration of issue severity levels.

**Values**:
- `Low`: Minor issue, unlikely to cause test failures
- `Medium`: Moderate issue, may cause occasional failures
- `High`: Significant issue, likely to cause test failures
- `Critical`: Critical issue, will cause test failures or hangs

**Severity Assignment**:
- InfiniteLoop: Critical
- MissingTimeout: High (unit tests), Medium (integration tests)
- LargeIteration: Medium (simple ops), High (complex ops)
- ActualImplementationUsage: High (unit tests), Low (integration tests)
- MixedMockRealUsage: Medium (unit tests), Low (integration tests)
- MissingMock: Medium
- SleepDelay: Low (if <100ms), Medium (if >=100ms)

### 7. Location
Represents the location of an issue in source code.

**Fields**:
- `Package` (string): Package name
- `File` (string): File path (relative to repo root)
- `Function` (string): Function name
- `LineStart` (int): Starting line number
- `LineEnd` (int): Ending line number (may equal LineStart for single-line issues)
- `ColumnStart` (int): Starting column (optional)
- `ColumnEnd` (int): Ending column (optional)

**Validation Rules**:
- Package must be valid Go identifier
- File path must be relative to repo root
- Function name must not be empty
- Line numbers must be positive (start <= end)

### 8. Fix
Represents a proposed or applied fix for a performance issue.

**Fields**:
- `Issue` (*PerformanceIssue): Reference to the issue being fixed
- `Type` (FixType): Type of fix being applied
- `Changes` ([]CodeChange): List of code modifications
- `Status` (FixStatus): Proposed, Applied, Validated, Failed, RolledBack
- `ValidationResult` (*ValidationResult): Result of validation (if validated)
- `BackupPath` (string): Path to backup file (if applied)

**Validation Rules**:
- Issue must not be nil
- Type must be valid FixType
- Changes must not be empty
- Status transitions must be valid

**State Transitions**:
- `Proposed` → User approval or auto-apply
- `Applied` → Code modified, backup created
- `Validated` → Tests pass, execution time improved
- `Failed` → Validation failed, rollback triggered
- `RolledBack` → Changes reverted

### 9. FixType
Enumeration of fix types.

**Values**:
- `AddTimeout`: Add timeout mechanism to test function
- `ReduceIterations`: Reduce excessive iteration counts
- `OptimizeSleep`: Reduce or remove sleep durations
- `AddLoopExit`: Add proper exit condition to infinite loop
- `ReplaceWithMock`: Replace actual implementation with mock
- `CreateMock`: Create missing mock implementation
- `UpdateTestFile`: Update test file to use new mock

### 10. CodeChange
Represents a single code modification.

**Fields**:
- `File` (string): File path (relative to repo root)
- `LineStart` (int): Starting line number
- `LineEnd` (int): Ending line number
- `OldCode` (string): Original code to be replaced
- `NewCode` (string): New code to replace with
- `Description` (string): Human-readable description of change

**Validation Rules**:
- File path must be valid
- Line numbers must be positive (start <= end)
- OldCode and NewCode must not both be empty
- Description must not be empty

### 11. ValidationResult
Represents the result of fix validation.

**Fields**:
- `Fix` (*Fix): Reference to validated fix
- `InterfaceCompatible` (bool): Whether interface compatibility check passed
- `TestsPass` (bool): Whether affected tests pass
- `ExecutionTimeImproved` (bool): Whether execution time improved
- `OriginalExecutionTime` (time.Duration): Execution time before fix
- `NewExecutionTime` (time.Duration): Execution time after fix
- `Errors` ([]error): List of validation errors (if failed)
- `TestOutput` (string): Output from test execution

**Validation Rules**:
- Fix must not be nil
- If validation failed, Errors must not be empty
- Execution times must be non-negative

### 12. MockImplementation
Represents a mock implementation following the established pattern.

**Fields**:
- `ComponentName` (string): Name of component being mocked
- `InterfaceName` (string): Name of interface being implemented
- `Package` (string): Package where mock should be created
- `FilePath` (string): File path for mock (usually `test_utils.go`)
- `Code` (string): Generated mock code
- `InterfaceMethods` ([]MethodSignature): List of interface methods
- `Status` (MockStatus): Template, Complete, Validated
- `RequiresManualCompletion` (bool): Whether mock needs manual completion

**Validation Rules**:
- ComponentName must not be empty
- InterfaceName must not be empty
- Package must be valid Go identifier
- FilePath must be valid
- Code must be valid Go code (syntax check)
- InterfaceMethods must match actual interface

### 13. MethodSignature
Represents a method signature from an interface.

**Fields**:
- `Name` (string): Method name
- `Parameters` ([]Parameter): List of parameters
- `Returns` ([]Return): List of return values
- `Receiver` (string): Receiver type (if method, not function)

**Validation Rules**:
- Name must be valid Go identifier
- Parameters and Returns must be valid Go types

### 14. AnalysisReport
Represents the aggregated results of the analysis, including summary statistics and categorized issues.

**Fields**:
- `PackagesAnalyzed` (int): Number of packages analyzed
- `FilesAnalyzed` (int): Number of test files analyzed
- `FunctionsAnalyzed` (int): Number of test functions analyzed
- `IssuesFound` (int): Total number of issues found
- `IssuesByType` (map[IssueType]int): Count of issues by type
- `IssuesBySeverity` (map[Severity]int): Count of issues by severity
- `IssuesByPackage` (map[string]int): Count of issues by package
- `FixesApplied` (int): Number of fixes successfully applied
- `FixesFailed` (int): Number of fixes that failed validation
- `ExecutionTime` (time.Duration): Total analysis execution time
- `GeneratedAt` (time.Time): Timestamp of report generation

**Validation Rules**:
- All counts must be non-negative
- ExecutionTime must be non-negative
- GeneratedAt must be valid timestamp

## Relationships

- **TestFile** has many **TestFunction** (1:N)
- **TestFunction** has many **PerformanceIssue** (1:N)
- **PerformanceIssue** has one **Fix** (1:1, optional)
- **Fix** has many **CodeChange** (1:N)
- **Fix** has one **ValidationResult** (1:1, optional)
- **PerformanceIssue** may require **MockImplementation** (1:1, optional)
- **MockImplementation** implements **Interface** (N:1)
- **AnalysisReport** aggregates all **PerformanceIssue** (1:N)

## State Machines

### TestFunction State Machine
```
[Initial] → [Analyzed] → [Fixed] → [Validated]
                ↓            ↓
            [Failed]    [RolledBack]
```

### Fix State Machine
```
[Proposed] → [Applied] → [Validated]
                ↓
            [Failed] → [RolledBack]
```

### MockImplementation State Machine
```
[Template] → [Complete] → [Validated]
```

## Validation Rules Summary

1. All file paths must be valid and exist (or will be created)
2. All line numbers must be positive and within file bounds
3. All package names must be valid Go identifiers
4. All function names must follow Go naming conventions
5. All code changes must result in valid Go syntax
6. All fixes must pass dual validation before being considered successful
7. All mock implementations must implement required interfaces
8. All execution times must be non-negative
9. All counts and statistics must be non-negative
10. All timestamps must be valid

