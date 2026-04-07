package cost

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Tracker tests ---

func TestInMemoryTracker_Record(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		usage   Usage
		wantErr bool
	}{
		{
			name: "happy path",
			usage: Usage{
				InputTokens:  100,
				OutputTokens: 50,
				TotalTokens:  150,
				Cost:         0.0015,
				Model:        "gpt-4o",
				Provider:     "openai",
				Timestamp:    now,
			},
		},
		{
			name: "zero values allowed",
			usage: Usage{
				Timestamp: now,
			},
		},
		{
			name: "with tenant",
			usage: Usage{
				InputTokens: 10,
				TotalTokens: 10,
				TenantID:    "tenant-1",
				Provider:    "anthropic",
				Model:       "claude-3-5-sonnet",
				Timestamp:   now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewInMemoryTracker()
			err := tracker.Record(context.Background(), tt.usage)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestInMemoryTracker_Record_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tracker := NewInMemoryTracker()
	err := tracker.Record(ctx, Usage{Timestamp: time.Now()})
	require.Error(t, err)
}

func TestInMemoryTracker_Query(t *testing.T) {
	now := time.Now()

	baseEntries := []Usage{
		{InputTokens: 100, OutputTokens: 50, TotalTokens: 150, Cost: 0.01, Model: "gpt-4o", Provider: "openai", TenantID: "t1", Timestamp: now.Add(-30 * time.Minute)},
		{InputTokens: 200, OutputTokens: 80, TotalTokens: 280, Cost: 0.02, Model: "gpt-4o-mini", Provider: "openai", TenantID: "t1", Timestamp: now.Add(-10 * time.Minute)},
		{InputTokens: 50, OutputTokens: 25, TotalTokens: 75, Cost: 0.005, Model: "claude-3-5-sonnet", Provider: "anthropic", TenantID: "t2", Timestamp: now.Add(-5 * time.Minute)},
	}

	tests := []struct {
		name            string
		filter          Filter
		wantEntryCount  int64
		wantInputTokens int64
		wantCost        float64
	}{
		{
			name:            "empty filter returns all",
			filter:          Filter{},
			wantEntryCount:  3,
			wantInputTokens: 350,
			wantCost:        0.035,
		},
		{
			name:            "filter by provider",
			filter:          Filter{Provider: "openai"},
			wantEntryCount:  2,
			wantInputTokens: 300,
			wantCost:        0.03,
		},
		{
			name:            "filter by model",
			filter:          Filter{Model: "gpt-4o"},
			wantEntryCount:  1,
			wantInputTokens: 100,
			wantCost:        0.01,
		},
		{
			name:            "filter by tenant",
			filter:          Filter{TenantID: "t2"},
			wantEntryCount:  1,
			wantInputTokens: 50,
			wantCost:        0.005,
		},
		{
			name:            "filter by since",
			filter:          Filter{Since: now.Add(-15 * time.Minute)},
			wantEntryCount:  2,
			wantInputTokens: 250,
			wantCost:        0.025,
		},
		{
			name:            "filter by until",
			filter:          Filter{Until: now.Add(-20 * time.Minute)},
			wantEntryCount:  1,
			wantInputTokens: 100,
			wantCost:        0.01,
		},
		{
			name:            "filter by since and until",
			filter:          Filter{Since: now.Add(-35 * time.Minute), Until: now.Add(-8 * time.Minute)},
			wantEntryCount:  2,
			wantInputTokens: 300,
			wantCost:        0.03,
		},
		{
			name:            "no matches",
			filter:          Filter{Provider: "cohere"},
			wantEntryCount:  0,
			wantInputTokens: 0,
			wantCost:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewInMemoryTracker()
			for _, e := range baseEntries {
				require.NoError(t, tracker.Record(context.Background(), e))
			}

			summary, err := tracker.Query(context.Background(), tt.filter)
			require.NoError(t, err)
			require.NotNil(t, summary)
			assert.Equal(t, tt.wantEntryCount, summary.EntryCount, "EntryCount")
			assert.Equal(t, tt.wantInputTokens, summary.TotalInputTokens, "TotalInputTokens")
			assert.InDelta(t, tt.wantCost, summary.TotalCost, 1e-9, "TotalCost")
		})
	}
}

func TestInMemoryTracker_Query_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tracker := NewInMemoryTracker()
	_, err := tracker.Query(ctx, Filter{})
	require.Error(t, err)
}

