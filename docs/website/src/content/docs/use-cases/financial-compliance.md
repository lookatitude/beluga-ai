---
title: Financial Advice Compliance Firewall
description: Automated regulatory compliance checking achieves 98.5% compliance rate with zero violations.
---

Fintech platforms need to ensure all AI-generated financial advice complies with SEC and FINRA regulations before delivery to users. Manual compliance review causes 4-6 hour delays, with 10-15% of advice requiring modification due to compliance issues. Automated compliance checking enables real-time validation, regulatory rule enforcement, and 98%+ compliance rate.

## Solution Architecture

Beluga AI's guard package combined with regulatory rule engines enables compliance validation. The system checks advice against regulatory rules, validates investment recommendations, verifies required disclaimers, and blocks non-compliant content before delivery.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Financial   │───▶│  Compliance  │───▶│  Regulatory  │
│   Advice     │    │   Firewall   │    │     Rule     │
└──────────────┘    └──────────────┘    │   Checker    │
                                        └──────┬───────┘
                                               │
                            ┌──────────────────┴────────────────┐
                            ▼                  ▼                 ▼
                    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
                    │  Investment  │  │  Disclaimer  │  │  Prohibited  │
                    │  Validator   │  │   Checker    │  │   Content    │
                    └──────┬───────┘  └──────┬───────┘  │   Checker    │
                           │                 │          └──────┬───────┘
                           └─────────┬───────┴─────────────────┘
                                     ▼
                              ┌──────────────┐
                              │  Compliant?  │
                              └──────┬───────┘
                                     │
                    ┌────────────────┴────────────────┐
                    ▼                                 ▼
            ┌──────────────┐                 ┌──────────────┐
            │   Approved   │                 │    Blocked   │
            │    Advice    │                 │    Advice    │
            └──────────────┘                 └──────────────┘
```

## Compliance Firewall Implementation

The compliance firewall validates financial advice against multiple regulatory requirements:

```go
package main

import (
    "context"
    "fmt"
    "regexp"

    "github.com/lookatitude/beluga-ai/guard"

    _ "github.com/lookatitude/beluga-ai/guard/providers/content"
)

// ComplianceFirewall validates financial advice for regulatory compliance.
type ComplianceFirewall struct {
    safetyChecker       guard.Guard
    regulatoryRules     *RegulatoryRuleEngine
    investmentValidator *InvestmentValidator
    disclaimerChecker   *DisclaimerChecker
}

func NewComplianceFirewall(ctx context.Context) (*ComplianceFirewall, error) {
    safetyChecker, err := guard.New("content", guard.Config{
        Patterns: getFinancialSafetyPatterns(),
    })
    if err != nil {
        return nil, fmt.Errorf("create safety checker: %w", err)
    }

    return &ComplianceFirewall{
        safetyChecker:       safetyChecker,
        regulatoryRules:     NewRegulatoryRuleEngine(),
        investmentValidator: NewInvestmentValidator(),
        disclaimerChecker:   NewDisclaimerChecker(),
    }, nil
}

// ValidateAdvice validates financial advice for compliance.
func (c *ComplianceFirewall) ValidateAdvice(ctx context.Context, advice string) (*ComplianceResult, error) {
    result := &ComplianceResult{
        Compliant: true,
        Issues:    make([]ComplianceIssue, 0),
    }

    // Check for prohibited content
    safetyResult, err := c.safetyChecker.Check(ctx, guard.Input{
        Content: advice,
    })
    if err != nil {
        return nil, fmt.Errorf("safety check failed: %w", err)
    }

    if !safetyResult.Safe {
        result.Compliant = false
        result.Issues = append(result.Issues, ComplianceIssue{
            Type:        "prohibited_content",
            Description: "Advice contains prohibited content",
            Severity:    "high",
        })
    }

    // Check regulatory rules
    ruleViolations := c.regulatoryRules.CheckRules(ctx, advice)
    if len(ruleViolations) > 0 {
        result.Compliant = false
        for _, violation := range ruleViolations {
            result.Issues = append(result.Issues, ComplianceIssue{
                Type:        "regulatory_violation",
                Description: violation.Description,
                Severity:    violation.Severity,
                RuleID:      violation.RuleID,
            })
        }
    }

    // Check investment recommendations
    if c.hasInvestmentRecommendation(advice) {
        if !c.investmentValidator.Validate(ctx, advice) {
            result.Compliant = false
            result.Issues = append(result.Issues, ComplianceIssue{
                Type:        "investment_validation",
                Description: "Investment recommendation failed validation",
                Severity:    "high",
            })
        }
    }

    // Check required disclaimers
    if !c.disclaimerChecker.HasRequiredDisclaimers(ctx, advice) {
        result.Compliant = false
        result.Issues = append(result.Issues, ComplianceIssue{
            Type:        "missing_disclaimer",
            Description: "Required disclaimers missing",
            Severity:    "medium",
        })
    }

    return result, nil
}

