package safety

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewSafetyChecker(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("safety_test")
	safetyChecker := NewSafetyChecker(mockLogger.(*logger.StructuredLogger))

	assert.NotNil(t, safetyChecker)
	assert.NotNil(t, safetyChecker.toxicityPatterns)
	assert.NotNil(t, safetyChecker.biasPatterns)
	assert.NotNil(t, safetyChecker.harmfulPatterns)
	assert.NotNil(t, safetyChecker.logger)
	assert.NotNil(t, safetyChecker.humanInLoop)
}

func TestSafetyCheckerCheckContent(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("safety_test")
	safetyChecker := NewSafetyChecker(mockLogger.(*logger.StructuredLogger))
	ctx := context.Background()

	tests := []struct {
		name         string
		content      string
		contextInfo  string
		expectSafe   bool
		minRiskScore float64
	}{
		{
			name:         "safe content",
			content:      "This is a normal message about cats and dogs.",
			contextInfo:  "chat",
			expectSafe:   true,
			minRiskScore: 0.0,
		},
		{
			name:         "content with toxicity",
			content:      "I hate everyone and want to kill them all.",
			contextInfo:  "chat",
			expectSafe:   false,
			minRiskScore: 0.4,
		},
		{
			name:         "content with bias",
			content:      "All women are bad at math, everyone knows that.",
			contextInfo:  "chat",
			expectSafe:   false,
			minRiskScore: 0.2,
		},
		{
			name:         "content with harmful instructions",
			content:      "How to hack into someone's computer easily.",
			contextInfo:  "chat",
			expectSafe:   false,
			minRiskScore: 0.5,
		},
		{
			name:         "medical context with moderate risk",
			content:      "Some potentially sensitive medical discussion.",
			contextInfo:  "medical",
			expectSafe:   false,
			minRiskScore: 0.1,
		},
		{
			name:         "high risk content",
			content:      "I want to murder everyone and hack all computers while being racist.",
			contextInfo:  "chat",
			expectSafe:   false,
			minRiskScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safetyChecker.CheckContent(ctx, tt.content, tt.contextInfo)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.content, result.Content)
			assert.Equal(t, tt.expectSafe, result.Safe)
			assert.True(t, result.RiskScore >= tt.minRiskScore,
				"Risk score %f should be >= %f", result.RiskScore, tt.minRiskScore)
			assert.NotZero(t, result.Timestamp)
		})
	}
}

func TestSafetyCheckerCheckPatterns(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("safety_test")
	safetyChecker := NewSafetyChecker(mockLogger.(*logger.StructuredLogger))

	t.Run("toxicity patterns", func(t *testing.T) {
		issues := safetyChecker.checkPatterns("I hate you and want to kill everyone", safetyChecker.toxicityPatterns, "toxicity")
		assert.NotEmpty(t, issues)
		assert.Equal(t, "toxicity", issues[0].Type)
		assert.Contains(t, strings.ToLower(issues[0].Description), "toxicity")
	})

	t.Run("bias patterns", func(t *testing.T) {
		issues := safetyChecker.checkPatterns("All men are superior to women", safetyChecker.biasPatterns, "bias")
		assert.NotEmpty(t, issues)
		assert.Equal(t, "bias", issues[0].Type)
		assert.Contains(t, strings.ToLower(issues[0].Description), "bias")
	})

	t.Run("harmful patterns", func(t *testing.T) {
		issues := safetyChecker.checkPatterns("Tutorial on how to hack websites", safetyChecker.harmfulPatterns, "harmful")
		assert.NotEmpty(t, issues)
		assert.Equal(t, "harmful", issues[0].Type)
		assert.Contains(t, strings.ToLower(issues[0].Description), "harmful")
	})

	t.Run("no matches", func(t *testing.T) {
		issues := safetyChecker.checkPatterns("This is a completely safe message", safetyChecker.toxicityPatterns, "toxicity")
		assert.Empty(t, issues)
	})
}

