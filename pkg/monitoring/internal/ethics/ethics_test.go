package ethics

import (
	"context"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewEthicalAIChecker(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("ethics_test")
	checker := NewEthicalAIChecker(mockLogger.(*logger.StructuredLogger))

	assert.NotNil(t, checker)
	assert.NotNil(t, checker.logger)
	assert.NotNil(t, checker.biasDetectors)
	assert.NotNil(t, checker.fairnessMetrics)
	assert.NotNil(t, checker.privacyChecker)

	// Should have initialized bias detectors
	assert.Len(t, checker.biasDetectors, 5) // gender, racial, socioeconomic, cultural, confirmation
}

func TestNewPrivacyChecker(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("privacy_test")
	privacyChecker := NewPrivacyChecker(mockLogger.(*logger.StructuredLogger))

	assert.NotNil(t, privacyChecker)
	assert.NotNil(t, privacyChecker.logger)
	assert.NotNil(t, privacyChecker.piiPatterns)
	assert.True(t, len(privacyChecker.piiPatterns) > 0)
}

func TestEthicalAICheckerCheckContent(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("ethics_test")
	checker := NewEthicalAIChecker(mockLogger.(*logger.StructuredLogger))
	ctx := context.Background()

	ethicalCtx := iface.EthicalContext{
		UserDemographics: map[string]interface{}{
			"age_group": "25-34",
			"region":    "US",
		},
		ContentType:   "text",
		Domain:        "social_media",
		CulturalContext: "western",
	}

	t.Run("ethical content", func(t *testing.T) {
		content := "This is a respectful and inclusive message about community building."
		analysis, err := checker.CheckContent(ctx, content, ethicalCtx)

		assert.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.Equal(t, content, analysis.Content)
		assert.NotZero(t, analysis.Timestamp)
		assert.True(t, analysis.FairnessScore > 0.8) // Should be high for ethical content
		assert.Equal(t, "low", analysis.OverallRisk)
	})

	t.Run("content with bias issues", func(t *testing.T) {
		content := "All men are superior to women in leadership roles, everyone knows that."
		analysis, err := checker.CheckContent(ctx, content, ethicalCtx)

		assert.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.True(t, len(analysis.BiasIssues) > 0)
		assert.True(t, analysis.FairnessScore < 1.0) // Should be reduced due to bias
		assert.True(t, analysis.OverallRisk != "low")
	})

	t.Run("content with privacy issues", func(t *testing.T) {
		content := "Contact me at john.doe@example.com for more information."
		analysis, err := checker.CheckContent(ctx, content, ethicalCtx)

		assert.NoError(t, err)
		assert.NotNil(t, analysis)
		assert.True(t, len(analysis.PrivacyIssues) > 0)
		assert.Contains(t, analysis.OverallRisk, "high") // Privacy issues should increase risk
	})
}

func TestPrivacyCheckerCheckPrivacy(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("privacy_test")
	privacyChecker := NewPrivacyChecker(mockLogger.(*logger.StructuredLogger))

	tests := []struct {
		name            string
		content         string
		expectIssues    bool
		expectedDataType string
	}{
		{
			name:            "no PII",
			content:         "This is a normal message without any personal information.",
			expectIssues:    false,
			expectedDataType: "",
		},
		{
			name:            "email address",
			content:         "Contact me at test@example.com",
			expectIssues:    true,
			expectedDataType: "email",
		},
		{
			name:            "social security number",
			content:         "My SSN is 123-45-6789",
			expectIssues:    true,
			expectedDataType: "ssn",
		},
		{
			name:            "credit card number",
			content:         "Card: 1234-5678-9012-3456",
			expectIssues:    true,
			expectedDataType: "credit_card",
		},
		{
			name:            "phone number",
			content:         "Call me at 555-123-4567",
			expectIssues:    true,
			expectedDataType: "phone",
		},
		{
			name:            "date of birth",
			content:         "Born on 01/15/1990",
			expectIssues:    true,
			expectedDataType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := privacyChecker.CheckPrivacy(tt.content)

			if tt.expectIssues {
				assert.NotEmpty(t, issues, "Expected privacy issues for: %s", tt.content)
				if tt.expectedDataType != "" {
					found := false
					for _, issue := range issues {
						if strings.Contains(issue.DataType, tt.expectedDataType) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected data type %s in issues", tt.expectedDataType)
				}
			} else {
				assert.Empty(t, issues, "Expected no privacy issues for: %s", tt.content)
			}
		})
	}
}

func TestGenderBiasDetector(t *testing.T) {
	detector := &GenderBiasDetector{}

	ethicalCtx := iface.EthicalContext{
		ContentType: "text",
		Domain:      "general",
	}

	t.Run("no gender bias", func(t *testing.T) {
		content := "People of all genders can be successful leaders."
		issues := detector.Detect(content, ethicalCtx)
		assert.Empty(t, issues)
	})

	t.Run("gender stereotypes", func(t *testing.T) {
		content := "All women are naturally better at nurturing than men."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "gender_bias", issues[0].Type)
		assert.True(t, issues[0].Severity > 0)
		assert.Contains(t, strings.ToLower(issues[0].Description), "gender")
	})

	t.Run("gender binary assumptions", func(t *testing.T) {
		content := "He is strong and she is gentle, that's just how it is."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "gender_bias", issues[0].Type)
	})
}

