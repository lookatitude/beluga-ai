// Package safety provides error types for safety validation.
package safety

import "errors"

// Common safety-related errors.
var (
	// ErrUnsafeContent is returned when content fails safety validation.
	ErrUnsafeContent = errors.New("content contains unsafe material")

	// ErrSafetyCheckFailed is returned when the safety check process fails.
	ErrSafetyCheckFailed = errors.New("safety check process failed")

	// ErrHighRiskContent is returned when content has a high safety risk score.
	ErrHighRiskContent = errors.New("content has high safety risk")
)
