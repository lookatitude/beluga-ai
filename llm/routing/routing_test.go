package routing

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestHeuristicClassifier(t *testing.T) {
	tests := []struct {
		name     string
		msgs     []schema.Message
		wantTier ModelTier
	}{
		{
			name:     "simple short message",
			msgs:     []schema.Message{schema.NewHumanMessage("hi")},
			wantTier: TierSmall,
		},
		{
			name: "medium conversation",
			msgs: []schema.Message{
				schema.NewHumanMessage("Tell me about Go programming"),
				schema.NewAIMessage("Go is a statically typed language..."),
				schema.NewHumanMessage("How does concurrency work?"),
			},
			wantTier: TierMedium,
		},
	}

	classifier := &HeuristicClassifier{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier, err := classifier.Classify(ctx, tt.msgs)
			if err != nil {
				t.Fatalf("Classify: %v", err)
			}
			if tier != tt.wantTier {
				t.Errorf("tier = %q, want %q", tier, tt.wantTier)
			}
		})
	}
}

func TestDefaultCostRouter(t *testing.T) {
	models := []ModelConfig{
		{ID: "gpt-4o-mini", Tier: TierSmall, CostPerInputToken: 0.00001, CostPerOutputToken: 0.00002, Priority: 1},
		{ID: "gpt-4o", Tier: TierMedium, CostPerInputToken: 0.00005, CostPerOutputToken: 0.0001, Priority: 1},
		{ID: "claude-3-opus", Tier: TierLarge, CostPerInputToken: 0.0001, CostPerOutputToken: 0.0002, Priority: 1},
	}

	router := NewCostRouter(WithModels(models...))
	ctx := context.Background()

	selection, err := router.SelectModel(ctx, []schema.Message{schema.NewHumanMessage("hello")})
	if err != nil {
		t.Fatalf("SelectModel: %v", err)
	}

	if selection.ModelID == "" {
		t.Error("expected non-empty model ID")
	}
	if selection.Tier == "" {
		t.Error("expected non-empty tier")
	}
	if selection.Reason == "" {
		t.Error("expected non-empty reason")
	}
}

func TestDefaultCostRouter_NoModels(t *testing.T) {
	router := NewCostRouter()
	_, err := router.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
	if err == nil {
		t.Error("expected error for no models")
	}
}

func TestDefaultCostRouter_WithBudget(t *testing.T) {
	models := []ModelConfig{
		{ID: "expensive", Tier: TierSmall, CostPerInputToken: 1.0, CostPerOutputToken: 1.0, Priority: 1},
	}

	enforcer := NewBudgetEnforcer(0.001) // Very small budget.
	router := NewCostRouter(
		WithModels(models...),
		WithBudgetEnforcer(enforcer),
	)

	_, err := router.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hello")})
	if err == nil {
		t.Error("expected error for budget exceeded")
	}
}

func TestDefaultCostRouter_FallbackTier(t *testing.T) {
	models := []ModelConfig{
		{ID: "small-model", Tier: TierSmall, CostPerInputToken: 0.00001, CostPerOutputToken: 0.00002, Priority: 1},
	}

	router := NewCostRouter(
		WithModels(models...),
		WithFallbackTier(TierSmall),
	)

	selection, err := router.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hello")})
	if err != nil {
		t.Fatalf("SelectModel: %v", err)
	}
	if selection.ModelID != "small-model" {
		t.Errorf("ModelID = %q, want small-model", selection.ModelID)
	}
}

func TestInMemoryBudgetEnforcer(t *testing.T) {
	enforcer := NewBudgetEnforcer(10.0)
	ctx := context.Background()

	// Check should allow.
	allowed, err := enforcer.Check(ctx, 5.0)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !allowed {
		t.Error("expected allowed for first check")
	}

	// Record spend.
	if err := enforcer.Record(ctx, 8.0); err != nil {
		t.Fatalf("Record: %v", err)
	}

	// Should not allow exceeding budget.
	allowed, err = enforcer.Check(ctx, 5.0)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if allowed {
		t.Error("expected not allowed after spending near limit")
	}

	// Remaining.
	rem := enforcer.Remaining()
	if rem != 2.0 {
		t.Errorf("Remaining = %v, want 2.0", rem)
	}
}

func TestCheckAndReserve_Concurrent(t *testing.T) {
	enforcer := NewBudgetEnforcer(10.0)
	ctx := context.Background()

	// 100 concurrent reservations of 1.0 each against a budget of 10.0.
	// Exactly 10 must succeed.
	const n = 100
	results := make(chan bool, n)
	for i := 0; i < n; i++ {
		go func() {
			ok, _ := enforcer.CheckAndReserve(ctx, 1.0)
			results <- ok
		}()
	}
	successes := 0
	for i := 0; i < n; i++ {
		if <-results {
			successes++
		}
	}
	if successes != 10 {
		t.Errorf("CheckAndReserve successes = %d, want 10", successes)
	}
}

func TestDefaultCostRouter_Enforcer(t *testing.T) {
	enforcer := NewBudgetEnforcer(100.0)
	router := NewCostRouter(
		WithModels(ModelConfig{ID: "m", Tier: TierSmall, CostPerInputToken: 0.1, CostPerOutputToken: 0.1}),
		WithBudgetEnforcer(enforcer),
	)
	if router.Enforcer() == nil {
		t.Fatal("Enforcer() returned nil")
	}
}

func TestRegistry(t *testing.T) {
	r, err := NewRouter("default", RouterConfig{
		Models: []ModelConfig{{ID: "m", Tier: TierSmall, CostPerInputToken: 0.1, CostPerOutputToken: 0.1}},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil router")
	}
	if _, err := NewRouter("bogus", RouterConfig{}); err == nil {
		t.Error("expected error for unknown router")
	}
	if len(ListRouters()) == 0 {
		t.Error("expected registered routers")
	}
	if _, err := NewClassifier("heuristic", ClassifierConfig{}); err != nil {
		t.Errorf("NewClassifier: %v", err)
	}
	if _, err := NewEnforcer("inmemory", EnforcerConfig{DailyLimit: 5.0}); err != nil {
		t.Errorf("NewEnforcer: %v", err)
	}
}

func TestModelTierConstants(t *testing.T) {
	tiers := []ModelTier{TierSmall, TierMedium, TierLarge}
	for _, tier := range tiers {
		if tier == "" {
			t.Error("tier constant should not be empty")
		}
	}
}
