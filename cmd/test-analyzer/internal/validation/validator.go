package validation

import (
	"time"
)

// Fix represents a fix to be validated.
type Fix struct {
	Type             string
	Changes          []CodeChange
	Status           string
	BackupPath       string
	AppliedAt        time.Time
}

// CodeChange represents a code modification.
type CodeChange struct {
	File        string
	LineStart   int
	LineEnd     int
	OldCode     string
	NewCode     string
	Description string
}

// ValidationResult represents the result of fix validation.
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

// validator wraps FixValidator for backward compatibility.
type validator struct {
	fixValidator FixValidator
}

// NewValidator creates a new FixValidator instance (wraps NewFixValidator).
func NewValidator() FixValidator {
	return NewFixValidator()
}

