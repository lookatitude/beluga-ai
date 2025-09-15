// Package iface provides safety and ethics related interfaces
package iface

import (
	"context"
	"time"
)

// SafetyChecker provides safety validation for AI content
type SafetyChecker interface {
	CheckContent(ctx context.Context, content string, contextInfo string) (SafetyResult, error)
	RequestHumanReview(ctx context.Context, content string, contextInfo string, riskScore float64) (ReviewDecision, error)
}

// SafetyResult represents the result of a safety check
type SafetyResult struct {
	Content   string        `json:"content"`
	Safe      bool          `json:"safe"`
	RiskScore float64       `json:"risk_score"`
	Issues    []SafetyIssue `json:"issues"`
	Timestamp time.Time     `json:"timestamp"`
}

// SafetyIssue represents a safety concern
type SafetyIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// ReviewDecision represents a human review decision
type ReviewDecision struct {
	Approved        bool      `json:"approved"`
	ModifiedContent string    `json:"modified_content,omitempty"`
	ReviewerID      string    `json:"reviewer_id"`
	Comments        string    `json:"comments"`
	Timestamp       time.Time `json:"timestamp"`
}

// EthicalChecker provides ethical AI validation
type EthicalChecker interface {
	CheckContent(ctx context.Context, content string, ethicalCtx EthicalContext) (EthicalAnalysis, error)
}

// EthicalContext provides context for ethical analysis
type EthicalContext struct {
	UserDemographics map[string]interface{} `json:"user_demographics"`
	ContentType      string                 `json:"content_type"`
	Domain           string                 `json:"domain"`
	CulturalContext  string                 `json:"cultural_context"`
	Stakeholders     []string               `json:"stakeholders"`
}

// EthicalAnalysis represents the result of ethical analysis
type EthicalAnalysis struct {
	Content         string         `json:"content"`
	Timestamp       time.Time      `json:"timestamp"`
	BiasIssues      []BiasIssue    `json:"bias_issues"`
	PrivacyIssues   []PrivacyIssue `json:"privacy_issues"`
	FairnessScore   float64        `json:"fairness_score"`
	OverallRisk     string         `json:"overall_risk"`
	Recommendations []string       `json:"recommendations"`
}

// BiasIssue represents a detected bias issue
type BiasIssue struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Severity    float64 `json:"severity"`
	Evidence    string  `json:"evidence"`
	Mitigation  string  `json:"mitigation"`
}

// PrivacyIssue represents a privacy violation
type PrivacyIssue struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	DataType    string   `json:"data_type"`
	Matches     []string `json:"matches"`
	Severity    string   `json:"severity"`
}

// BiasDetector detects various types of bias in AI outputs
type BiasDetector interface {
	Name() string
	Detect(content string, context *EthicalContext) []BiasIssue
}

// PrivacyChecker checks for privacy violations
type PrivacyChecker interface {
	CheckPrivacy(content string) []PrivacyIssue
}

// HumanInLoopIntegration provides human oversight capabilities
type HumanInLoopIntegration interface {
	ShouldTriggerReview(analysis *EthicalAnalysis) bool
	RequestReview(ctx context.Context, analysis *EthicalAnalysis) error
}

// Reviewer represents a human reviewer
type Reviewer interface {
	Review(ctx context.Context, request *ReviewRequest) (*ReviewDecision, error)
	GetID() string
}

// ReviewRequest represents a request for human review
type ReviewRequest struct {
	ID           string
	Content      string
	Context      string
	RiskScore    float64
	Timestamp    time.Time
	ResponseChan chan *ReviewDecision
}

// EthicalFilter provides ethical filtering capabilities
type EthicalFilter interface {
	FilterContent(ctx context.Context, content string) (string, error)
}

// ConcurrencyLimiter provides concurrency limiting
type ConcurrencyLimiter interface {
	Execute(ctx context.Context, fn func() error) error
	GetCurrentConcurrency() int
}
