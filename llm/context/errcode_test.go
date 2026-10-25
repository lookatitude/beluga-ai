package context

import (
	gocontext "context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
)

type failingStep struct{ err error }

func (s *failingStep) Process(gocontext.Context, []ContextItem) ([]ContextItem, error) {
	return nil, s.err
}
func (s *failingStep) Name() string { return "failing" }

// TestExecute_StepErrorKeepsCode checks that a typed step error is not
// re-coded as invalid_input.
func TestExecute_StepErrorKeepsCode(t *testing.T) {
	p := NewPipeline(WithStep(&failingStep{err: core.Errorf(core.ErrTimeout, "slow")}))
	_, err := p.Execute(gocontext.Background(), PipelineInput{})
	var ce *core.Error
	if !errors.As(err, &ce) || ce.Code != core.ErrTimeout {
		t.Fatalf("want timeout preserved, got %v", err)
	}

	p = NewPipeline(WithStep(&failingStep{err: errors.New("plain")}))
	_, err = p.Execute(gocontext.Background(), PipelineInput{})
	if !errors.As(err, &ce) || ce.Code != core.ErrInvalidInput {
		t.Fatalf("want invalid_input for an untyped error, got %v", err)
	}
}
