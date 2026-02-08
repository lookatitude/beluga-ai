package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/hitl"
)

func TestLLMActivity(t *testing.T) {
	activity := LLMActivity(func(_ context.Context, prompt string) (string, error) {
		return "response: " + prompt, nil
	})

	result, err := activity(context.Background(), "hello")
	if err != nil {
		t.Fatalf("LLMActivity: %v", err)
	}
	if result != "response: hello" {
		t.Errorf("expected 'response: hello', got %v", result)
	}
}

func TestLLMActivity_WrongInputType(t *testing.T) {
	activity := LLMActivity(func(_ context.Context, prompt string) (string, error) {
		return prompt, nil
	})

	_, err := activity(context.Background(), 42)
	if err == nil {
		t.Fatal("expected error for wrong input type")
	}
}

func TestLLMActivity_Error(t *testing.T) {
	activity := LLMActivity(func(_ context.Context, _ string) (string, error) {
		return "", fmt.Errorf("llm error")
	})

	_, err := activity(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error from LLM")
	}
}

func TestToolActivity(t *testing.T) {
	activity := ToolActivity(func(_ context.Context, name string, args map[string]any) (any, error) {
		return fmt.Sprintf("executed %s with %v", name, args), nil
	})

	result, err := activity(context.Background(), map[string]any{
		"name": "calculator",
		"args": map[string]any{"expr": "2+2"},
	})
	if err != nil {
		t.Fatalf("ToolActivity: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestToolActivity_WrongInputType(t *testing.T) {
	activity := ToolActivity(func(_ context.Context, _ string, _ map[string]any) (any, error) {
		return nil, nil
	})

	_, err := activity(context.Background(), "not a map")
	if err == nil {
		t.Fatal("expected error for wrong input type")
	}
}

func TestToolActivity_MissingName(t *testing.T) {
	activity := ToolActivity(func(_ context.Context, _ string, _ map[string]any) (any, error) {
		return nil, nil
	})

	_, err := activity(context.Background(), map[string]any{"args": nil})
	if err == nil {
		t.Fatal("expected error for missing tool name")
	}
}

func TestHumanActivity(t *testing.T) {
	mgr := hitl.NewManager(hitl.WithTimeout(5 * time.Second))

	activity := HumanActivity(mgr)

	req := hitl.InteractionRequest{
		ID:       "human-test",
		Type:     hitl.TypeApproval,
		ToolName: "test_tool",
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		mgr.Respond(context.Background(), "human-test", hitl.InteractionResponse{
			Decision: hitl.DecisionApprove,
		})
	}()

	result, err := activity(context.Background(), req)
	if err != nil {
		t.Fatalf("HumanActivity: %v", err)
	}

	resp, ok := result.(*hitl.InteractionResponse)
	if !ok {
		t.Fatalf("expected *hitl.InteractionResponse, got %T", result)
	}
	if resp.Decision != hitl.DecisionApprove {
		t.Errorf("expected approve, got %s", resp.Decision)
	}
}

func TestHumanActivity_WrongInputType(t *testing.T) {
	mgr := hitl.NewManager()
	activity := HumanActivity(mgr)

	_, err := activity(context.Background(), "not a request")
	if err == nil {
		t.Fatal("expected error for wrong input type")
	}
}

func TestHumanActivity_Timeout(t *testing.T) {
	mgr := hitl.NewManager(hitl.WithTimeout(50 * time.Millisecond))
	activity := HumanActivity(mgr)

	req := hitl.InteractionRequest{
		ID:       "human-timeout",
		Type:     hitl.TypeApproval,
		ToolName: "test_tool",
	}

	_, err := activity(context.Background(), req)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
