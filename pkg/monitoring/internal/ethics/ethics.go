// Package ethics provides ethical AI validation implementations
package ethics

import (
	"context"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/logger"
)

// EthicalAIChecker provides comprehensive ethical AI validation.
type EthicalAIChecker struct {
	logger          *logger.StructuredLogger
	fairnessMetrics *FairnessMetrics
	privacyChecker  *PrivacyChecker
	biasDetectors   []BiasDetector
}

// BiasDetector detects various types of bias in AI outputs.
type BiasDetector interface {
	Name() string
	Detect(content string, context iface.EthicalContext) []iface.BiasIssue
}

// FairnessMetrics tracks fairness and equity metrics.
type FairnessMetrics struct {
	DemographicParity map[string]float64 `json:"demographic_parity"`
	EqualOpportunity  map[string]float64 `json:"equal_opportunity"`
	DisparateImpact   map[string]float64 `json:"disparate_impact"`
}

// PrivacyChecker checks for privacy violations.
type PrivacyChecker struct {
	logger      *logger.StructuredLogger
	piiPatterns []*regexp.Regexp
}

// NewEthicalAIChecker creates a new ethical AI checker.
func NewEthicalAIChecker(logger *logger.StructuredLogger) *EthicalAIChecker {
	checker := &EthicalAIChecker{
		logger: logger,
		fairnessMetrics: &FairnessMetrics{
			DemographicParity: make(map[string]float64),
			EqualOpportunity:  make(map[string]float64),
			DisparateImpact:   make(map[string]float64),
		},
		privacyChecker: NewPrivacyChecker(logger),
	}

	// Initialize bias detectors
	checker.initializeBiasDetectors()

	return checker
}

// NewPrivacyChecker creates a new privacy checker.
func NewPrivacyChecker(logger *logger.StructuredLogger) *PrivacyChecker {
	pc := &PrivacyChecker{
		logger: logger,
	}

	// Initialize PII detection patterns
	pc.initializePIIPatterns()

	return pc
}

// initializeBiasDetectors sets up various bias detection algorithms.
func (eac *EthicalAIChecker) initializeBiasDetectors() {
	eac.biasDetectors = []BiasDetector{
		&GenderBiasDetector{},
		&RacialBiasDetector{},
		&SocioeconomicBiasDetector{},
		&CulturalBiasDetector{},
		&ConfirmationBiasDetector{},
	}
}