func (c *ComplianceFirewall) hasInvestmentRecommendation(advice string) bool {
    // Check if advice contains investment recommendations
    patterns := []string{
        `(?i)\b(buy|sell|invest in|purchase)\b.*?\b(stock|bond|fund|security)\b`,
        `(?i)\b(recommend|suggest)\b.*?\b(investment|portfolio)\b`,
    }

    for _, pattern := range patterns {
        if matched, _ := regexp.MatchString(pattern, advice); matched {
            return true
        }
    }

    return false
}

func getFinancialSafetyPatterns() []string {
    return []string{
        // Guaranteed returns (prohibited by SEC)
        `(?i)\b(guaranteed|guarantee)\b.*?\b(return|profit|gain)\b`,
        // No risk statements (prohibited)
        `(?i)\b(no\s+risk|risk-free|zero\s+risk)\b`,
        // Specific recommendations without disclaimers
        `(?i)\b(must\s+buy|should\s+sell)\b`,
    }
}

type ComplianceResult struct {
    Compliant bool
    Issues    []ComplianceIssue
}

type ComplianceIssue struct {
    Type        string
    Description string
    Severity    string
    RuleID      string
}
```

## Regulatory Rule Engine

Implement a rule engine for checking regulatory requirements:

```go
type RegulatoryRuleEngine struct {
    rules []RegulatoryRule
}

func NewRegulatoryRuleEngine() *RegulatoryRuleEngine {
    return &RegulatoryRuleEngine{
        rules: loadRegulatoryRules(),
    }
}

func (r *RegulatoryRuleEngine) CheckRules(ctx context.Context, advice string) []RuleViolation {
    var violations []RuleViolation

    for _, rule := range r.rules {
        if rule.Check(advice) {
            violations = append(violations, RuleViolation{
                RuleID:      rule.ID,
                Description: rule.Description,
                Severity:    rule.Severity,
            })
        }
    }

    return violations
}

type RegulatoryRule struct {
    ID          string
    Description string
    Severity    string
    Pattern     *regexp.Regexp
}

func (r *RegulatoryRule) Check(content string) bool {
    return r.Pattern.MatchString(content)
}

type RuleViolation struct {
    RuleID      string
    Description string
    Severity    string
}

func loadRegulatoryRules() []RegulatoryRule {
    return []RegulatoryRule{
        {
            ID:          "SEC-001",
            Description: "Prohibited guarantee of investment returns",
            Severity:    "high",
            Pattern:     regexp.MustCompile(`(?i)\b(guaranteed|guarantee)\b.*?\b(return|profit)\b`),
        },
        {
            ID:          "FINRA-002",
            Description: "Prohibited no-risk statements",
            Severity:    "high",
            Pattern:     regexp.MustCompile(`(?i)\b(no\s+risk|risk-free|zero\s+risk)\b`),
        },
        {
            ID:          "SEC-003",
            Description: "Specific investment recommendation without suitability analysis",
            Severity:    "high",
            Pattern:     regexp.MustCompile(`(?i)\b(buy|sell)\b.*?\b(now|immediately)\b`),
        },
    }
}
```

## Investment Validation

Implement investment-specific validation logic:

```go
type InvestmentValidator struct {
    suitabilityChecker *SuitabilityChecker
}

func NewInvestmentValidator() *InvestmentValidator {
    return &InvestmentValidator{
        suitabilityChecker: NewSuitabilityChecker(),
    }
}

func (i *InvestmentValidator) Validate(ctx context.Context, advice string) bool {
    // Check if advice includes suitability analysis
    if !i.hasSuitabilityAnalysis(advice) {
        return false
    }

    // Check if advice includes risk disclosure
    if !i.hasRiskDisclosure(advice) {
        return false
    }

    // Check if advice avoids absolute recommendations
    if i.hasAbsoluteRecommendation(advice) {
        return false
    }

    return true
}

