package main

import "fmt"

// Error codes for test analyzer operations.
const (
	ErrCodePackageNotFound          = "PACKAGE_NOT_FOUND"
	ErrCodeInvalidPackage           = "INVALID_PACKAGE"
	ErrCodeAnalysisTimeout          = "ANALYSIS_TIMEOUT"
	ErrCodeFileNotFound             = "FILE_NOT_FOUND"
	ErrCodeInvalidGoFile            = "INVALID_GO_FILE"
	ErrCodeParseError               = "PARSE_ERROR"
	ErrCodeInvalidFunction          = "INVALID_FUNCTION"
	ErrCodeAnalysisError            = "ANALYSIS_ERROR"
	ErrCodeFixNotApplicable         = "FIX_NOT_APPLICABLE"
	ErrCodeBackupFailed             = "BACKUP_FAILED"
	ErrCodeFileModificationFailed   = "FILE_MODIFICATION_FAILED"
	ErrCodeInvalidFix               = "INVALID_FIX"
	ErrCodeValidationTimeout        = "VALIDATION_TIMEOUT"
	ErrCodeTestsFailed              = "TESTS_FAILED"
	ErrCodeExecutionTimeWorse       = "EXECUTION_TIME_WORSE"
	ErrCodeBackupNotFound           = "BACKUP_NOT_FOUND"
	ErrCodeRollbackFailed           = "ROLLBACK_FAILED"
	ErrCodeInterfaceNotFound        = "INTERFACE_NOT_FOUND"
	ErrCodeInvalidInterface         = "INVALID_INTERFACE"
	ErrCodeGenerationFailed         = "GENERATION_FAILED"
	ErrCodeTemplateGenerationFailed = "TEMPLATE_GENERATION_FAILED"
	ErrCodeVerificationFailed       = "VERIFICATION_FAILED"
	ErrCodeInvalidChange            = "INVALID_CHANGE"
	ErrCodeSyntaxError              = "SYNTAX_ERROR"
	ErrCodeWriteFailed              = "WRITE_FAILED"
	ErrCodeFormatFailed             = "FORMAT_FAILED"
	ErrCodeInvalidFormat            = "INVALID_FORMAT"
)

// AnalyzerError represents an error in the analyzer with operation context.
type AnalyzerError struct {
	Op   string
	Err  error
	Code string
}

// Error implements the error interface.
func (e *AnalyzerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Code, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Code)
}

// Unwrap returns the underlying error.
func (e *AnalyzerError) Unwrap() error {
	return e.Err
}

// NewAnalyzerError creates a new AnalyzerError.
func NewAnalyzerError(op, code string, err error) *AnalyzerError {
	return &AnalyzerError{
		Op:   op,
		Err:  err,
		Code: code,
	}
}

// Predefined error constructors for common errors.
var (
	ErrPackageNotFound = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodePackageNotFound, err)
	}
	ErrInvalidPackage = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeInvalidPackage, err)
	}
	ErrAnalysisTimeout = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeAnalysisTimeout, err)
	}
	ErrFileNotFound = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeFileNotFound, err)
	}
	ErrInvalidGoFile = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeInvalidGoFile, err)
	}
	ErrParseError = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeParseError, err)
	}
	ErrInvalidFunction = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeInvalidFunction, err)
	}
	ErrAnalysisError = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeAnalysisError, err)
	}
	ErrFixNotApplicable = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeFixNotApplicable, err)
	}
	ErrBackupFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeBackupFailed, err)
	}
	ErrFileModificationFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeFileModificationFailed, err)
	}
	ErrInvalidFix = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeInvalidFix, err)
	}
	ErrValidationTimeout = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeValidationTimeout, err)
	}
	ErrTestsFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeTestsFailed, err)
	}
	ErrExecutionTimeWorse = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeExecutionTimeWorse, err)
	}
	ErrBackupNotFound = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeBackupNotFound, err)
	}
	ErrRollbackFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeRollbackFailed, err)
	}
	ErrInterfaceNotFound = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeInterfaceNotFound, err)
	}
	ErrInvalidInterface = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeInvalidInterface, err)
	}
	ErrGenerationFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeGenerationFailed, err)
	}
	ErrTemplateGenerationFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeTemplateGenerationFailed, err)
	}
	ErrVerificationFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeVerificationFailed, err)
	}
	ErrInvalidChange = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeInvalidChange, err)
	}
	ErrSyntaxError = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeSyntaxError, err)
	}
	ErrWriteFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeWriteFailed, err)
	}
	ErrFormatFailed = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeFormatFailed, err)
	}
	ErrInvalidFormat = func(op string, err error) *AnalyzerError {
		return NewAnalyzerError(op, ErrCodeInvalidFormat, err)
	}
)