// initializePIIPatterns sets up PII detection patterns.
func (pc *PrivacyChecker) initializePIIPatterns() {
	patterns := []string{
		`\b\d{3}-\d{2}-\d{4}\b`,                               // SSN
		`\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`,             // Credit card
		`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
		`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`,                       // Phone number
		`\b\d{5}(?:[-\s]\d{4})?\b`,                            // ZIP code
		`\b\d{1,2}/\d{1,2}/\d{4}\b`,                           // Date of birth
	}

	pc.piiPatterns = make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		pc.piiPatterns[i] = regexp.MustCompile(pattern)
	}
}

// CheckContent performs comprehensive ethical analysis of content.
func (eac *EthicalAIChecker) CheckContent(ctx context.Context, content string, ethicalCtx iface.EthicalContext) (iface.EthicalAnalysis, error) {
	analysis := iface.EthicalAnalysis{
		Content:         content,
		Timestamp:       time.Now(),
		BiasIssues:      make([]iface.BiasIssue, 0),
		PrivacyIssues:   make([]iface.PrivacyIssue, 0),
		FairnessScore:   1.0,
		OverallRisk:     "low",
		Recommendations: make([]string, 0),
	}

	// Detect bias
	for _, detector := range eac.biasDetectors {
		issues := detector.Detect(content, ethicalCtx)
		analysis.BiasIssues = append(analysis.BiasIssues, issues...)

		for _, issue := range issues {
			analysis.FairnessScore *= (1.0 - issue.Severity)
		}
	}

	// Check privacy
	privacyIssues := eac.privacyChecker.CheckPrivacy(content)
	analysis.PrivacyIssues = append(analysis.PrivacyIssues, privacyIssues...)

	// Calculate overall risk
	calculateOverallRisk(&analysis)

	// Generate recommendations
	generateRecommendations(&analysis)

	// Log analysis
	eac.logger.Info(ctx, "Ethical analysis completed",
		map[string]any{
			"bias_issues":    len(analysis.BiasIssues),
			"privacy_issues": len(analysis.PrivacyIssues),
			"fairness_score": analysis.FairnessScore,
			"overall_risk":   analysis.OverallRisk,
		})

	return analysis, nil
}

// CheckPrivacy checks for privacy violations in content.
func (pc *PrivacyChecker) CheckPrivacy(content string) []iface.PrivacyIssue {
	issues := make([]iface.PrivacyIssue, 0)

	for _, pattern := range pc.piiPatterns {
		matches := pattern.FindAllString(content, -1)
		if len(matches) > 0 {
			issues = append(issues, iface.PrivacyIssue{
				Type:        "pii_detected",
				Description: "Potentially sensitive personal information detected",
				DataType:    pc.classifyPII(pattern.String()),
				Matches:     matches,
				Severity:    "high",
			})
		}
	}

	return issues
}

// classifyPII classifies the type of PII detected.
func (pc *PrivacyChecker) classifyPII(pattern string) string {
	switch {
	case strings.Contains(pattern, "email") || strings.Contains(pattern, "@"):
		return "email"
	case strings.Contains(pattern, "\\d{3}-\\d{2}-\\d{4}"):
		return "ssn"
	case strings.Contains(pattern, "\\d{4}[- ]?\\d{4}[- ]?\\d{4}[- ]?\\d{4}"):
		return "credit_card"
	case strings.Contains(pattern, "\\d{3}[-.]?\\d{3}[-.]?\\d{4}"):
		return "phone"
	case strings.Contains(pattern, "\\d{5}"):
		return "zip_code"
	case strings.Contains(pattern, "\\d{1,2}/\\d{1,2}/\\d{4}"):
		return "date_of_birth"
	default:
		return "unknown"
	}
}

// GenderBiasDetector detects gender-related bias.
type GenderBiasDetector struct{}

func (gbd *GenderBiasDetector) Name() string { return "gender_bias" }

func (gbd *GenderBiasDetector) Detect(content string, ctx iface.EthicalContext) []iface.BiasIssue {
	issues := make([]iface.BiasIssue, 0)

	patterns := []struct {
		pattern string
		message string
	}{
		{`(?i)\b(he|him|his)\b.*\b(she|her|hers)\b|\b(she|her|hers)\b.*\b(he|him|his)\b`, "Gender binary assumptions"},
		{`(?i)\b(men|man|male|men|boys)\b.*\b(should|must|always|are|better|superior|inferior|worse)\b`, "Stereotypical gender roles"},
		{`(?i)\b(women|woman|female|girls)\b.*\b(should|must|always|are|better|superior|inferior|worse)\b`, "Stereotypical gender roles"},
		{`(?i)\b(all|every)\b.*\b(women?|men|male|female|girls?|boys?)\b.*\b(are|should|must|always)\b`, "Generalized gender stereotypes"},
	}

	for _, p := range patterns {
		matched, err := regexp.MatchString(p.pattern, content)
		if err == nil && matched {
			issues = append(issues, iface.BiasIssue{
				Type:        "gender_bias",
				Description: p.message,
				Severity:    0.6,
				Evidence:    "Pattern match in content",
				Mitigation:  "Use gender-neutral language and avoid stereotypes",
			})
		}
	}

	return issues
}

// RacialBiasDetector detects racial and ethnic bias.
type RacialBiasDetector struct{}

func (rbd *RacialBiasDetector) Name() string { return "racial_bias" }

func (rbd *RacialBiasDetector) Detect(content string, ctx iface.EthicalContext) []iface.BiasIssue {
	issues := make([]iface.BiasIssue, 0)

	patterns := []string{
		`(?i)\b(white|black|asian|hispanic|latino|race|racial)\b.*\b(superior|inferior|better|worse)\b`,
		`(?i)\b(immigrant|migrant)\b.*\b(problem|issue|threat)\b`,
		`(?i)\b(they|those people)\b.*\b(don't|can't|won't|always|never|all)\b`,
		`(?i)\b(those people|they)\b.*\b(always|never|all|every)\b`, // More flexible pattern for "those people"
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, content)
		if err == nil && matched {
			issues = append(issues, iface.BiasIssue{
				Type:        "racial_bias",
				Description: "Potential racial or ethnic bias detected",
				Severity:    0.8,
				Evidence:    "Stereotypical language patterns",
				Mitigation:  "Use inclusive language and avoid generalizations about groups",
			})
			// Only add one issue per content to avoid duplicates
			break
		}
	}

	return issues
}