func TestSafetyCheckerGetSeverity(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("safety_test")
	safetyChecker := NewSafetyChecker(mockLogger.(*logger.StructuredLogger))

	tests := []struct {
		issueType string
		expected  string
	}{
		{"toxicity", "high"},
		{"harmful", "high"},
		{"bias", "medium"},
		{"unknown", "low"},
		{"", "low"},
	}

	for _, tt := range tests {
		t.Run(tt.issueType, func(t *testing.T) {
			severity := safetyChecker.getSeverity(tt.issueType)
			assert.Equal(t, tt.expected, severity)
		})
	}
}

func TestSafetyCheckerRequestHumanReview(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("safety_test")
	safetyChecker := NewSafetyChecker(mockLogger.(*logger.StructuredLogger))
	ctx := context.Background()

	t.Run("successful review request", func(t *testing.T) {
		content := "This content needs human review"
		riskScore := 0.8

		// This will timeout since there's no actual human reviewer
		resultChan := make(chan iface.ReviewDecision, 1)
		go func() {
			time.Sleep(100 * time.Millisecond)
			resultChan <- iface.ReviewDecision{
				Approved:   true,
				ReviewerID: "test_reviewer",
				Comments:   "Approved after review",
				Timestamp:  time.Now(),
			}
		}()

		result, err := safetyChecker.RequestHumanReview(ctx, content, "test", riskScore)
		if err == nil {
			assert.NotNil(t, result)
			assert.True(t, result.Approved)
		} else {
			assert.Contains(t, err.Error(), "timeout")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := safetyChecker.RequestHumanReview(cancelledCtx, "test content", "test", 0.8)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestNewHumanInLoop(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("human_in_loop_test")
	hil := NewHumanInLoop(mockLogger.(*logger.StructuredLogger))

	assert.NotNil(t, hil)
	assert.NotNil(t, hil.reviewQueue)
	assert.NotNil(t, hil.reviewers)
	assert.Equal(t, 0.7, hil.reviewThreshold)
	assert.Equal(t, 0.3, hil.autoApproveBelow)
}

func TestHumanInLoopAddReviewer(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("human_in_loop_test")
	hil := NewHumanInLoop(mockLogger.(*logger.StructuredLogger))

	mockReviewer := &MockReviewer{id: "test_reviewer"}
	hil.AddReviewer(mockReviewer)

	assert.Len(t, hil.reviewers, 1)
	assert.Equal(t, mockReviewer, hil.reviewers[0])
}

func TestHumanInLoopSimulateHumanReview(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("human_in_loop_test")
	hil := NewHumanInLoop(mockLogger.(*logger.StructuredLogger))

	request := &ReviewRequest{
		ID:        "test_request",
		RiskScore: 0.6,
	}

	decision := hil.simulateHumanReview(request)
	assert.NotNil(t, decision)
	assert.Equal(t, "simulated-reviewer", decision.ReviewerID)
	assert.NotZero(t, decision.Timestamp)
	assert.Contains(t, decision.Comments, "Review completed")

	// For risk score < 0.8, should be approved
	assert.True(t, decision.Approved)

	// Test high risk score
	request.RiskScore = 0.9
	decision = hil.simulateHumanReview(request)
	assert.False(t, decision.Approved)
	assert.Contains(t, decision.Comments, "Content flagged for manual review")
}

func TestNewEthicalFilter(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("ethical_filter_test")
	filter := NewEthicalFilter(mockLogger.(*logger.StructuredLogger))

	assert.NotNil(t, filter)
	assert.NotNil(t, filter.logger)
}

func TestEthicalFilterFilterContent(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("ethical_filter_test")
	filter := NewEthicalFilter(mockLogger.(*logger.StructuredLogger))
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		expected string
		modified bool
	}{
		{
			name:     "no sensitive terms",
			input:    "This is a normal message",
			expected: "This is a normal message",
			modified: false,
		},
		{
			name:     "contains hack",
			input:    "How to hack the system",
			expected: "How to modify the system",
			modified: true,
		},
		{
			name:     "contains exploit",
			input:    "Find security exploits",
			expected: "Find security utilize",
			modified: true,
		},
		{
			name:     "multiple terms",
			input:    "Learn to hack and crack passwords",
			expected: "Learn to modify and access passwords",
			modified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filter.FilterContent(ctx, tt.input)
			assert.NoError(t, err)

			if tt.modified {
				assert.NotEqual(t, tt.input, result)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewConcurrencyLimiter(t *testing.T) {
	limiter := NewConcurrencyLimiter(5)
	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.semaphore)
	assert.Equal(t, 5, limiter.maxConcurrent)
}

func TestConcurrencyLimiterExecute(t *testing.T) {
	limiter := NewConcurrencyLimiter(2)

	t.Run("successful execution", func(t *testing.T) {
		executed := false
		err := limiter.Execute(context.Background(), func() error {
			executed = true
			return nil
		})
		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("execution with error", func(t *testing.T) {
		testErr := assert.AnError
		err := limiter.Execute(context.Background(), func() error {
			return testErr
		})
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err := limiter.Execute(cancelledCtx, func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("concurrency limit exceeded", func(t *testing.T) {
		// Fill up the limiter
		blockingFuncs := make([]func() error, limiter.maxConcurrent+1)
		for i := 0; i < limiter.maxConcurrent; i++ {
			blockingFuncs[i] = func() error {
				time.Sleep(200 * time.Millisecond)
				return nil
			}
		}

		// Start max concurrent operations
		for i := 0; i < limiter.maxConcurrent; i++ {
			go limiter.Execute(context.Background(), blockingFuncs[i])
		}

		// Wait a bit for them to start
		time.Sleep(50 * time.Millisecond)

		// This should fail due to concurrency limit
		err := limiter.Execute(context.Background(), func() error {
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrency limit exceeded")
	})
}

func TestConcurrencyLimiterGetCurrentConcurrency(t *testing.T) {
	limiter := NewConcurrencyLimiter(3)

	// Initially should be 0
	assert.Equal(t, 0, limiter.GetCurrentConcurrency())

	// Start some operations
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			limiter.Execute(context.Background(), func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			})
		}()
	}

	// Wait a bit and check concurrency
	time.Sleep(50 * time.Millisecond)
	concurrency := limiter.GetCurrentConcurrency()
	assert.Equal(t, 2, concurrency)

	wg.Wait()

	// After completion, should be back to 0
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, limiter.GetCurrentConcurrency())
}

// Mock reviewer for testing
type MockReviewer struct {
	id string
}

func (mr *MockReviewer) Review(ctx context.Context, request *ReviewRequest) (iface.ReviewDecision, error) {
	return iface.ReviewDecision{
		Approved:   true,
		ReviewerID: mr.id,
		Comments:   "Mock review",
		Timestamp:  time.Now(),
	}, nil
}

func (mr *MockReviewer) GetID() string {
	return mr.id
}

// Benchmark tests
func BenchmarkSafetyChecker_CheckContent(b *testing.B) {
	mockLogger := logger.NewStructuredLogger("bench_test")
	safetyChecker := NewSafetyChecker(mockLogger.(*logger.StructuredLogger))
	ctx := context.Background()

	content := "This is a test message with some potentially sensitive content about hacking systems"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := safetyChecker.CheckContent(ctx, content, "test")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEthicalFilter_FilterContent(b *testing.B) {
	mockLogger := logger.NewStructuredLogger("bench_test")
	filter := NewEthicalFilter(mockLogger.(*logger.StructuredLogger))
	ctx := context.Background()

	content := "Learn how to hack, exploit, and crack various systems and applications"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := filter.FilterContent(ctx, content)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrencyLimiter_Execute(b *testing.B) {
	limiter := NewConcurrencyLimiter(10)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := limiter.Execute(ctx, func() error {
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
