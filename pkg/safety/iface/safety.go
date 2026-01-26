// Package iface defines interfaces for the safety package.
package iface

import (
	"context"
	"time"
)

// SafetyChecker defines the interface for content safety validation.
type SafetyChecker interface {
	// CheckContent performs safety validation on the given content.
	CheckContent(ctx context.Context, content string) (SafetyResult, error)
}

// SafetyResult represents the result of a safety check.
type SafetyResult struct {
	Timestamp time.Time     `json:"timestamp"`
	Issues    []SafetyIssue `json:"issues,omitempty"`
	RiskScore float64       `json:"risk_score"`
	Safe      bool          `json:"safe"`
}

// SafetyIssue represents a specific safety concern found during validation.
type SafetyIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// Severity levels for safety issues.
const (
	SeverityLow    = "low"
	SeverityMedium = "medium"
	SeverityHigh   = "high"
)

// Issue types for safety concerns.
const (
	IssueTypeToxicity = "toxicity"
	IssueTypeBias     = "bias"
	IssueTypeHarmful  = "harmful"
)