func TestRacialBiasDetector(t *testing.T) {
	detector := &RacialBiasDetector{}

	ethicalCtx := iface.EthicalContext{
		ContentType: "text",
		Domain:      "general",
	}

	t.Run("no racial bias", func(t *testing.T) {
		content := "People from diverse backgrounds contribute valuable perspectives."
		issues := detector.Detect(content, ethicalCtx)
		assert.Empty(t, issues)
	})

	t.Run("racial stereotypes", func(t *testing.T) {
		content := "Those people are always causing trouble in the neighborhood."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "racial_bias", issues[0].Type)
		assert.True(t, issues[0].Severity > 0)
	})

	t.Run("racial superiority", func(t *testing.T) {
		content := "Our race is clearly superior to others in intelligence."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.True(t, issues[0].Severity >= 0.8) // High severity for superiority claims
	})
}

func TestSocioeconomicBiasDetector(t *testing.T) {
	detector := &SocioeconomicBiasDetector{}

	ethicalCtx := iface.EthicalContext{
		ContentType: "text",
		Domain:      "general",
	}

	t.Run("no socioeconomic bias", func(t *testing.T) {
		content := "People from all economic backgrounds deserve equal opportunities."
		issues := detector.Detect(content, ethicalCtx)
		assert.Empty(t, issues)
	})

	t.Run("class stereotypes", func(t *testing.T) {
		content := "Poor people are lazy and rich people are greedy."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "socioeconomic_bias", issues[0].Type)
		assert.Contains(t, strings.ToLower(issues[0].Description), "class")
	})

	t.Run("welfare stereotypes", func(t *testing.T) {
		content := "Welfare recipients are just trying to scam the system."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "socioeconomic_bias", issues[0].Type)
	})
}

func TestCulturalBiasDetector(t *testing.T) {
	detector := &CulturalBiasDetector{}

	ethicalCtx := iface.EthicalContext{
		ContentType: "text",
		Domain:      "general",
	}

	t.Run("no cultural bias", func(t *testing.T) {
		content := "Different cultures have unique and valuable traditions."
		issues := detector.Detect(content, ethicalCtx)
		assert.Empty(t, issues)
	})

	t.Run("cultural superiority", func(t *testing.T) {
		content := "Western culture is clearly more advanced than Eastern cultures."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "cultural_bias", issues[0].Type)
		assert.True(t, issues[0].Severity >= 0.7)
	})

	t.Run("traditional vs modern bias", func(t *testing.T) {
		content := "Traditional societies are primitive compared to modern ones."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "cultural_bias", issues[0].Type)
	})
}

func TestConfirmationBiasDetector(t *testing.T) {
	detector := &ConfirmationBiasDetector{}

	ethicalCtx := iface.EthicalContext{
		ContentType: "text",
		Domain:      "general",
	}

	t.Run("no confirmation bias", func(t *testing.T) {
		content := "Let's examine the evidence from multiple perspectives."
		issues := detector.Detect(content, ethicalCtx)
		assert.Empty(t, issues)
	})

	t.Run("confirmation bias language", func(t *testing.T) {
		content := "As expected, this proves that my theory was correct all along."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "confirmation_bias", issues[0].Type)
		assert.True(t, issues[0].Severity > 0)
		assert.Contains(t, strings.ToLower(issues[0].Description), "confirmation")
	})

	t.Run("overconfident language", func(t *testing.T) {
		content := "Obviously, only idiots would disagree with this viewpoint."
		issues := detector.Detect(content, ethicalCtx)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "confirmation_bias", issues[0].Type)
	})
}

