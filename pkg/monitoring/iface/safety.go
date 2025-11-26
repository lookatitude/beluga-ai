// Package iface provides safety and ethics related interfaces
package iface

import (
	"context"
	"time"
)

// SafetyChecker provides safety validation for AI content.
type SafetyChecker interface {
	CheckContent(ctx context.Context, content, contextInfo string) (SafetyResult, error)
	RequestHumanReview(ctx context.Context, content, contextInfo string, riskScore float64) (ReviewDecision, error)
}

// SafetyResult represents the result of a safety check.
type SafetyResult struct {
	Timestamp time.Time     `json:"timestamp"`
	Content   string        `json:"content"`
	Issues    []SafetyIssue `json:"issues"`
	RiskScore float64       `json:"risk_score"`
	Safe      bool          `json:"safe"`
}

// SafetyIssue represents a safety concern.
type SafetyIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// ReviewDecision represents a human review decision.
type ReviewDecision struct {
	Timestamp       time.Time `json:"timestamp"`
	ModifiedContent string    `json:"modified_content,omitempty"`
	ReviewerID      string    `json:"reviewer_id"`
	Comments        string    `json:"comments"`
	Approved        bool      `json:"approved"`
}

// EthicalChecker provides ethical AI validation.
type EthicalChecker interface {
	CheckContent(ctx context.Context, content string, ethicalCtx EthicalContext) (EthicalAnalysis, error)
}

// EthicalContext provides context for ethical analysis.
type EthicalContext struct {
	UserDemographics map[string]any `json:"user_demographics"`
	ContentType      string         `json:"content_type"`
	Domain           string         `json:"domain"`
	CulturalContext  string         `json:"cultural_context"`
	Stakeholders     []string       `json:"stakeholders"`
}

// EthicalAnalysis represents the result of ethical analysis.
type EthicalAnalysis struct {
	Timestamp       time.Time      `json:"timestamp"`
	Content         string         `json:"content"`
	OverallRisk     string         `json:"overall_risk"`
	BiasIssues      []BiasIssue    `json:"bias_issues"`
	PrivacyIssues   []PrivacyIssue `json:"privacy_issues"`
	Recommendations []string       `json:"recommendations"`
	FairnessScore   float64        `json:"fairness_score"`
}

// BiasIssue represents a detected bias issue.
type BiasIssue struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Evidence    string  `json:"evidence"`
	Mitigation  string  `json:"mitigation"`
	Severity    float64 `json:"severity"`
}

// PrivacyIssue represents a privacy violation.
type PrivacyIssue struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	DataType    string   `json:"data_type"`
	Severity    string   `json:"severity"`
	Matches     []string `json:"matches"`
}

// BiasDetector detects various types of bias in AI outputs.
type BiasDetector interface {
	Name() string
	Detect(content string, context *EthicalContext) []BiasIssue
}

// PrivacyChecker checks for privacy violations.
type PrivacyChecker interface {
	CheckPrivacy(content string) []PrivacyIssue
}

// HumanInLoopIntegration provides human oversight capabilities.
type HumanInLoopIntegration interface {
	ShouldTriggerReview(analysis *EthicalAnalysis) bool
	RequestReview(ctx context.Context, analysis *EthicalAnalysis) error
}

// Reviewer represents a human reviewer.
type Reviewer interface {
	Review(ctx context.Context, request *ReviewRequest) (*ReviewDecision, error)
	GetID() string
}

// ReviewRequest represents a request for human review.
type ReviewRequest struct {
	Timestamp    time.Time
	ResponseChan chan *ReviewDecision
	ID           string
	Content      string
	Context      string
	RiskScore    float64
}

// EthicalFilter provides ethical filtering capabilities.
type EthicalFilter interface {
	FilterContent(ctx context.Context, content string) (string, error)
}

// ConcurrencyLimiter provides concurrency limiting.
type ConcurrencyLimiter interface {
	Execute(ctx context.Context, fn func() error) error
	GetCurrentConcurrency() int
}
