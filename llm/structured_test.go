package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

type testPerson struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestStructuredOutput_Generate_Success(t *testing.T) {
	person := testPerson{Name: "Alice", Age: 30}
	jsonBytes, _ := json.Marshal(person)

	model := &stubModel{
		id: "structured-test",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage(string(jsonBytes)), nil
		},
	}

	so := NewStructured[testPerson](model)
	result, err := so.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("test")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Alice" {
		t.Errorf("Name = %q, want %q", result.Name, "Alice")
	}
	if result.Age != 30 {
		t.Errorf("Age = %d, want %d", result.Age, 30)
	}
}

func TestStructuredOutput_Generate_RetriesOnInvalidJSON(t *testing.T) {
	calls := 0
	model := &stubModel{
		id: "retry-test",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			calls++
			if calls == 1 {
				return schema.NewAIMessage("not valid json"), nil
			}
			return schema.NewAIMessage(`{"name":"Bob","age":25}`), nil
		},
	}

	so := NewStructured[testPerson](model, WithMaxRetries(2))
	result, err := so.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("test")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Bob" {
		t.Errorf("Name = %q, want %q", result.Name, "Bob")
	}
	if calls != 2 {
		t.Errorf("expected 2 Generate calls, got %d", calls)
	}
}

func TestStructuredOutput_Generate_ExhaustsRetries(t *testing.T) {
	model := &stubModel{
		id: "fail-test",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return schema.NewAIMessage("still not json"), nil
		},
	}

	so := NewStructured[testPerson](model, WithMaxRetries(1))
	_, err := so.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("test")})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}

func TestStructuredOutput_Generate_ModelError(t *testing.T) {
	modelErr := context.DeadlineExceeded
	model := &stubModel{
		id: "error-test",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			return nil, modelErr
		},
	}

	so := NewStructured[testPerson](model)
	_, err := so.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("test")})
	if err == nil {
		t.Fatal("expected model error to propagate")
	}
}

func TestStructuredOutput_Schema(t *testing.T) {
	model := &stubModel{id: "schema-test"}
	so := NewStructured[testPerson](model)
	s := so.Schema()
	if s == nil {
		t.Fatal("Schema() returned nil")
	}
	if s["type"] != "object" {
		t.Errorf("schema type = %v, want %q", s["type"], "object")
	}
	props, ok := s["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties is not map[string]any: %T", s["properties"])
	}
	if _, ok := props["name"]; !ok {
		t.Error("schema missing 'name' property")
	}
	if _, ok := props["age"]; !ok {
		t.Error("schema missing 'age' property")
	}
}

func TestStructuredOutput_DefaultRetries(t *testing.T) {
	calls := 0
	model := &stubModel{
		id: "default-retry",
		generateFn: func(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
			calls++
			return schema.NewAIMessage("bad"), nil
		},
	}

	so := NewStructured[testPerson](model) // default maxRetries=2
	_, _ = so.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("test")})
	// Default maxRetries=2 means 3 total attempts (0, 1, 2).
	if calls != 3 {
		t.Errorf("expected 3 calls with default retries, got %d", calls)
	}
}
