// Package contracts defines the API contracts for validation operations
// These interfaces define the contract for configuration validation functionality
// implementing FR-009, FR-025

package contracts

import (
	"context"
	"reflect"
	"time"
)

// Validator defines the contract for configuration validation
type Validator interface {
	// ValidateStruct validates a complete configuration structure
	// Implements FR-009: System MUST validate configuration schemas
	ValidateStruct(ctx context.Context, config interface{}) error

	// ValidateField validates a single field with its constraints
	ValidateField(ctx context.Context, field interface{}, tag string) error

	// ValidateKey validates a configuration value by key
	ValidateKey(ctx context.Context, key string, value interface{}) error

	// RegisterCustomValidator registers a custom validation function
	// Implements FR-009: System MUST validate with custom validation rules
	RegisterCustomValidator(tag string, validator CustomValidatorFunc) error

	// UnregisterCustomValidator removes a custom validation function
	UnregisterCustomValidator(tag string) error

	// GetValidationReport generates a detailed validation report
	// Implements FR-025: System MUST support configuration validation reporting
	GetValidationReport(ctx context.Context, config interface{}) ValidationReport
}

// CrossFieldValidator defines validation that spans multiple fields
type CrossFieldValidator interface {
	// ValidateCrossFields validates relationships between fields
	// Implements FR-009: System MUST validate with cross-field validation
	ValidateCrossFields(ctx context.Context, config interface{}) error

	// RegisterCrossFieldRule registers a rule that validates field relationships
	RegisterCrossFieldRule(name string, rule CrossFieldRule) error

	// UnregisterCrossFieldRule removes a cross-field validation rule
	UnregisterCrossFieldRule(name string) error

	// GetCrossFieldRules returns all registered cross-field rules
	GetCrossFieldRules() map[string]CrossFieldRule
}

// EnhancedValidator combines all validation capabilities
type EnhancedValidator interface {
	Validator
	CrossFieldValidator
	ValidationReporter
}

// ValidationReporter provides detailed validation reporting
type ValidationReporter interface {
	// GenerateReport creates a comprehensive validation report
	// Implements FR-025: System MUST provide actionable error messages
	GenerateReport(ctx context.Context, config interface{}) ValidationReport

	// GetFieldReport gets validation status for a specific field
	GetFieldReport(ctx context.Context, config interface{}, fieldPath string) FieldValidationReport

	// ExportReport exports validation report in specified format
	ExportReport(report ValidationReport, format ReportFormat) ([]byte, error)
}

// CustomValidatorFunc defines the signature for custom validation functions
type CustomValidatorFunc func(ctx context.Context, field reflect.Value) error

// CrossFieldRule defines a validation rule that spans multiple fields
type CrossFieldRule struct {
	Name        string                                              `json:"name"`
	Description string                                              `json:"description"`
	Validator   func(ctx context.Context, config interface{}) error `json:"-"`
	Fields      []string                                            `json:"fields"`
	Severity    ValidationSeverity                                  `json:"severity"`
	Message     string                                              `json:"message"`
}

// ValidationReport contains comprehensive validation results
type ValidationReport struct {
	Valid         bool                             `json:"valid"`
	Timestamp     time.Time                        `json:"timestamp"`
	ConfigType    string                           `json:"config_type"`
	TotalFields   int                              `json:"total_fields"`
	ValidFields   int                              `json:"valid_fields"`
	InvalidFields int                              `json:"invalid_fields"`
	Warnings      int                              `json:"warnings"`
	Errors        []ValidationError                `json:"errors,omitempty"`
	Warnings      []ValidationWarning              `json:"warnings,omitempty"`
	FieldReports  map[string]FieldValidationReport `json:"field_reports"`
	Summary       ValidationSummary                `json:"summary"`
}

// FieldValidationReport contains validation results for a single field
type FieldValidationReport struct {
	FieldPath   string              `json:"field_path"`
	Valid       bool                `json:"valid"`
	Value       interface{}         `json:"value,omitempty"`
	Constraints []string            `json:"constraints"`
	Errors      []ValidationError   `json:"errors,omitempty"`
	Warnings    []ValidationWarning `json:"warnings,omitempty"`
	Suggestions []string            `json:"suggestions,omitempty"`
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field      string                 `json:"field"`
	Tag        string                 `json:"tag"`
	Value      interface{}            `json:"value,omitempty"`
	Message    string                 `json:"message"`
	Severity   ValidationSeverity     `json:"severity"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Code       string                 `json:"code"`
}

// ValidationWarning represents a validation concern that doesn't prevent operation
type ValidationWarning struct {
	Field      string                 `json:"field"`
	Message    string                 `json:"message"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// ValidationSeverity defines the severity level of validation issues
type ValidationSeverity string

const (
	ValidationSeverityError   ValidationSeverity = "error"   // Blocks operation
	ValidationSeverityWarning ValidationSeverity = "warning" // Allows operation with concerns
	ValidationSeverityInfo    ValidationSeverity = "info"    // Informational only
)

// ValidationSummary provides aggregated validation information
type ValidationSummary struct {
	OverallStatus   string         `json:"overall_status"`
	ErrorsByField   map[string]int `json:"errors_by_field"`
	ErrorsByTag     map[string]int `json:"errors_by_tag"`
	MostCommonError string         `json:"most_common_error"`
	Recommendations []string       `json:"recommendations"`
	ComplianceScore float64        `json:"compliance_score"`
}

// ReportFormat defines supported validation report formats
type ReportFormat string

const (
	ReportFormatJSON ReportFormat = "json"
	ReportFormatYAML ReportFormat = "yaml"
	ReportFormatHTML ReportFormat = "html"
	ReportFormatText ReportFormat = "text"
)

// ValidationOptions configures validation behavior
type ValidationOptions struct {
	StrictMode            bool          `mapstructure:"strict_mode" default:"false"`
	FailOnWarnings        bool          `mapstructure:"fail_on_warnings" default:"false"`
	EnableSuggestions     bool          `mapstructure:"enable_suggestions" default:"true"`
	MaxErrors             int           `mapstructure:"max_errors" default:"100"`
	ValidationTimeout     time.Duration `mapstructure:"validation_timeout" default:"30s"`
	EnableCrossField      bool          `mapstructure:"enable_cross_field" default:"true"`
	EnableAsyncValidation bool          `mapstructure:"enable_async_validation" default:"false"`
}

// ValidatorError defines structured errors for validation operations
type ValidatorError struct {
	Op    string // operation that failed
	Field string // field being validated
	Tag   string // validation tag
	Err   error  // underlying error
	Code  string // error code
}

const (
	ErrCodeInvalidStructure       = "INVALID_STRUCTURE"
	ErrCodeValidationFailed       = "VALIDATION_FAILED"
	ErrCodeCustomValidatorFailed  = "CUSTOM_VALIDATOR_FAILED"
	ErrCodeCrossFieldFailed       = "CROSS_FIELD_FAILED"
	ErrCodeValidationTimeout      = "VALIDATION_TIMEOUT"
	ErrCodeReportGenerationFailed = "REPORT_GENERATION_FAILED"
)
