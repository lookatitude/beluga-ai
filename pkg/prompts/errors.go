package prompts

// Re-export error types and functions from iface package for public API

import (
	"errors"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
)

// Error codes for the prompts package.
const (
	ErrCodeTemplateParse      = iface.ErrCodeTemplateParse
	ErrCodeTemplateExecute    = iface.ErrCodeTemplateExecute
	ErrCodeVariableMissing    = iface.ErrCodeVariableMissing
	ErrCodeVariableInvalid    = iface.ErrCodeVariableInvalid
	ErrCodeValidationFailed   = iface.ErrCodeValidationFailed
	ErrCodeCacheError         = iface.ErrCodeCacheError
	ErrCodeAdapterError       = iface.ErrCodeAdapterError
	ErrCodeConfigurationError = iface.ErrCodeConfigurationError
	ErrCodeTimeout            = iface.ErrCodeTimeout
)

// PromptError represents a custom error type for the prompts package.
type PromptError = iface.PromptError

// NewTemplateParseError creates a new template parse error.
func NewTemplateParseError(op, templateName string, err error) *PromptError {
	return iface.NewTemplateParseError(op, templateName, err)
}

// NewTemplateExecuteError creates a new template execution error.
func NewTemplateExecuteError(op, templateName string, err error) *PromptError {
	return iface.NewTemplateExecuteError(op, templateName, err)
}

// NewVariableMissingError creates a new variable missing error.
func NewVariableMissingError(op, variableName, templateName string) *PromptError {
	return iface.NewVariableMissingError(op, variableName, templateName)
}

// NewVariableInvalidError creates a new variable invalid error.
func NewVariableInvalidError(op, variableName, expectedType, actualType string) *PromptError {
	return iface.NewVariableInvalidError(op, variableName, expectedType, actualType)
}

// NewValidationError creates a new validation error.
func NewValidationError(op, details string, err error) *PromptError {
	return iface.NewValidationError(op, details, err)
}

// NewCacheError creates a new cache error.
func NewCacheError(op, details string, err error) *PromptError {
	return iface.NewCacheError(op, details, err)
}

// NewAdapterError creates a new adapter error.
func NewAdapterError(op, adapterType string, err error) *PromptError {
	return iface.NewAdapterError(op, adapterType, err)
}

// NewConfigurationError creates a new configuration error.
func NewConfigurationError(op, details string, err error) *PromptError {
	return iface.NewConfigurationError(op, details, err)
}

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(op, timeout string) *PromptError {
	return iface.NewTimeoutError(op, timeout)
}

// IsPromptError checks if an error is a PromptError.
func IsPromptError(err error) bool {
	var promptErr *PromptError
	return errors.As(err, &promptErr)
}

// AsPromptError attempts to convert an error to a PromptError.
func AsPromptError(err error) (*PromptError, bool) {
	var promptErr *PromptError
	if errors.As(err, &promptErr) {
		return promptErr, true
	}
	return nil, false
}

// GetErrorCode extracts the error code from an error if it's a PromptError.
func GetErrorCode(err error) (string, bool) {
	if promptErr, ok := AsPromptError(err); ok {
		return promptErr.Code, true
	}
	return "", false
}

// IsErrorCode checks if an error has a specific error code.
func IsErrorCode(err error, code string) bool {
	if promptErr, ok := AsPromptError(err); ok {
		return promptErr.Code == code
	}
	return false
}