// SocioeconomicBiasDetector detects socioeconomic bias.
type SocioeconomicBiasDetector struct{}

func (sebd *SocioeconomicBiasDetector) Name() string { return "socioeconomic_bias" }

func (sebd *SocioeconomicBiasDetector) Detect(content string, ctx iface.EthicalContext) []iface.BiasIssue {
	issues := make([]iface.BiasIssue, 0)

	patterns := []string{
		`(?i)\b(poor|rich|wealthy)\b.*\b(lazy|hardworking|intelligent|stupid|greedy)\b`,
		`(?i)\b(welfare|benefits|recipients)\b.*\b(abuse|fraud|cheat|scam|trying)\b`,
		`(?i)\b(working class|middle class|upper class)\b.*\b(deserve|should have)\b`,
		`(?i)\b(poor people|rich people)\b.*\b(lazy|greedy)\b`, // More specific pattern
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, content)
		if err == nil && matched {
			issues = append(issues, iface.BiasIssue{
				Type:        "socioeconomic_bias",
				Description: "Potential socioeconomic or class bias detected",
				Severity:    0.5,
				Evidence:    "Class-based stereotypes",
				Mitigation:  "Avoid linking socioeconomic status to character traits",
			})
			// Only add one issue per content to avoid duplicates
			break
		}
	}

	return issues
}

// CulturalBiasDetector detects cultural bias.
type CulturalBiasDetector struct{}

func (cbd *CulturalBiasDetector) Name() string { return "cultural_bias" }

func (cbd *CulturalBiasDetector) Detect(content string, ctx iface.EthicalContext) []iface.BiasIssue {
	issues := make([]iface.BiasIssue, 0)

	patterns := []string{
		`(?i)\b(western|asian|african|european)\b.*\b(civilized|primitive|advanced|backward)\b`,
		`(?i)\b(traditional|modern)\b.*\b(better|worse|superior|primitive|compared)\b`,
		`(?i)\b(our culture|their culture)\b.*\b(right|wrong|superior)\b`,
		`(?i)\b(traditional.*societies|modern.*societies)\b.*\b(primitive|advanced)\b`, // More specific pattern
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, content)
		if err == nil && matched {
			issues = append(issues, iface.BiasIssue{
				Type:        "cultural_bias",
				Description: "Potential cultural bias detected",
				Severity:    0.7,
				Evidence:    "Cultural superiority assumptions",
				Mitigation:  "Respect cultural diversity and avoid ethnocentric judgments",
			})
			// Only add one issue per content to avoid duplicates
			break
		}
	}

	return issues
}

// ConfirmationBiasDetector detects confirmation bias.
type ConfirmationBiasDetector struct{}

func (cbd *ConfirmationBiasDetector) Name() string { return "confirmation_bias" }

func (cbd *ConfirmationBiasDetector) Detect(content string, ctx iface.EthicalContext) []iface.BiasIssue {
	issues := make([]iface.BiasIssue, 0)

	patterns := []string{
		`(?i)\b(as expected|as we know|obviously|clearly|of course)\b`,
		`(?i)\b(this proves|this shows|this confirms)\b.*\b(I was right|we were right)\b`,
		`(?i)\b(alternative|other)\b.*\b(view|perspective|theory)\b.*\b(wrong|incorrect|flawed)\b`,
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, content)
		if err == nil && matched {
			issues = append(issues, iface.BiasIssue{
				Type:        "confirmation_bias",
				Description: "Potential confirmation bias detected",
				Severity:    0.4,
				Evidence:    "Language suggesting preconceived notions",
				Mitigation:  "Consider alternative viewpoints and evidence",
			})
		}
	}

	return issues
}

