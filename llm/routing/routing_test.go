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

// stubClassifier returns a fixed tier, used to exercise WithClassifier
// and classifier error paths.
type stubClassifier struct {
	tier ModelTier
	err  error
}

func (s *stubClassifier) Classify(_ context.Context, _ []schema.Message) (ModelTier, error) {
	return s.tier, s.err
}

func TestHeuristicClassifier_LargeAndSystem(t *testing.T) {
	classifier := &HeuristicClassifier{}
	ctx := context.Background()

	// Many messages → TierLarge.
	many := make([]schema.Message, 12)
	for i := range many {
		many[i] = schema.NewHumanMessage("msg")
	}
	tier, err := classifier.Classify(ctx, many)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if tier != TierLarge {
		t.Errorf("many msgs tier = %q, want %q", tier, TierLarge)
	}

	// System prompt with long content → TierLarge.
	longSys := make([]byte, 2100)
	for i := range longSys {
		longSys[i] = 'a'
	}
	sysMsgs := []schema.Message{
		schema.NewSystemMessage(string(longSys)),
		schema.NewHumanMessage("hi"),
	}
	tier, err = classifier.Classify(ctx, sysMsgs)
	if err != nil {
		t.Fatalf("Classify: %v", err)
	}
	if tier != TierLarge {
		t.Errorf("system+long tier = %q, want %q", tier, TierLarge)
	}
}

func TestWithClassifier_AndFallbackOnError(t *testing.T) {
	models := []ModelConfig{
		{ID: "m-small", Tier: TierSmall, CostPerInputToken: 0.00001, CostPerOutputToken: 0.00002, Priority: 1},
		{ID: "m-medium", Tier: TierMedium, CostPerInputToken: 0.00005, CostPerOutputToken: 0.0001, Priority: 1},
	}

	// WithClassifier stub returning an error — router must fall back.
	router := NewCostRouter(
		WithModels(models...),
		WithClassifier(&stubClassifier{tier: TierSmall, err: context.Canceled}),
		WithFallbackTier(TierMedium),
	)
	sel, err := router.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("x")})
	if err != nil {
		t.Fatalf("SelectModel: %v", err)
	}
	if sel.Tier != TierMedium {
		t.Errorf("fallback Tier = %q, want %q", sel.Tier, TierMedium)
	}
}

func TestDefaultCostRouter_TierDowngrade(t *testing.T) {
	// Only a small model is configured; requesting a Large tier forces the
	// downgrade path via allModelsSorted.
	models := []ModelConfig{
		{ID: "only-small", Tier: TierSmall, CostPerInputToken: 0.00001, CostPerOutputToken: 0.00002, Priority: 1},
	}
	router := NewCostRouter(
		WithModels(models...),
		WithClassifier(&stubClassifier{tier: TierLarge}),
	)
	sel, err := router.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hello")})
	if err != nil {
		t.Fatalf("SelectModel: %v", err)
	}
	if sel.ModelID != "only-small" {
		t.Errorf("ModelID = %q, want only-small", sel.ModelID)
	}
	if !contains(sel.Reason, "downgraded") {
		t.Errorf("expected downgrade note in reason, got %q", sel.Reason)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

func TestEstimateTokens_EmptyMinimum(t *testing.T) {
	// Messages with no text content should fall to the minimum of 100.
	got := estimateTokens(nil)
	if got != 100 {
		t.Errorf("estimateTokens(nil) = %d, want 100", got)
	}
}

func TestRegistry_ListAndUnknown(t *testing.T) {
	if len(ListClassifiers()) == 0 {
		t.Error("expected at least one classifier registered")
	}
	if len(ListEnforcers()) == 0 {
		t.Error("expected at least one enforcer registered")
	}
	if _, err := NewClassifier("bogus", ClassifierConfig{}); err == nil {
		t.Error("expected error for unknown classifier")
	}
	if _, err := NewEnforcer("bogus", EnforcerConfig{}); err == nil {
		t.Error("expected error for unknown enforcer")
	}
}

func TestRegistry_DefaultRouterWithOptionalFields(t *testing.T) {
	// Exercise the default router factory branches where classifier,
	// enforcer, and fallback tier are all provided.
	r, err := NewRouter("default", RouterConfig{
		Models:       []ModelConfig{{ID: "m", Tier: TierSmall, CostPerInputToken: 0.1, CostPerOutputToken: 0.1}},
		Classifier:   &HeuristicClassifier{},
		Enforcer:     NewBudgetEnforcer(1000.0),
		FallbackTier: TierSmall,
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	sel, err := r.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
	if err != nil {
		t.Fatalf("SelectModel: %v", err)
	}
	if sel.ModelID != "m" {
		t.Errorf("ModelID = %q, want m", sel.ModelID)
	}
}
