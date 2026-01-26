package safety

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/safety/iface"
)

func TestSafetyCheckerTableDriven(t *testing.T) {
	tests := []struct {
		name           string
		content        string
		wantSafe       bool
		wantIssueTypes []string
		minRiskScore   float64
		maxRiskScore   float64
	}{
		{
			name:         "safe content",
			content:      "Hello, how can I help you today?",
			wantSafe:     true,
			minRiskScore: 0.0,
			maxRiskScore: 0.29,
		},
		{
			name:           "toxicity detected",
			content:        "I hate you so much",
			wantSafe:       false,
			wantIssueTypes: []string{iface.IssueTypeToxicity},
			minRiskScore:   0.3,
			maxRiskScore:   1.0,
		},
		{
			name:           "bias detected - safe due to low weight",
			content:        "Everyone knows that this is true",
			wantSafe:       true, // Bias weight is 0.2, below 0.3 threshold
			wantIssueTypes: []string{iface.IssueTypeBias},
			minRiskScore:   0.1,
			maxRiskScore:   0.29,
		},
		{
			name:           "harmful content detected",
			content:        "Here is a tutorial on how to make explosives",
			wantSafe:       false,
			wantIssueTypes: []string{iface.IssueTypeHarmful},
			minRiskScore:   0.3,
			maxRiskScore:   1.0,
		},
		{
			name:         "empty content",
			content:      "",
			wantSafe:     true,
			minRiskScore: 0.0,
			maxRiskScore: 0.0,
		},
		{
			name:         "whitespace only",
			content:      "   \n\t  ",
			wantSafe:     true,
			minRiskScore: 0.0,
			maxRiskScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := TestContext()
			defer cancel()

			checker := NewSafetyChecker()
			result, err := checker.CheckContent(ctx, tt.content)

			require.NoError(t, err)
			assert.Equal(t, tt.wantSafe, result.Safe)
			assert.GreaterOrEqual(t, result.RiskScore, tt.minRiskScore)
			assert.LessOrEqual(t, result.RiskScore, tt.maxRiskScore)

			if len(tt.wantIssueTypes) > 0 {
				assert.NotEmpty(t, result.Issues)
				for _, wantType := range tt.wantIssueTypes {
					found := false
					for _, issue := range result.Issues {
						if issue.Type == wantType {
							found = true
							break
						}
					}
					assert.True(t, found, "expected issue type %s not found", wantType)
				}
			}
		})
	}
}

func TestSafetyCheckerConcurrency(t *testing.T) {
	ctx, cancel := TestContext()
	defer cancel()

	checker := NewSafetyChecker()

	const numGoroutines = 100
	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines)

	testContents := []string{
		"Hello world",
		"How are you today?",
		"This is a safe message",
		"Please help me with this task",
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			content := testContents[idx%len(testContents)]
			result, err := checker.CheckContent(ctx, content)
			if err != nil {
				errs <- err
				return
			}
			if !result.Safe {
				errs <- errors.New("expected safe content to be marked safe")
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent error: %v", err)
	}
}

func TestSafetyCheckerContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	checker := NewSafetyChecker()

	// Cancel context immediately
	cancel()

	// Should still work since CheckContent doesn't do blocking operations
	result, err := checker.CheckContent(ctx, "test content")
	require.NoError(t, err)
	assert.True(t, result.Safe)
}

