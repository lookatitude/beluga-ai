package prompts

// Re-export error types and functions from iface package for public API

import "github.com/lookatitude/beluga-ai/pkg/prompts/iface"

// Error codes for the prompts package
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

// PromptError represents a custom error type for the prompts package
type PromptError = iface.PromptError

// NewTemplateParseError creates a new template parse error
func NewTemplateParseError(op string, templateName string, err error) *PromptError {
	return iface.NewTemplateParseError(op, templateName, err)
}

// NewTemplateExecuteError creates a new template execution error
func NewTemplateExecuteError(op string, templateName string, err error) *PromptError {
	return iface.NewTemplateExecuteError(op, templateName, err)
}

// NewVariableMissingError creates a new variable missing error
func NewVariableMissingError(op string, variableName string, templateName string) *PromptError {
	return iface.NewVariableMissingError(op, variableName, templateName)
}

// NewVariableInvalidError creates a new variable invalid error
func NewVariableInvalidError(op string, variableName string, expectedType string, actualType string) *PromptError {
	return iface.NewVariableInvalidError(op, variableName, expectedType, actualType)
}

// NewValidationError creates a new validation error
func NewValidationError(op string, details string, err error) *PromptError {
	return iface.NewValidationError(op, details, err)
}

// NewCacheError creates a new cache error
func NewCacheError(op string, details string, err error) *PromptError {
	return iface.NewCacheError(op, details, err)
}

// NewAdapterError creates a new adapter error
func NewAdapterError(op string, adapterType string, err error) *PromptError {
	return iface.NewAdapterError(op, adapterType, err)
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(op string, details string, err error) *PromptError {
	return iface.NewConfigurationError(op, details, err)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(op string, timeout string) *PromptError {
	return iface.NewTimeoutError(op, timeout)
}
