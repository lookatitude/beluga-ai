package iface

import "fmt"

// SchemaError represents errors specific to schema operations.
// It provides structured error information for programmatic error handling.
type SchemaError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Cause   error  // Underlying error that caused this error
}

// Error implements the error interface.
func (e *SchemaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error for error wrapping compatibility.
func (e *SchemaError) Unwrap() error {
	return e.Cause
}

// NewSchemaError creates a new SchemaError with the given code and message.
func NewSchemaError(code, message string, args ...interface{}) *SchemaError {
	return &SchemaError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

// WrapError wraps an existing error with schema context.
func WrapError(cause error, code, message string, args ...interface{}) *SchemaError {
	return &SchemaError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrCodeInvalidConfig         = "invalid_config"
	ErrCodeValidationFailed      = "validation_failed"
	ErrCodeInvalidMessage        = "invalid_message"
	ErrCodeInvalidDocument       = "invalid_document"
	ErrCodeSerializationFailed   = "serialization_failed"
	ErrCodeDeserializationFailed = "deserialization_failed"
	ErrCodeTypeConversionFailed  = "type_conversion_failed"
	ErrCodeSchemaMismatch        = "schema_mismatch"
	ErrCodeInvalidParameters     = "invalid_parameters"

	// A2A Communication error codes
	ErrCodeAgentMessageInvalid  = "agent_message_invalid"
	ErrCodeAgentRequestInvalid  = "agent_request_invalid"
	ErrCodeAgentResponseInvalid = "agent_response_invalid"
	ErrCodeAgentNotFound        = "agent_not_found"
	ErrCodeMessageTimeout       = "message_timeout"
	ErrCodeCommunicationFailed  = "communication_failed"
	ErrCodeConversationNotFound = "conversation_not_found"

	// Event error codes
	ErrCodeEventInvalid          = "event_invalid"
	ErrCodeEventPublishFailed    = "event_publish_failed"
	ErrCodeEventConsumeFailed    = "event_consume_failed"
	ErrCodeEventHandlerNotFound  = "event_handler_not_found"
	ErrCodeEventValidationFailed = "event_validation_failed"

	// Task and Workflow error codes
	ErrCodeTaskNotFound            = "task_not_found"
	ErrCodeTaskInvalid             = "task_invalid"
	ErrCodeWorkflowNotFound        = "workflow_not_found"
	ErrCodeWorkflowInvalid         = "workflow_invalid"
	ErrCodeWorkflowExecutionFailed = "workflow_execution_failed"

	// Validation error codes
	ErrCodeMessageTooLong          = "message_too_long"
	ErrCodeMessageEmpty            = "message_empty"
	ErrCodeInvalidMessageType      = "invalid_message_type"
	ErrCodeToolCallsExceeded       = "tool_calls_exceeded"
	ErrCodeEmbeddingTooLarge       = "embedding_too_large"
	ErrCodeMetadataTooLarge        = "metadata_too_large"
	ErrCodeRequiredFieldMissing    = "required_field_missing"
	ErrCodeInvalidFieldValue       = "invalid_field_value"
	ErrCodeContentValidationFailed = "content_validation_failed"

	// Configuration error codes
	ErrCodeConfigValidationFailed = "config_validation_failed"
	ErrCodeConfigLoadFailed       = "config_load_failed"
	ErrCodeConfigParseFailed      = "config_parse_failed"
	ErrCodeInvalidConfigFormat    = "invalid_config_format"

	// Factory error codes
	ErrCodeFactoryCreationFailed = "factory_creation_failed"
	ErrCodeFactoryNotFound       = "factory_not_found"
	ErrCodeInvalidFactoryConfig  = "invalid_factory_config"

	// Storage and persistence error codes
	ErrCodeStorageOperationFailed = "storage_operation_failed"
	ErrCodePersistenceFailed      = "persistence_failed"
	ErrCodeHistoryOperationFailed = "history_operation_failed"
	ErrCodeCacheOperationFailed   = "cache_operation_failed"
)

// IsSchemaError checks if an error is a SchemaError with the given code.
func IsSchemaError(err error, code string) bool {
	var schErr *SchemaError
	if !AsSchemaError(err, &schErr) {
		return false
	}
	return schErr.Code == code
}

// AsSchemaError attempts to cast an error to SchemaError.
func AsSchemaError(err error, target **SchemaError) bool {
	for err != nil {
		if schErr, ok := err.(*SchemaError); ok {
			*target = schErr
			return true
		}
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return false
}