func TestMockSafetyChecker(t *testing.T) {
	tests := []struct {
		name       string
		mockOpts   []MockOption
		content    string
		wantSafe   bool
		wantErr    bool
		wantCalls  int
	}{
		{
			name:      "default safe",
			mockOpts:  nil,
			content:   "test",
			wantSafe:  true,
			wantCalls: 1,
		},
		{
			name: "configured unsafe",
			mockOpts: []MockOption{
				WithMockSafe(false),
			},
			content:   "test",
			wantSafe:  false,
			wantCalls: 1,
		},
		{
			name: "configured error",
			mockOpts: []MockOption{
				WithMockError(errors.New("mock error")),
			},
			content:   "test",
			wantErr:   true,
			wantCalls: 1,
		},
		{
			name: "with risk score",
			mockOpts: []MockOption{
				WithMockRiskScore(0.8),
			},
			content:   "test",
			wantSafe:  false,
			wantCalls: 1,
		},
		{
			name: "with issues",
			mockOpts: []MockOption{
				WithMockIssues([]iface.SafetyIssue{MakeToxicityIssue()}),
			},
			content:   "test",
			wantSafe:  true, // issues don't automatically make it unsafe
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := TestContext()
			defer cancel()

			mock := NewMockSafetyChecker(tt.mockOpts...)
			result, err := mock.CheckContent(ctx, tt.content)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantSafe, result.Safe)
			assert.Equal(t, tt.wantCalls, mock.CallCount())
			assert.Equal(t, tt.content, mock.LastInput())
		})
	}
}

func TestMockSafetyCheckerDelay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mock := NewMockSafetyChecker(
		WithMockDelay(200 * time.Millisecond),
	)

	_, err := mock.CheckContent(ctx, "test")
	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestMockSafetyCheckerReset(t *testing.T) {
	ctx, cancel := TestContext()
	defer cancel()

	mock := NewMockSafetyChecker()

	// Make some calls
	_, _ = mock.CheckContent(ctx, "test1")
	_, _ = mock.CheckContent(ctx, "test2")
	assert.Equal(t, 2, mock.CallCount())
	assert.Equal(t, "test2", mock.LastInput())

	// Reset
	mock.Reset()
	assert.Equal(t, 0, mock.CallCount())
	assert.Equal(t, "", mock.LastInput())
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name:    "default config valid",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "risk threshold too high",
			cfg: &Config{
				RiskThreshold: 1.5, // > 1
			},
			wantErr: true,
		},
		{
			name: "risk threshold negative",
			cfg: &Config{
				RiskThreshold: -0.1,
			},
			wantErr: true,
		},
		{
			name: "valid custom threshold",
			cfg: &Config{
				RiskThreshold:  0.5,
				ToxicityWeight: 0.3,
				BiasWeight:     0.2,
				HarmfulWeight:  0.4,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigOptions(t *testing.T) {
	cfg := NewTestConfig(
		WithEnabled(false),
		WithRiskThreshold(0.5),
		WithToxicityWeight(0.3),
		WithBiasWeight(0.1),
		WithHarmfulWeight(0.6),
		WithEnableMetrics(false),
	)

	assert.False(t, cfg.Enabled)
	assert.Equal(t, 0.5, cfg.RiskThreshold)
	assert.Equal(t, 0.3, cfg.ToxicityWeight)
	assert.Equal(t, 0.1, cfg.BiasWeight)
	assert.Equal(t, 0.6, cfg.HarmfulWeight)
	assert.False(t, cfg.EnableMetrics)
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantType string
	}{
		{
			name:     "unsafe content error",
			err:      ErrUnsafeContent,
			wantType: "unsafe_content",
		},
		{
			name:     "check failed error",
			err:      ErrSafetyCheckFailed,
			wantType: "check_failed",
		},
		{
			name:     "high risk error",
			err:      ErrHighRiskContent,
			wantType: "high_risk",
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown"),
			wantType: "unknown",
		},
		{
			name:     "nil error",
			err:      nil,
			wantType: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType := errorType(tt.err)
			assert.Equal(t, tt.wantType, errType)
		})
	}
}

func BenchmarkSafetyChecker(b *testing.B) {
	ctx := context.Background()
	checker := NewSafetyChecker()
	content := "Hello, this is a test message for benchmarking the safety checker."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = checker.CheckContent(ctx, content)
	}
}

func BenchmarkSafetyCheckerParallel(b *testing.B) {
	ctx := context.Background()
	checker := NewSafetyChecker()
	content := "Hello, this is a test message for benchmarking the safety checker."

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = checker.CheckContent(ctx, content)
		}
	})
}
