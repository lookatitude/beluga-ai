# Fixer Interface Contract

## Interface: Fixer

### Purpose
Interface for applying automated fixes to identified performance issues.

### Methods

#### ApplyFix
```go
ApplyFix(ctx context.Context, issue *PerformanceIssue) (*Fix, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation and timeout
- `issue` (*PerformanceIssue): Issue to fix

**Output**:
- `*Fix`: Applied fix with status
- `error`: Error if fix application fails

**Behavior**:
- Determines appropriate fix type based on issue
- Generates code changes
- Creates backup before modification
- Applies changes to source file
- Updates fix status to Applied

**Errors**:
- `ErrFixNotApplicable`: Issue cannot be automatically fixed
- `ErrBackupFailed`: Failed to create backup
- `ErrFileModificationFailed`: Failed to modify file
- `ErrInvalidFix`: Fix is invalid or unsafe

#### ValidateFix
```go
ValidateFix(ctx context.Context, fix *Fix) (*ValidationResult, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation and timeout
- `fix` (*Fix): Fix to validate

**Output**:
- `*ValidationResult`: Validation results
- `error`: Error if validation fails

**Behavior**:
- Performs dual validation:
  1. Interface compatibility check (for mock-related fixes)
  2. Test execution (runs affected tests)
- Verifies tests pass and execution time improves
- Returns validation result with details

**Errors**:
- `ErrValidationTimeout`: Validation exceeded timeout
- `ErrTestsFailed`: Tests failed after fix
- `ErrExecutionTimeWorse`: Execution time did not improve

#### RollbackFix
```go
RollbackFix(ctx context.Context, fix *Fix) error
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `fix` (*Fix): Fix to rollback

**Output**:
- `error`: Error if rollback fails

**Behavior**:
- Restores file from backup
- Updates fix status to RolledBack
- Removes backup file after successful rollback

**Errors**:
- `ErrBackupNotFound`: Backup file not found
- `ErrRollbackFailed`: Failed to restore file

## Interface: MockGenerator

### Purpose
Interface for generating mock implementations following the established pattern.

### Methods

#### GenerateMock
```go
GenerateMock(ctx context.Context, componentName string, interfaceName string, packagePath string) (*MockImplementation, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `componentName` (string): Name of component being mocked
- `interfaceName` (string): Name of interface to implement
- `packagePath` (string): Package path where mock should be created

**Output**:
- `*MockImplementation`: Generated mock implementation
- `error`: Error if generation fails

**Behavior**:
- Analyzes interface definition
- Generates mock following established pattern:
  - `AdvancedMock{ComponentName}` struct with `mock.Mock` embedded
  - Functional options pattern
  - Configurable behavior (errors, delays)
  - Health check support
  - Call counting
- If complex initialization required, generates template with TODOs

**Errors**:
- `ErrInterfaceNotFound`: Interface not found in package
- `ErrInvalidInterface`: Interface is invalid or cannot be mocked
- `ErrGenerationFailed`: Mock generation failed

#### GenerateMockTemplate
```go
GenerateMockTemplate(ctx context.Context, componentName string, interfaceName string, packagePath string, reason string) (*MockImplementation, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `componentName` (string): Name of component being mocked
- `interfaceName` (string): Name of interface to implement
- `packagePath` (string): Package path where mock should be created
- `reason` (string): Reason why template is needed (complex dependencies, etc.)

**Output**:
- `*MockImplementation`: Generated mock template
- `error`: Error if generation fails

**Behavior**:
- Generates mock template with TODOs and placeholders
- Preserves established pattern structure
- Includes comments explaining what needs manual completion
- Marks as requiring manual completion

**Errors**:
- `ErrInterfaceNotFound`: Interface not found
- `ErrTemplateGenerationFailed`: Template generation failed

#### VerifyInterfaceCompatibility
```go
VerifyInterfaceCompatibility(ctx context.Context, mock *MockImplementation, actualInterface string) (bool, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `mock` (*MockImplementation): Generated mock
- `actualInterface` (string): Name of actual interface

**Output**:
- `bool`: True if compatible
- `error`: Error if verification fails

**Behavior**:
- Compares mock interface methods with actual interface
- Verifies method signatures match (name, parameters, return types)
- Returns true if all methods match

**Errors**:
- `ErrInterfaceNotFound`: Interface not found
- `ErrVerificationFailed`: Verification process failed

## Interface: CodeModifier

### Purpose
Interface for safely modifying Go source code.

### Methods

#### CreateBackup
```go
CreateBackup(ctx context.Context, filePath string) (string, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `filePath` (string): Path to file to backup

**Output**:
- `string`: Path to backup file
- `error`: Error if backup fails

**Behavior**:
- Creates backup copy of file
- Returns path to backup file
- Uses timestamp in backup filename

**Errors**:
- `ErrFileNotFound`: File does not exist
- `ErrBackupFailed`: Failed to create backup

#### ApplyCodeChange
```go
ApplyCodeChange(ctx context.Context, change *CodeChange) error
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `change` (*CodeChange): Code change to apply

**Output**:
- `error`: Error if application fails

**Behavior**:
- Reads file content
- Replaces old code with new code at specified location
- Formats code using `go/format`
- Writes modified content back to file
- Validates syntax after modification

**Errors**:
- `ErrFileNotFound`: File does not exist
- `ErrInvalidChange`: Change is invalid (line numbers out of bounds, etc.)
- `ErrSyntaxError`: Modified code has syntax errors
- `ErrWriteFailed`: Failed to write file

#### FormatCode
```go
FormatCode(ctx context.Context, code string) (string, error)
```

**Input**:
- `ctx` (context.Context): Context for cancellation
- `code` (string): Go code to format

**Output**:
- `string`: Formatted code
- `error`: Error if formatting fails

**Behavior**:
- Formats code using `go/format`
- Returns formatted code

**Errors**:
- `ErrSyntaxError`: Code has syntax errors
- `ErrFormatFailed`: Formatting failed

## Data Types

### Fix
```go
type Fix struct {
    Issue            *PerformanceIssue
    Type             FixType
    Changes          []CodeChange
    Status           FixStatus
    ValidationResult *ValidationResult
    BackupPath       string
    AppliedAt        time.Time
}
```

### ValidationResult
```go
type ValidationResult struct {
    Fix                   *Fix
    InterfaceCompatible    bool
    TestsPass              bool
    ExecutionTimeImproved  bool
    OriginalExecutionTime  time.Duration
    NewExecutionTime       time.Duration
    Errors                 []error
    TestOutput             string
    ValidatedAt            time.Time
}
```

### MockImplementation
```go
type MockImplementation struct {
    ComponentName          string
    InterfaceName          string
    Package                string
    FilePath               string
    Code                   string
    InterfaceMethods       []MethodSignature
    Status                 MockStatus
    RequiresManualCompletion bool
    GeneratedAt            time.Time
}
```