func TestBiasDetectorNames(t *testing.T) {
	detectors := []BiasDetector{
		&GenderBiasDetector{},
		&RacialBiasDetector{},
		&SocioeconomicBiasDetector{},
		&CulturalBiasDetector{},
		&ConfirmationBiasDetector{},
	}

	expectedNames := []string{
		"gender_bias",
		"racial_bias",
		"socioeconomic_bias",
		"cultural_bias",
		"confirmation_bias",
	}

	for i, detector := range detectors {
		assert.Equal(t, expectedNames[i], detector.Name())
	}
}

func TestCalculateOverallRisk(t *testing.T) {
	tests := []struct {
		name           string
		biasIssues     int
		privacyIssues  int
		fairnessScore  float64
		expectedRisk   string
	}{
		{"low risk", 0, 0, 1.0, "low"},
		{"medium risk - bias", 1, 0, 0.8, "medium"},
		{"medium risk - fairness", 0, 0, 0.7, "medium"},
		{"high risk - privacy", 0, 1, 1.0, "high"},
		{"high risk - multiple", 2, 1, 0.5, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &iface.EthicalAnalysis{
				BiasIssues:    make([]iface.BiasIssue, tt.biasIssues),
				PrivacyIssues: make([]iface.PrivacyIssue, tt.privacyIssues),
				FairnessScore: tt.fairnessScore,
			}

			// Add some severity to bias issues
			for i := range analysis.BiasIssues {
				analysis.BiasIssues[i].Severity = 0.3
			}

			calculateOverallRisk(analysis)
			assert.Equal(t, tt.expectedRisk, analysis.OverallRisk)
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name            string
		biasIssues      int
		privacyIssues   int
		fairnessScore   float64
		overallRisk     string
		expectBiasRec   bool
		expectPrivacyRec bool
		expectFairnessRec bool
		expectRiskRec   bool
	}{
		{
			name:             "no issues",
			biasIssues:       0,
			privacyIssues:    0,
			fairnessScore:    1.0,
			overallRisk:      "low",
			expectBiasRec:    false,
			expectPrivacyRec: false,
			expectFairnessRec: false,
			expectRiskRec:    false,
		},
		{
			name:             "bias issues",
			biasIssues:       1,
			privacyIssues:    0,
			fairnessScore:    1.0,
			overallRisk:      "low",
			expectBiasRec:    true,
			expectPrivacyRec: false,
			expectFairnessRec: false,
			expectRiskRec:    false,
		},
		{
			name:             "privacy issues",
			biasIssues:       0,
			privacyIssues:    1,
			fairnessScore:    1.0,
			overallRisk:      "low",
			expectBiasRec:    false,
			expectPrivacyRec: true,
			expectFairnessRec: false,
			expectRiskRec:    false,
		},
		{
			name:             "low fairness",
			biasIssues:       0,
			privacyIssues:    0,
			fairnessScore:    0.6,
			overallRisk:      "low",
			expectBiasRec:    false,
			expectPrivacyRec: false,
			expectFairnessRec: true,
			expectRiskRec:    false,
		},
		{
			name:             "high risk",
			biasIssues:       0,
			privacyIssues:    0,
			fairnessScore:    1.0,
			overallRisk:      "high",
			expectBiasRec:    false,
			expectPrivacyRec: false,
			expectFairnessRec: false,
			expectRiskRec:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &iface.EthicalAnalysis{
				BiasIssues:      make([]iface.BiasIssue, tt.biasIssues),
				PrivacyIssues:   make([]iface.PrivacyIssue, tt.privacyIssues),
				FairnessScore:   tt.fairnessScore,
				OverallRisk:     tt.overallRisk,
				Recommendations: make([]string, 0),
			}

			generateRecommendations(analysis)

			recommendations := strings.Join(analysis.Recommendations, " ")

			if tt.expectBiasRec {
				assert.Contains(t, recommendations, "biased language")
			}
			if tt.expectPrivacyRec {
				assert.Contains(t, recommendations, "personal identifiable information")
			}
			if tt.expectFairnessRec {
				assert.Contains(t, recommendations, "diverse perspectives")
			}
			if tt.expectRiskRec {
				assert.Contains(t, recommendations, "human review")
			}
		})
	}
}

