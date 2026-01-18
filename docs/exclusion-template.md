# Test Coverage Exclusion Documentation Template

**Purpose**: Document untestable code paths and unmockable dependencies per package
**Usage**: Copy this template to each package's test_utils.go or create a separate exclusions.md file

## Package: [PACKAGE_NAME]

### Excluded Code Paths

#### 1. Error Handling Paths
- **File**: `[filename].go`
- **Lines**: `[line-range]`
- **Reason**: [Why this path cannot be tested]
- **Example**: 
  ```go
  // Cannot test: OS-level panic recovery
  defer func() {
      if r := recover(); r != nil {
          // Untestable OS panic
      }
  }()
  ```

#### 2. External System Dependencies
- **File**: `[filename].go`
- **Lines**: `[line-range]`
- **Reason**: [Why this dependency cannot be mocked]
- **Example**:
  ```go
  // Cannot mock: Direct OS calls
  if _, err := os.Stat(path); err != nil {
      // Requires actual file system
  }
  ```

#### 3. Race Condition Scenarios
- **File**: `[filename].go`
- **Lines**: `[line-range]`
- **Reason**: [Why race conditions cannot be reliably tested]
- **Example**:
  ```go
  // Cannot reliably test: Timing-dependent race condition
  go func() {
      // Race condition that depends on timing
  }()
  ```

#### 4. Platform-Specific Code
- **File**: `[filename].go`
- **Lines**: `[line-range]`
- **Reason**: [Why platform-specific code cannot be tested on all platforms]
- **Example**:
  ```go
  // Cannot test on all platforms: Windows-specific code
  // +build windows
  ```

#### 5. Third-Party Library Limitations
- **File**: `[filename].go`
- **Lines**: `[line-range]`
- **Reason**: [Why third-party library cannot be mocked]
- **Example**:
  ```go
  // Cannot mock: Third-party library internal implementation
  library.DoSomething() // No mock interface available
  ```

### Excluded Test Scenarios

#### 1. Network Timeout Scenarios
- **Scenario**: Network timeout handling
- **Reason**: Requires actual network delays or complex time manipulation
- **Coverage Impact**: [X]% of code

#### 2. Concurrent Access Patterns
- **Scenario**: Complex concurrent access patterns
- **Reason**: Difficult to reliably reproduce race conditions
- **Coverage Impact**: [X]% of code

#### 3. Resource Exhaustion
- **Scenario**: Memory/CPU exhaustion handling
- **Reason**: Requires system-level resource manipulation
- **Coverage Impact**: [X]% of code

### Mock Limitations

#### 1. Unmockable Dependencies
- **Dependency**: [Dependency name]
- **Reason**: [Why it cannot be mocked]
- **Workaround**: [Alternative testing approach]

#### 2. Partial Mock Support
- **Dependency**: [Dependency name]
- **Limitation**: [What cannot be mocked]
- **Coverage**: [X]% of dependency functionality

### Coverage Metrics

- **Total Lines**: [X]
- **Excluded Lines**: [Y]
- **Testable Lines**: [X - Y]
- **Current Coverage**: [Z]%
- **Target Coverage**: 100% of testable lines

### Review Status

- **Last Reviewed**: [Date]
- **Reviewed By**: [Name]
- **Status**: [Approved/Pending/Needs Review]

## Notes

- All exclusions must be documented with clear reasoning
- Exclusions should be reviewed periodically
- Alternative testing approaches should be considered before excluding code
- Exclusions should be minimal and well-justified

## Example Usage

```go
// Package: pkg/llms
// File: test_utils.go

// EXCLUSION: Error handling for OS-level panics
// Lines: llms.go:45-50
// Reason: Cannot reliably test OS-level panic recovery
// Coverage Impact: 0.1% of code

// EXCLUSION: Network timeout scenarios
// Lines: providers/openai/openai.go:120-125
// Reason: Requires actual network delays or complex time manipulation
// Coverage Impact: 0.5% of code
// Workaround: Test timeout logic separately with mock HTTP client
```
