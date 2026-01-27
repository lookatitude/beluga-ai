package rag

import (
	"errors"
	"fmt"
)

// Error codes for the convenience RAG package.
const (
	ErrCodeMissingEmbedder     = "missing_embedder"
	ErrCodeEmbedderCreation    = "embedder_creation_failed"
	ErrCodeVectorStoreCreation = "vectorstore_creation_failed"
	ErrCodeLLMCreation         = "llm_creation_failed"
	ErrCodeSplitterCreation    = "splitter_creation_failed"
	ErrCodeRetrieverCreation   = "retriever_creation_failed"
	ErrCodeRetrievalFailed     = "retrieval_failed"
	ErrCodeGenerationFailed    = "generation_failed"
	ErrCodeNoLLM               = "no_llm_configured"
	ErrCodeIngestionFailed     = "ingestion_failed"
	ErrCodeInvalidConfig       = "invalid_config"
	ErrCodeDocumentLoad        = "document_load_failed"
)

// Error represents a convenience RAG error following the Op/Err/Code pattern.
type Error struct {
	Err     error
	Fields  map[string]any
	Op      string
	Code    string
	Message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("convenience/rag %s: %s (code: %s)", e.Op, e.Message, e.Code)
	}
	if e.Err != nil {
		return fmt.Sprintf("convenience/rag %s: %v (code: %s)", e.Op, e.Err, e.Code)
	}
	return fmt.Sprintf("convenience/rag %s: unknown error (code: %s)", e.Op, e.Code)
}

// Unwrap returns the underlying error for error wrapping.
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new Error following the Op/Err/Code pattern.
func NewError(op, code string, err error) *Error {
	return &Error{
		Op:     op,
		Code:   code,
		Err:    err,
		Fields: make(map[string]any),
	}
}

// NewErrorWithMessage creates a new Error with a custom message.
func NewErrorWithMessage(op, code, message string, err error) *Error {
	return &Error{
		Op:      op,
		Code:    code,
		Message: message,
		Err:     err,
		Fields:  make(map[string]any),
	}
}

// WithField adds a context field to the error.
func (e *Error) WithField(key string, value any) *Error {
	if e.Fields == nil {
		e.Fields = make(map[string]any)
	}
	e.Fields[key] = value
	return e
}

// Common error variables.
var (
	ErrMissingEmbedder = errors.New("embedder is required")
	ErrNoLLM           = errors.New("LLM is not configured for generation")
	ErrInvalidConfig   = errors.New("invalid configuration")
)

// IsError checks if an error is a convenience RAG Error.
func IsError(err error) bool {
	var ragErr *Error
	return errors.As(err, &ragErr)
}

// GetErrorCode extracts the error code from an Error if present.
func GetErrorCode(err error) string {
	var ragErr *Error
	if errors.As(err, &ragErr) {
		return ragErr.Code
	}
	return ""
}