func TestInMemoryTracker_MaxEntries(t *testing.T) {
	tracker := NewInMemoryTracker(WithMaxEntries(2))
	now := time.Now()

	entries := []Usage{
		{InputTokens: 10, TotalTokens: 10, Cost: 0.001, Timestamp: now.Add(-3 * time.Minute)},
		{InputTokens: 20, TotalTokens: 20, Cost: 0.002, Timestamp: now.Add(-2 * time.Minute)},
		{InputTokens: 30, TotalTokens: 30, Cost: 0.003, Timestamp: now.Add(-1 * time.Minute)},
	}
	for _, e := range entries {
		require.NoError(t, tracker.Record(context.Background(), e))
	}

	summary, err := tracker.Query(context.Background(), Filter{})
	require.NoError(t, err)
	// First entry evicted; only last 2 remain.
	assert.Equal(t, int64(2), summary.EntryCount)
	assert.Equal(t, int64(50), summary.TotalInputTokens)
}

// --- Registry tests ---

func TestRegistry_ListContainsInmemory(t *testing.T) {
	names := List()
	assert.Contains(t, names, "inmemory")
}

func TestRegistry_New_Inmemory(t *testing.T) {
	tracker, err := New("inmemory", Config{})
	require.NoError(t, err)
	require.NotNil(t, tracker)
}

