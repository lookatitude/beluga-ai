package optimizer

import (
	"context"
	"testing"
	"time"
)

func TestOptimizerType_Values(t *testing.T) {
	tests := []struct {
		name     string
		ot       OptimizerType
		expected string
	}{
		{"bootstrap_few_shot", OptimizerBootstrapFewShot, "bootstrap_few_shot"},
		{"mipro_v2", OptimizerMIPROv2, "mipro_v2"},
		{"gepa", OptimizerGEPA, "gepa"},
		{"simba", OptimizerSIMBA, "simba"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ot) != tt.expected {
				t.Errorf("got %q, want %q", string(tt.ot), tt.expected)
			}
		})
	}
}

func TestOptimizationStrategy_Values(t *testing.T) {
	tests := []struct {
		name     string
		strategy OptimizationStrategy
		expected string
	}{
		{"bootstrap_few_shot", StrategyBootstrapFewShot, "bootstrap_few_shot"},
		{"mipro_v2", StrategyMIPROv2, "mipro_v2"},
		{"gepa", StrategyGEPA, "gepa"},
		{"simba", StrategySIMBA, "simba"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.strategy) != tt.expected {
				t.Errorf("got %q, want %q", string(tt.strategy), tt.expected)
			}
		})
	}
}

func TestCompilePhase_Values(t *testing.T) {
	tests := []struct {
		name     string
		phase    CompilePhase
		expected string
	}{
		{"initializing", PhaseInitializing, "initializing"},
		{"training", PhaseTraining, "training"},
		{"validating", PhaseValidating, "validating"},
		{"finalizing", PhaseFinalizing, "finalizing"},
		{"complete", PhaseComplete, "complete"},
		{"error", PhaseError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.phase) != tt.expected {
				t.Errorf("got %q, want %q", string(tt.phase), tt.expected)
			}
		})
	}
}

func TestConvergenceStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   ConvergenceStatus
		expected string
	}{
		{"not_reached", ConvergenceNotReached, "not_reached"},
		{"reached", ConvergenceReached, "reached"},
		{"max_iterations", ConvergenceMaxIterations, "max_iterations"},
		{"max_cost", ConvergenceMaxCost, "max_cost"},
		{"unknown", ConvergenceStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProgress_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		phase    CompilePhase
		expected bool
	}{
		{"initializing is not complete", PhaseInitializing, false},
		{"training is not complete", PhaseTraining, false},
		{"validating is not complete", PhaseValidating, false},
		{"finalizing is not complete", PhaseFinalizing, false},
		{"complete is complete", PhaseComplete, true},
		{"error is complete", PhaseError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Progress{Phase: tt.phase}
			if got := p.IsComplete(); got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProgress_PercentComplete(t *testing.T) {
	tests := []struct {
		name         string
		currentTrial int
		totalTrials  int
		expected     float64
	}{
		{"zero total", 0, 0, 0.0},
		{"zero progress", 0, 10, 0.0},
		{"half complete", 5, 10, 0.5},
		{"fully complete", 10, 10, 1.0},
		{"quarter complete", 25, 100, 0.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Progress{CurrentTrial: tt.currentTrial, TotalTrials: tt.totalTrials}
			got := p.PercentComplete()
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBudget_IsExceeded(t *testing.T) {
	tests := []struct {
		name    string
		budget  Budget
		cost    float64
		calls   int
		elapsed time.Duration
		want    bool
	}{
		{
			name:   "nothing exceeded",
			budget: Budget{MaxCost: 10.0, MaxCalls: 100, MaxDuration: time.Hour},
			cost:   5.0, calls: 50, elapsed: 30 * time.Minute,
			want: false,
		},
		{
			name:   "cost exceeded",
			budget: Budget{MaxCost: 10.0},
			cost:   10.0, calls: 0, elapsed: 0,
			want: true,
		},
		{
			name:   "calls exceeded",
			budget: Budget{MaxCalls: 100},
			cost:   0, calls: 100, elapsed: 0,
			want: true,
		},
		{
			name:   "duration exceeded",
			budget: Budget{MaxDuration: time.Hour},
			cost:   0, calls: 0, elapsed: time.Hour,
			want: true,
		},
		{
			name:   "zero budget never exceeded",
			budget: Budget{},
			cost:   1000, calls: 1000, elapsed: 1000 * time.Hour,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.budget.IsExceeded(tt.cost, tt.calls, tt.elapsed)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBudget_Remaining(t *testing.T) {
	b := Budget{MaxCost: 100.0, MaxCalls: 200, MaxDuration: time.Hour}
	remaining := b.Remaining(25.0, 50, 15*time.Minute)

	if got := remaining["cost"]; got != 0.75 {
		t.Errorf("cost remaining: got %v, want 0.75", got)
	}
	if got := remaining["calls"]; got != 0.75 {
		t.Errorf("calls remaining: got %v, want 0.75", got)
	}
	if got := remaining["duration"]; got != 0.75 {
		t.Errorf("duration remaining: got %v, want 0.75", got)
	}
}

func TestBudget_Remaining_Clamped(t *testing.T) {
	b := Budget{MaxCost: 10.0, MaxCalls: 10}
	remaining := b.Remaining(20.0, 20, 0)

	if got := remaining["cost"]; got != 0.0 {
		t.Errorf("cost remaining: got %v, want 0.0", got)
	}
	if got := remaining["calls"]; got != 0.0 {
		t.Errorf("calls remaining: got %v, want 0.0", got)
	}
}

func TestCallbackFunc_NilHandlers(t *testing.T) {
	// Ensure nil handler functions don't panic.
	cb := CallbackFunc{}
	cb.OnProgress(nil, Progress{})
	cb.OnTrialComplete(nil, Trial{})
	cb.OnComplete(nil, Result{})
}

func TestCallbackFunc_WithHandlers(t *testing.T) {
	var progressCalled, trialCalled, completeCalled bool

	cb := CallbackFunc{
		OnProgressFunc:      func(_ context.Context, _ Progress) { progressCalled = true },
		OnTrialCompleteFunc: func(_ context.Context, _ Trial) { trialCalled = true },
		OnCompleteFunc:      func(_ context.Context, _ Result) { completeCalled = true },
	}

	cb.OnProgress(nil, Progress{})
	cb.OnTrialComplete(nil, Trial{})
	cb.OnComplete(nil, Result{})

	if !progressCalled {
		t.Error("OnProgress not called")
	}
	if !trialCalled {
		t.Error("OnTrialComplete not called")
	}
	if !completeCalled {
		t.Error("OnComplete not called")
	}
}
