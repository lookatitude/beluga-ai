package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

type failingEnforcer struct{ err error }

func (f *failingEnforcer) Check(context.Context, float64) (bool, error) { return false, f.err }
func (f *failingEnforcer) CheckAndReserve(context.Context, float64) (bool, error) {
	return false, f.err
}
func (f *failingEnforcer) Record(context.Context, float64) error { return nil }

// TestSelectModel_EnforcerErrorKeepsCode checks that a typed enforcer error is
// not re-coded as budget_exhausted (FWC-03 pattern within llm/).
func TestSelectModel_EnforcerErrorKeepsCode(t *testing.T) {
	inner := core.Errorf(core.ErrProviderDown, "budget store unreachable")
	r := NewCostRouter(
		WithModels(ModelConfig{ID: "m", Tier: TierSmall}),
		WithBudgetEnforcer(&failingEnforcer{err: inner}),
	)
	_, err := r.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
	var ce *core.Error
	if !errors.As(err, &ce) || ce.Code != core.ErrProviderDown {
		t.Fatalf("want provider_unavailable preserved, got %v", err)
	}

	r = NewCostRouter(
		WithModels(ModelConfig{ID: "m", Tier: TierSmall}),
		WithBudgetEnforcer(&failingEnforcer{err: errors.New("plain")}),
	)
	_, err = r.SelectModel(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
	if !errors.As(err, &ce) || ce.Code != core.ErrBudgetExhausted {
		t.Fatalf("want budget_exhausted for an untyped error, got %v", err)
	}
}