func TestRegistry_New_Unknown(t *testing.T) {
	_, err := New("does-not-exist", Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

func TestRegistry_Register_And_New(t *testing.T) {
	Register("test-tracker", func(cfg Config) (Tracker, error) {
		return NewInMemoryTracker(), nil
	})
	defer func() {
		mu.Lock()
		delete(registry, "test-tracker")
		mu.Unlock()
	}()

	tracker, err := New("test-tracker", Config{})
	require.NoError(t, err)
	require.NotNil(t, tracker)
	assert.Contains(t, List(), "test-tracker")
}

// --- BudgetChecker tests ---

func TestInMemoryBudgetChecker_Check(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		existing      []Usage
		budget        Budget
		estimated     Usage
		wantAllowed   bool
		wantRatioGT   float64 // UsageRatio must be > this value
		wantReasonHas string
	}{
		{
			name:        "no limits set — always allowed",
			budget:      Budget{},
			estimated:   Usage{TotalTokens: 1000, Cost: 1.0},
			wantAllowed: true,
		},
		{
			name: "within token budget",
			existing: []Usage{
				{InputTokens: 1000, OutputTokens: 500, TotalTokens: 1500, Timestamp: now.Add(-30 * time.Minute)},
			},
			budget:      Budget{MaxTokensPerHour: 10_000, Action: BudgetActionReject},
			estimated:   Usage{TotalTokens: 500},
			wantAllowed: true,
		},
		{
			name: "token budget exceeded — reject",
			existing: []Usage{
				{InputTokens: 5000, OutputTokens: 4000, TotalTokens: 9000, Timestamp: now.Add(-30 * time.Minute)},
			},
			budget:        Budget{MaxTokensPerHour: 10_000, Action: BudgetActionReject},
			estimated:     Usage{TotalTokens: 2000},
			wantAllowed:   false,
			wantReasonHas: "hourly token budget exceeded",
		},
		{
			name: "token budget exceeded — alert action allows",
			existing: []Usage{
				{InputTokens: 5000, OutputTokens: 4000, TotalTokens: 9000, Timestamp: now.Add(-30 * time.Minute)},
			},
			budget:        Budget{MaxTokensPerHour: 10_000, Action: BudgetActionAlert},
			estimated:     Usage{TotalTokens: 2000},
			wantAllowed:   true,
			wantReasonHas: "hourly token budget exceeded",
		},
		{
			name: "approaching token budget — alert threshold",
			existing: []Usage{
				{InputTokens: 4000, OutputTokens: 3500, TotalTokens: 7500, Timestamp: now.Add(-30 * time.Minute)},
			},
			budget:        Budget{MaxTokensPerHour: 10_000, AlertThreshold: 0.8, Action: BudgetActionReject},
			estimated:     Usage{TotalTokens: 500},
			wantAllowed:   true,
			wantReasonHas: "approaching",
		},
		{
			name: "within cost budget",
			existing: []Usage{
				{Cost: 5.0, Timestamp: now.Add(-2 * time.Hour)},
			},
			budget:      Budget{MaxCostPerDay: 20.0, Action: BudgetActionReject},
			estimated:   Usage{Cost: 1.0},
			wantAllowed: true,
		},
		{
			name: "cost budget exceeded — reject",
			existing: []Usage{
				{Cost: 18.0, Timestamp: now.Add(-2 * time.Hour)},
			},
			budget:        Budget{MaxCostPerDay: 20.0, Action: BudgetActionReject},
			estimated:     Usage{Cost: 3.0},
			wantAllowed:   false,
			wantReasonHas: "daily cost budget exceeded",
		},
		{
			name: "cost budget exceeded — alert action allows",
			existing: []Usage{
				{Cost: 18.0, Timestamp: now.Add(-2 * time.Hour)},
			},
			budget:        Budget{MaxCostPerDay: 20.0, Action: BudgetActionAlert},
			estimated:     Usage{Cost: 3.0},
			wantAllowed:   true,
			wantReasonHas: "daily cost budget exceeded",
		},
		{
			name: "approaching cost budget — alert threshold",
			existing: []Usage{
				{Cost: 15.0, Timestamp: now.Add(-2 * time.Hour)},
			},
			budget:        Budget{MaxCostPerDay: 20.0, AlertThreshold: 0.8, Action: BudgetActionReject},
			estimated:     Usage{Cost: 1.0},
			wantAllowed:   true,
			wantReasonHas: "approaching",
		},
		{
			name: "old token records outside hour window are excluded",
			existing: []Usage{
				{InputTokens: 9000, OutputTokens: 9000, TotalTokens: 18000, Timestamp: now.Add(-90 * time.Minute)},
			},
			budget:      Budget{MaxTokensPerHour: 10_000, Action: BudgetActionReject},
			estimated:   Usage{TotalTokens: 5000},
			wantAllowed: true,
		},
		{
			name: "tenant isolation — only matching tenant counted",
			existing: []Usage{
				{InputTokens: 9000, OutputTokens: 9000, TotalTokens: 18000, TenantID: "other", Timestamp: now.Add(-30 * time.Minute)},
			},
			budget:      Budget{MaxTokensPerHour: 10_000, Action: BudgetActionReject},
			estimated:   Usage{TotalTokens: 5000, TenantID: "mine"},
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewInMemoryTracker()
			for _, e := range tt.existing {
				require.NoError(t, tracker.Record(context.Background(), e))
			}

			checker := NewInMemoryBudgetChecker(tracker)
			decision, err := checker.Check(context.Background(), tt.budget, tt.estimated)
			require.NoError(t, err)

			assert.Equal(t, tt.wantAllowed, decision.Allowed, "Allowed mismatch")
			if tt.wantReasonHas != "" {
				assert.Contains(t, decision.Reason, tt.wantReasonHas, "Reason mismatch")
			}
		})
	}
}

func TestInMemoryBudgetChecker_Check_TrackerError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so tracker.Query returns an error

	tracker := NewInMemoryTracker()
	checker := NewInMemoryBudgetChecker(tracker)

	_, err := checker.Check(ctx, Budget{MaxTokensPerHour: 1000, Action: BudgetActionReject}, Usage{TotalTokens: 100})
	require.Error(t, err)
}