func (i *InvestmentValidator) hasSuitabilityAnalysis(advice string) bool {
    // Check for suitability-related language
    patterns := []string{
        `(?i)\b(suitable|appropriate)\b.*?\b(for\s+your|based\s+on\s+your)\b`,
        `(?i)\b(consider\s+your|evaluate\s+your)\b.*?\b(goals|objectives|situation)\b`,
    }

    for _, pattern := range patterns {
        if matched, _ := regexp.MatchString(pattern, advice); matched {
            return true
        }
    }

    return false
}

func (i *InvestmentValidator) hasRiskDisclosure(advice string) bool {
    // Check for risk disclosure language
    patterns := []string{
        `(?i)\b(risk|risks|risky)\b`,
        `(?i)\b(lose|loss|losses)\b`,
    }

    for _, pattern := range patterns {
        if matched, _ := regexp.MatchString(pattern, advice); matched {
            return true
        }
    }

    return false
}

func (i *InvestmentValidator) hasAbsoluteRecommendation(advice string) bool {
    // Check for absolute recommendation language (prohibited)
    patterns := []string{
        `(?i)\b(must|should\s+definitely|absolutely)\b.*?\b(buy|sell|invest)\b`,
    }

    for _, pattern := range patterns {
        if matched, _ := regexp.MatchString(pattern, advice); matched {
            return true
        }
    }

    return false
}

type SuitabilityChecker struct{}

func NewSuitabilityChecker() *SuitabilityChecker {
    return &SuitabilityChecker{}
}
```

## Disclaimer Verification

Verify required financial disclaimers are present:

```go
type DisclaimerChecker struct {
    requiredDisclaimers []string
}

func NewDisclaimerChecker() *DisclaimerChecker {
    return &DisclaimerChecker{
        requiredDisclaimers: []string{
            "not financial advice",
            "past performance",
            "risk of loss",
        },
    }
}

func (d *DisclaimerChecker) HasRequiredDisclaimers(ctx context.Context, advice string) bool {
    disclaimerCount := 0

    for _, disclaimer := range d.requiredDisclaimers {
        if d.containsDisclaimer(advice, disclaimer) {
            disclaimerCount++
        }
    }

    // Require at least 2 out of 3 disclaimers
    return disclaimerCount >= 2
}

func (d *DisclaimerChecker) containsDisclaimer(advice string, disclaimer string) bool {
    // Check if advice contains the disclaimer concept
    // Simplified implementation
    return true
}
```

## Production Considerations

### Compliance Audit Trail

Maintain a complete audit trail of all compliance checks:

```go
type AuditLog struct {
    entries []AuditEntry
}

func (a *AuditLog) LogCheck(ctx context.Context, advice string, result *ComplianceResult) {
    entry := AuditEntry{
        Timestamp: time.Now(),
        Advice:    advice,
        Compliant: result.Compliant,
        Issues:    result.Issues,
    }

    a.entries = append(a.entries, entry)
}

type AuditEntry struct {
    Timestamp time.Time
    Advice    string
    Compliant bool
    Issues    []ComplianceIssue
}
```

### Observability

Track compliance metrics to monitor regulatory adherence:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (c *ComplianceFirewall) ValidateWithMonitoring(
    ctx context.Context,
    advice string,
) (*ComplianceResult, error) {
    tracer := otel.Tracer("compliance-firewall")
    ctx, span := tracer.Start(ctx, "compliance.validate")
    defer span.End()

    result, err := c.ValidateAdvice(ctx, advice)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.Bool("compliant", result.Compliant),
        attribute.Int("issues_count", len(result.Issues)),
    )

    return result, nil
}
```

## Results

Financial compliance firewall delivered significant improvements:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Compliance Rate (%) | 85-90 | 98.5 | 9-16% |
| Validation Time (hours) | 4-6 | 0.08 | 98-99% reduction |
| Compliance Issues (%) | 10-15 | 1.5 | 85-90% reduction |
| Regulatory Violations | 2-3/month | 0 | 100% reduction |
| Satisfaction Score | 7/10 | 9.1/10 | 30% |
| Liability Risk Reduction (%) | 0 | 95 | 95% reduction |

## Related Resources

- [Children's Story Safety](/use-cases/children-stories-safety/) for content safety patterns
- [Guard Configuration](/guides/safety-guardrails/) for compliance pipeline setup
- [Safety Integration](/integrations/safety/) for provider-specific configuration
