// Package safety provides safety validation and ethical AI checks for Beluga AI.
// It extracts safety middleware functionality from the monitoring package to provide
// reusable safety validation across the framework.
package safety

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// SafetyChecker provides comprehensive safety validation for AI operations.
type SafetyChecker struct {
	toxicityPatterns []*regexp.Regexp
	biasPatterns     []*regexp.Regexp
	harmfulPatterns  []*regexp.Regexp
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

// NewSafetyChecker creates a new safety checker with default patterns.
func NewSafetyChecker() *SafetyChecker {
	sc := &SafetyChecker{}
	sc.initializePatterns()
	return sc
}

// CheckContent performs safety validation on the given content.
func (sc *SafetyChecker) CheckContent(ctx context.Context, content string) (SafetyResult, error) {
	result := SafetyResult{
		Safe:      true,
		RiskScore: 0.0,
		Issues:    make([]SafetyIssue, 0),
		Timestamp: time.Now(),
	}

	// Check for toxicity
	if issues := sc.checkPatterns(content, sc.toxicityPatterns, "toxicity"); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		result.RiskScore += 0.4
	}

	// Check for bias
	if issues := sc.checkPatterns(content, sc.biasPatterns, "bias"); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		result.RiskScore += 0.2
	}

	// Check for harmful content
	if issues := sc.checkPatterns(content, sc.harmfulPatterns, "harmful"); len(issues) > 0 {
		result.Issues = append(result.Issues, issues...)
		result.RiskScore += 0.5
	}

	result.Safe = result.RiskScore < 0.3
	return result, nil
}

// checkPatterns checks content against a set of regex patterns.
func (sc *SafetyChecker) checkPatterns(content string, patterns []*regexp.Regexp, issueType string) []SafetyIssue {
	issues := make([]SafetyIssue, 0)

	for _, pattern := range patterns {
		if pattern.MatchString(content) {
			issues = append(issues, SafetyIssue{
				Type:        issueType,
				Description: issueType + " pattern detected in content",
				Severity:    "medium",
			})
		}
	}

	return issues
}

// initializePatterns sets up safety and bias detection patterns.
func (sc *SafetyChecker) initializePatterns() {
	// Toxicity patterns
	sc.toxicityPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(hate|kill|murder|rape|suicide)\b`),
		regexp.MustCompile(`(?i)\b(fuck|shit|cunt|asshole)\b`),
		regexp.MustCompile(`(?i)\b(nigger|chink|spic|wetback)\b`),
	}

	// Bias patterns
	sc.biasPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(all\s+(men|women|people))\b.*?\b(are|should|must)\b`),
		regexp.MustCompile(`(?i)\b(everyone\s+knows)\b`),
		regexp.MustCompile(`(?i)\b(it's\s+a\s+fact)\b.*?\b(that)\b`),
	}

	// Harmful patterns
	sc.harmfulPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(how\s+to|tutorial|guide)\b.*?\b(make|build|create)\b.*?\b(bombs?|weapons?|explosives?|drugs?)\b`),
		regexp.MustCompile(`(?i)\b(hack|crack|exploit)\b.*?\b(passwords?|accounts?|systems?)\b`),
		regexp.MustCompile(`(?i)\b(illegal|criminal|felonies|felony)\b.*?\b(activities|behaviors|actions|activity|behavior|action)\b`),
	}
}

// SafetyMiddleware wraps an agent with safety checks.
type SafetyMiddleware struct {
	iface.CompositeAgent
	checker *SafetyChecker
}

// NewSafetyMiddleware creates a new safety middleware.
func NewSafetyMiddleware(next iface.CompositeAgent) iface.CompositeAgent {
	return &SafetyMiddleware{
		CompositeAgent: next,
		checker:        NewSafetyChecker(),
	}
}

// Plan implements the Agent interface with safety checks.
func (sm *SafetyMiddleware) Plan(ctx context.Context, steps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	// Check input for safety if it's a string
	if inputStr, ok := inputs["input"].(string); ok {
		if result, err := sm.checker.CheckContent(ctx, inputStr); err != nil {
			return iface.AgentAction{}, iface.AgentFinish{}, fmt.Errorf("safety check failed: %w", err)
		} else if !result.Safe {
			return iface.AgentAction{}, iface.AgentFinish{
				ReturnValues: map[string]any{
					"error":  "Content failed safety validation",
					"issues": result.Issues,
				},
			}, nil
		}
	}

	return sm.CompositeAgent.Plan(ctx, steps, inputs)
}

// InputVariables returns the expected input variables for the agent.
func (sm *SafetyMiddleware) InputVariables() []string {
	return sm.CompositeAgent.InputVariables()
}

// OutputVariables returns the expected output variables from the agent.
func (sm *SafetyMiddleware) OutputVariables() []string {
	return sm.CompositeAgent.OutputVariables()
}

// GetTools returns the tools available to the agent.
func (sm *SafetyMiddleware) GetTools() []iface.Tool {
	return sm.CompositeAgent.GetTools()
}

// GetConfig returns the agent's configuration.
func (sm *SafetyMiddleware) GetConfig() schema.AgentConfig {
	return sm.CompositeAgent.GetConfig()
}

// GetLLM returns the LLM instance used by the agent.
func (sm *SafetyMiddleware) GetLLM() llmsiface.LLM {
	return sm.CompositeAgent.GetLLM()
}

// GetMetrics returns the metrics recorder for the agent.
func (sm *SafetyMiddleware) GetMetrics() iface.MetricsRecorder {
	return sm.CompositeAgent.GetMetrics()
}