// calculateOverallRisk calculates the overall risk level.
func calculateOverallRisk(ea *iface.EthicalAnalysis) {
	totalIssues := len(ea.BiasIssues) + len(ea.PrivacyIssues)
	maxSeverity := 0.0
	hasHighPrivacyIssue := false

	for _, issue := range ea.BiasIssues {
		if issue.Severity > maxSeverity {
			maxSeverity = issue.Severity
		}
	}

	for _, issue := range ea.PrivacyIssues {
		if issue.Severity == "high" {
			hasHighPrivacyIssue = true
			maxSeverity = math.Max(maxSeverity, 0.9)
		}
	}

	// Risk score: lower fairness score and higher severity = higher risk
	// Invert the formula so that lower fairness and higher severity increase risk
	riskScore := ((1.0 - ea.FairnessScore) * 0.6) + (maxSeverity * 0.4)
	riskScore = math.Max(0.0, math.Min(1.0, riskScore))

	// If there are high-severity privacy issues, automatically set to high risk
	if hasHighPrivacyIssue {
		ea.OverallRisk = "high"
		return
	}

	// If there are privacy issues (even without explicit "high" severity), set to at least medium
	// Multiple issues or low fairness score with privacy issues should be high risk
	if len(ea.PrivacyIssues) > 0 {
		if totalIssues > 1 || ea.FairnessScore < 0.6 {
			ea.OverallRisk = "high"
			return
		}
		ea.OverallRisk = "high" // Privacy issues are always high risk
		return
	}

	// If there are any issues but risk score is still low, set to at least medium
	if totalIssues > 0 && riskScore < 0.3 {
		ea.OverallRisk = "medium"
		return
	}

	// If fairness score is below 0.8 but no other issues, still consider it medium risk
	if totalIssues == 0 && ea.FairnessScore < 0.8 {
		ea.OverallRisk = "medium"
		return
	}

	switch {
	case riskScore < 0.3 || (totalIssues == 0 && ea.FairnessScore >= 0.8):
		ea.OverallRisk = "low"
	case riskScore < 0.7:
		ea.OverallRisk = "medium"
	default:
		ea.OverallRisk = "high"
	}
}

// generateRecommendations generates recommendations based on analysis.
func generateRecommendations(ea *iface.EthicalAnalysis) {
	if len(ea.BiasIssues) > 0 {
		ea.Recommendations = append(ea.Recommendations,
			"Review content for biased language and stereotypes")
	}

	if len(ea.PrivacyIssues) > 0 {
		ea.Recommendations = append(ea.Recommendations,
			"Remove or anonymize personal identifiable information")
	}

	if ea.FairnessScore < 0.7 {
		ea.Recommendations = append(ea.Recommendations,
			"Consider diverse perspectives and avoid generalizations")
	}

	if ea.OverallRisk == "high" {
		ea.Recommendations = append(ea.Recommendations,
			"Content requires human review before use")
	}
}

// HumanInTheLoopIntegration provides integration with human reviewers.
type HumanInTheLoopIntegration struct {
	logger     *logger.StructuredLogger
	thresholds map[string]float64
	reviewers  []string
}

// NewHumanInTheLoopIntegration creates a new HITL integration.
func NewHumanInTheLoopIntegration(logger *logger.StructuredLogger) *HumanInTheLoopIntegration {
	return &HumanInTheLoopIntegration{
		logger: logger,
		thresholds: map[string]float64{
			"bias":     0.6,
			"privacy":  0.8,
			"fairness": 0.5,
		},
	}
}

// ShouldTriggerReview determines if human review is needed.
func (hitl *HumanInTheLoopIntegration) ShouldTriggerReview(analysis *iface.EthicalAnalysis) bool {
	if analysis.OverallRisk == "high" {
		return true
	}

	if analysis.FairnessScore < hitl.thresholds["fairness"] {
		return true
	}

	return len(analysis.BiasIssues) > 2 || len(analysis.PrivacyIssues) > 0
}

// RequestReview requests human review for flagged content.
func (hitl *HumanInTheLoopIntegration) RequestReview(ctx context.Context, analysis *iface.EthicalAnalysis) error {
	hitl.logger.Warning(ctx, "Human review requested for ethical concerns",
		map[string]any{
			"risk_level":     analysis.OverallRisk,
			"bias_issues":    len(analysis.BiasIssues),
			"privacy_issues": len(analysis.PrivacyIssues),
			"fairness_score": analysis.FairnessScore,
		})

	// In a real implementation, this would integrate with a human review system
	// For now, just log the request

	return nil
}
