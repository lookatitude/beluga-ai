package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/orchestration"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	fmt.Println("🔗 Beluga AI - Chain Orchestration Example")
	fmt.Println("===========================================")

	ctx := context.Background()

	// Step 1: Create LLM for the chain steps
	llm, err := createLLM(ctx)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Step 2: Create chain steps
	// Each step is a Runnable that processes input and produces output
	steps := []core.Runnable{
		&preprocessingStep{name: "preprocessor"},
		&llmStep{llm: llm, name: "llm-processor"},
		&postprocessingStep{name: "postprocessor"},
	}

	fmt.Printf("✅ Created chain with %d steps\n", len(steps))

	// Step 3: Create a chain
	chain, err := orchestration.NewChain(steps)
	if err != nil {
		log.Fatalf("Failed to create chain: %v", err)
	}

	// Step 4: Execute the chain
	fmt.Println("\n🚀 Executing chain...")
	input := map[string]interface{}{
		"input": "Hello, how are you?",
	}

	result, err := chain.Invoke(ctx, input)
	if err != nil {
		log.Fatalf("Chain execution failed: %v", err)
	}

	// Step 5: Display the result
	fmt.Printf("\n✅ Chain Result:\n%v\n", result)

	// Step 6: Demonstrate batch processing
	fmt.Println("\n📦 Executing chain in batch mode...")
	inputs := []interface{}{
		map[string]interface{}{"input": "First message"},
		map[string]interface{}{"input": "Second message"},
		map[string]interface{}{"input": "Third message"},
	}

	results, err := chain.Batch(ctx, inputs)
	if err != nil {
		log.Fatalf("Batch execution failed: %v", err)
	}

	fmt.Printf("✅ Processed %d inputs:\n", len(results))
	for i, res := range results {
		fmt.Printf("  %d. %v\n", i+1, res)
	}

	fmt.Println("\n✨ Chain orchestration example completed successfully!")
}

// preprocessingStep is a simple preprocessing step
type preprocessingStep struct {
	name string
}

func (s *preprocessingStep) Invoke(ctx context.Context, input interface{}, options ...core.Option) (interface{}, error) {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("preprocessing step expects map[string]interface{}")
	}

	text, ok := inputMap["input"].(string)
	if !ok {
		return nil, fmt.Errorf("input field must be a string")
	}

	// Preprocess: convert to uppercase and add prefix
	processed := fmt.Sprintf("PREPROCESSED: %s", strings.ToUpper(text))
	return map[string]interface{}{
		"input":     text,
		"processed": processed,
	}, nil
}

func (s *preprocessingStep) Batch(ctx context.Context, inputs []interface{}, options ...core.Option) ([]interface{}, error) {
	results := make([]interface{}, len(inputs))
	for i, input := range inputs {
		result, err := s.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (s *preprocessingStep) Stream(ctx context.Context, input interface{}, options ...core.Option) (<-chan interface{}, error) {
	result, err := s.Invoke(ctx, input, options...)
	if err != nil {
		return nil, err
	}
	ch := make(chan interface{}, 1)
	ch <- result
	close(ch)
	return ch, nil
}

// llmStep processes input using an LLM
type llmStep struct {
	llm  interface{}
	name string
}

func (s *llmStep) Invoke(ctx context.Context, input interface{}, options ...core.Option) (interface{}, error) {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("llm step expects map[string]interface{}")
	}

	processed, ok := inputMap["processed"].(string)
	if !ok {
		processed, _ = inputMap["input"].(string)
	}

	// Create messages for LLM
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant."),
		schema.NewHumanMessage(processed),
	}

	// Call LLM (using mock for demonstration)
	response := fmt.Sprintf("LLM Response to: %s", processed)

	return map[string]interface{}{
		"input":    inputMap["input"],
		"processed": processed,
		"response":  response,
	}, nil
}

func (s *llmStep) Batch(ctx context.Context, inputs []interface{}, options ...core.Option) ([]interface{}, error) {
	results := make([]interface{}, len(inputs))
	for i, input := range inputs {
		result, err := s.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (s *llmStep) Stream(ctx context.Context, input interface{}, options ...core.Option) (<-chan interface{}, error) {
	result, err := s.Invoke(ctx, input, options...)
	if err != nil {
		return nil, err
	}
	ch := make(chan interface{}, 1)
	ch <- result
	close(ch)
	return ch, nil
}

// postprocessingStep is a simple postprocessing step
type postprocessingStep struct {
	name string
}

func (s *postprocessingStep) Invoke(ctx context.Context, input interface{}, options ...core.Option) (interface{}, error) {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("postprocessing step expects map[string]interface{}")
	}

	response, ok := inputMap["response"].(string)
	if !ok {
		return nil, fmt.Errorf("response field must be a string")
	}

	// Postprocess: add suffix
	final := fmt.Sprintf("%s [POSTPROCESSED]", response)
	return map[string]interface{}{
		"output": final,
	}, nil
}

func (s *postprocessingStep) Batch(ctx context.Context, inputs []interface{}, options ...core.Option) ([]interface{}, error) {
	results := make([]interface{}, len(inputs))
	for i, input := range inputs {
		result, err := s.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (s *postprocessingStep) Stream(ctx context.Context, input interface{}, options ...core.Option) (<-chan interface{}, error) {
	result, err := s.Invoke(ctx, input, options...)
	if err != nil {
		return nil, err
	}
	ch := make(chan interface{}, 1)
	ch <- result
	close(ch)
	return ch, nil
}

// createLLM creates an LLM instance (mock for demonstration)
func createLLM(ctx context.Context) (interface{}, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENAI_API_KEY not set, using mock LLM")
		return &mockLLM{}, nil
	}

	config := llms.NewConfig(
		llms.WithProvider("openai"),
		llms.WithModelName("gpt-3.5-turbo"),
		llms.WithAPIKey(apiKey),
	)

	factory := llms.NewFactory()
	llm, err := factory.CreateChatModel("openai", config)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	return llm, nil
}

// mockLLM is a simple mock implementation
type mockLLM struct{}

func (m *mockLLM) Generate(ctx context.Context, messages []interface{}) (interface{}, error) {
	return &mockMessage{content: "Mock LLM response"}, nil
}

type mockMessage struct {
	content string
}

func (m *mockMessage) GetContent() string {
	return m.content
}
