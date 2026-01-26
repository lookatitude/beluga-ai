// Package safety provides test utilities for safety validation.
package safety

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/safety/iface"
)

// MockSafetyChecker implements iface.SafetyChecker for testing.
type MockSafetyChecker struct {
	mu sync.RWMutex

	// Configurable behavior
	result    iface.SafetyResult
	err       error
	delay     time.Duration
	callCount int
	lastInput string
}

// MockOption configures MockSafetyChecker behavior.
type MockOption func(*MockSafetyChecker)

// NewMockSafetyChecker creates a new mock safety checker.
func NewMockSafetyChecker(opts ...MockOption) *MockSafetyChecker {
	m := &MockSafetyChecker{
		result: iface.SafetyResult{
			Safe:      true,
			RiskScore: 0.0,
			Timestamp: time.Now(),
		},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithMockResult configures the result to return.
func WithMockResult(result iface.SafetyResult) MockOption {
	return func(m *MockSafetyChecker) {
		m.result = result
	}
}

// WithMockSafe configures whether content is safe.
func WithMockSafe(safe bool) MockOption {
	return func(m *MockSafetyChecker) {
		m.result.Safe = safe
		if !safe {
			m.result.RiskScore = 0.5
		}
	}
}

// WithMockRiskScore configures the risk score.
func WithMockRiskScore(score float64) MockOption {
	return func(m *MockSafetyChecker) {
		m.result.RiskScore = score
		m.result.Safe = score < 0.3
	}
}

// WithMockIssues configures the issues to return.
func WithMockIssues(issues []iface.SafetyIssue) MockOption {
	return func(m *MockSafetyChecker) {
		m.result.Issues = issues
	}
}

// WithMockError configures an error to return.
func WithMockError(err error) MockOption {
	return func(m *MockSafetyChecker) {
		m.err = err
	}
}

// WithMockDelay adds a delay to simulate processing time.
func WithMockDelay(delay time.Duration) MockOption {
	return func(m *MockSafetyChecker) {
		m.delay = delay
	}
}

// CheckContent implements iface.SafetyChecker.
func (m *MockSafetyChecker) CheckContent(ctx context.Context, content string) (iface.SafetyResult, error) {
	m.mu.Lock()
	m.callCount++
	m.lastInput = content
	m.mu.Unlock()

	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return iface.SafetyResult{}, ctx.Err()
		}
	}

	if m.err != nil {
		return iface.SafetyResult{}, m.err
	}

	result := m.result
	result.Timestamp = time.Now()
	return result, nil
}

// CallCount returns the number of CheckContent calls.
func (m *MockSafetyChecker) CallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// LastInput returns the last input checked.
func (m *MockSafetyChecker) LastInput() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastInput
}

// Reset clears the call history.
func (m *MockSafetyChecker) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.lastInput = ""
}

// NewTestConfig creates a Config for testing.
func NewTestConfig(opts ...ConfigOption) *Config {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// TestContext creates a context with timeout for tests.
func TestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// MakeSafetyIssue creates a SafetyIssue for testing.
func MakeSafetyIssue(issueType, description, severity string) iface.SafetyIssue {
	return iface.SafetyIssue{
		Type:        issueType,
		Description: description,
		Severity:    severity,
	}
}

// MakeToxicityIssue creates a toxicity SafetyIssue for testing.
func MakeToxicityIssue() iface.SafetyIssue {
	return MakeSafetyIssue(iface.IssueTypeToxicity, "toxicity pattern detected", iface.SeverityMedium)
}

// MakeBiasIssue creates a bias SafetyIssue for testing.
func MakeBiasIssue() iface.SafetyIssue {
	return MakeSafetyIssue(iface.IssueTypeBias, "bias pattern detected", iface.SeverityMedium)
}

// MakeHarmfulIssue creates a harmful content SafetyIssue for testing.
func MakeHarmfulIssue() iface.SafetyIssue {
	return MakeSafetyIssue(iface.IssueTypeHarmful, "harmful pattern detected", iface.SeverityHigh)
}