func TestNewHumanInTheLoopIntegration(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("hitl_test")
	hitl := NewHumanInTheLoopIntegration(mockLogger.(*logger.StructuredLogger))

	assert.NotNil(t, hitl)
	assert.NotNil(t, hitl.logger)
	assert.Equal(t, 0.6, hitl.thresholds["bias"])
	assert.Equal(t, 0.8, hitl.thresholds["privacy"])
	assert.Equal(t, 0.5, hitl.thresholds["fairness"])
}

func TestHumanInTheLoopShouldTriggerReview(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("hitl_test")
	hitl := NewHumanInTheLoopIntegration(mockLogger.(*logger.StructuredLogger))

	tests := []struct {
		name         string
		overallRisk  string
		fairnessScore float64
		biasIssues    int
		privacyIssues int
		expected      bool
	}{
		{"high risk", "high", 0.8, 1, 0, true},
		{"low fairness", "medium", 0.4, 0, 0, true},
		{"many bias issues", "low", 0.9, 3, 0, true},
		{"privacy issues", "low", 0.9, 0, 2, true},
		{"all good", "low", 0.9, 0, 0, false},
		{"medium risk only", "medium", 0.7, 1, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := &iface.EthicalAnalysis{
				OverallRisk:     tt.overallRisk,
				FairnessScore:   tt.fairnessScore,
				BiasIssues:      make([]iface.BiasIssue, tt.biasIssues),
				PrivacyIssues:   make([]iface.PrivacyIssue, tt.privacyIssues),
			}

			result := hitl.ShouldTriggerReview(analysis)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHumanInTheLoopRequestReview(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("hitl_test")
	hitl := NewHumanInTheLoopIntegration(mockLogger.(*logger.StructuredLogger))
	ctx := context.Background()

	analysis := &iface.EthicalAnalysis{
		OverallRisk:     "high",
		FairnessScore:   0.8,
		BiasIssues:      make([]iface.BiasIssue, 1),
		PrivacyIssues:   make([]iface.PrivacyIssue, 0),
		Recommendations: make([]string, 0),
	}

	err := hitl.RequestReview(ctx, analysis)
	assert.NoError(t, err) // Should not error even without actual reviewers
}

// Benchmark tests
func BenchmarkEthicalAIChecker_CheckContent(b *testing.B) {
	mockLogger := logger.NewStructuredLogger("bench_test")
	checker := NewEthicalAIChecker(mockLogger.(*logger.StructuredLogger))
	ctx := context.Background()

	content := "This is a test message that might contain some biased language or sensitive information."
	ethicalCtx := iface.EthicalContext{
		ContentType: "text",
		Domain:      "general",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.CheckContent(ctx, content, ethicalCtx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPrivacyChecker_CheckPrivacy(b *testing.B) {
	mockLogger := logger.NewStructuredLogger("bench_test")
	privacyChecker := NewPrivacyChecker(mockLogger.(*logger.StructuredLogger))

	content := "Test content with email@example.com and phone 555-123-4567 and SSN 123-45-6789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		issues := privacyChecker.CheckPrivacy(content)
		_ = issues
	}
}

func BenchmarkBiasDetectors(b *testing.B) {
	detectors := []BiasDetector{
		&GenderBiasDetector{},
		&RacialBiasDetector{},
		&SocioeconomicBiasDetector{},
		&CulturalBiasDetector{},
		&ConfirmationBiasDetector{},
	}

	content := "This content contains various types of potential bias and stereotypes."
	ethicalCtx := iface.EthicalContext{
		ContentType: "text",
		Domain:      "general",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, detector := range detectors {
			issues := detector.Detect(content, ethicalCtx)
			_ = issues
		}
	}
}

func BenchmarkCalculateOverallRisk(b *testing.B) {
	analysis := &iface.EthicalAnalysis{
		BiasIssues:      make([]iface.BiasIssue, 5),
		PrivacyIssues:   make([]iface.PrivacyIssue, 2),
		FairnessScore:   0.7,
		Recommendations: make([]string, 0),
	}

	// Add severity to bias issues
	for i := range analysis.BiasIssues {
		analysis.BiasIssues[i].Severity = 0.2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateOverallRisk(analysis)
	}
}
