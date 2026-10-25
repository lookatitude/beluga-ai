package hitl

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

func TestActivity(t *testing.T) {
	mgr := NewManager(WithTimeout(5 * time.Second))
	act := Activity(mgr)

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = mgr.Respond(context.Background(), "a1", InteractionResponse{Decision: DecisionApprove})
	}()
	out, err := act(context.Background(), InteractionRequest{ID: "a1", Type: TypeApproval, ToolName: "t"})
	if err != nil {
		t.Fatalf("Activity: %v", err)
	}
	if resp, ok := out.(*InteractionResponse); !ok || resp.Decision != DecisionApprove {
		t.Fatalf("result = %#v", out)
	}

	if _, err := act(context.Background(), "nope"); core.CodeOf(err, "") != core.ErrInvalidInput {
		t.Errorf("wrong input: err = %v", err)
	}

	short := Activity(NewManager(WithTimeout(10 * time.Millisecond)))
	_, err = short(context.Background(), InteractionRequest{ID: "a2", Type: TypeApproval, ToolName: "t"})
	if err == nil || core.CodeOf(err, "") != core.ErrTimeout {
		t.Errorf("timeout: err = %v, want timeout code preserved", err)
	}
}
