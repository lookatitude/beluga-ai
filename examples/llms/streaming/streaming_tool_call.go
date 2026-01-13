// Package streaming demonstrates streaming LLM calls with tool calling in Beluga AI.
// This example shows how to build responsive AI applications that stream responses
// while executing tools mid-conversation.
//
// Key patterns demonstrated:
//   - Streaming LLM responses via channels
//   - Tool binding and execution
//   - OTEL instrumentation for observability
//   - Error handling for streaming scenarios
//   - Context propagation for cancellation
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/gofunc"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// We define a tracer for observability - this helps debug in production
// The naming follows Beluga AI conventions: beluga.{package}.{component}
var tracer = otel.Tracer("beluga.llms.streaming.example")

// StreamingResult holds the result of a streaming session
type StreamingResult struct {
	ResponseText   string        // Final text response
	ToolsCalled    []string      // Names of tools that were called
	TotalDuration  time.Duration // Total time from start to finish
	ChunksReceived int           // Number of chunks received
}

// StreamingToolCallExample demonstrates streaming LLM calls with tool calling.
// It creates an LLM client, binds tools, and processes a streaming response
// that may include tool calls.
type StreamingToolCallExample struct {
	client iface.ChatModel
	tools  []tools.Tool
}

// NewStreamingToolCallExample creates a new example with the given LLM client and tools.
// We use dependency injection here - pass in the client rather than creating it internally.
// This makes the code testable with mock clients.
func NewStreamingToolCallExample(client iface.ChatModel, availableTools []tools.Tool) *StreamingToolCallExample {
	return &StreamingToolCallExample{
		client: client,
		tools:  availableTools,
	}
}

// Run executes the streaming example with the given prompt.
// It handles the full lifecycle: streaming, tool detection, tool execution,
// and continuation of the response.
func (e *StreamingToolCallExample) Run(ctx context.Context, prompt string) (*StreamingResult, error) {
	// Start a span for the entire operation - this groups all sub-operations together
	ctx, span := tracer.Start(ctx, "streaming.run",
		trace.WithAttributes(
			attribute.String("prompt", prompt),
			attribute.Int("tools_count", len(e.tools)),
		))
	defer span.End()

	start := time.Now()
	result := &StreamingResult{}

	// Bind tools to the model - this returns a new model instance with tools attached.
	// The LLM will see these tool definitions and can choose to call them.
	modelWithTools := e.client.BindTools(e.tools)

	// Create initial messages for the conversation
	messages := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	// Process the initial stream - this may include text and/or tool calls
	responseText, toolCalls, chunks, err := e.processStream(ctx, modelWithTools, messages)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("initial stream failed: %w", err)
	}

	result.ResponseText = responseText
	result.ChunksReceived = chunks

	// If the LLM requested tool calls, we need to execute them and continue
	if len(toolCalls) > 0 {
		span.AddEvent("tool_calls_detected", trace.WithAttributes(
			attribute.Int("tool_call_count", len(toolCalls)),
		))

		// Execute all requested tool calls
		toolResults, toolNames, err := e.executeToolCalls(ctx, toolCalls)
		if err != nil {
			// We log the error but continue - partial tool results are still useful
			span.AddEvent("tool_execution_partial_failure", trace.WithAttributes(
				attribute.String("error", err.Error()),
			))
			log.Printf("Some tool calls failed (continuing anyway): %v", err)
		}

		result.ToolsCalled = toolNames

		// Build the continuation messages:
		// 1. AI's response (which may be empty if it went straight to tools)
		// 2. Tool results for each tool call
		messages = append(messages, schema.NewAIMessage(responseText))
		messages = append(messages, toolResults...)

		// Continue the conversation - the LLM now has tool results and can respond
		finalText, _, finalChunks, err := e.processStream(ctx, modelWithTools, messages)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("continuation stream failed: %w", err)
		}

		result.ResponseText = finalText
		result.ChunksReceived += finalChunks
	}

	result.TotalDuration = time.Since(start)

	// Record final metrics for observability
	span.SetAttributes(
		attribute.Int("total_chunks", result.ChunksReceived),
		attribute.Int("tools_called", len(result.ToolsCalled)),
		attribute.Float64("duration_ms", float64(result.TotalDuration.Milliseconds())),
		attribute.Int("response_length", len(result.ResponseText)),
	)
	span.SetStatus(codes.Ok, "streaming completed successfully")

	return result, nil
}

// processStream handles a single streaming session.
// It reads chunks from the channel, accumulates text, and collects tool calls.
// Returns the accumulated text, any tool calls, chunk count, and any error.
func (e *StreamingToolCallExample) processStream(
	ctx context.Context,
	model iface.ChatModel,
	messages []schema.Message,
) (string, []schema.ToolCallChunk, int, error) {
	ctx, span := tracer.Start(ctx, "streaming.process_stream")
	defer span.End()

	// Start streaming - this initiates the request and returns a channel
	streamChan, err := model.StreamChat(ctx, messages)
	if err != nil {
		span.RecordError(err)
		return "", nil, 0, fmt.Errorf("failed to start stream: %w", err)
	}

	var responseText strings.Builder
	var toolCalls []schema.ToolCallChunk
	chunks := 0

	// Process each chunk as it arrives
	for chunk := range streamChan {
		chunks++

		// Always check for errors first - they can arrive at any point
		if chunk.Err != nil {
			span.RecordError(chunk.Err)
			span.SetStatus(codes.Error, "stream error")
			return responseText.String(), toolCalls, chunks, fmt.Errorf("stream error: %w", chunk.Err)
		}

		// Accumulate text content
		// In a real application, you might print this directly for a "typing" effect
		if chunk.Content != "" {
			responseText.WriteString(chunk.Content)
		}

		// Collect tool call chunks
		// Note: Tool calls may arrive across multiple chunks, so we accumulate them
		if len(chunk.ToolCallChunks) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCallChunks...)
		}
	}

	span.SetAttributes(
		attribute.Int("chunks_received", chunks),
		attribute.Int("tool_calls_count", len(toolCalls)),
		attribute.Int("response_length", responseText.Len()),
	)

	return responseText.String(), toolCalls, chunks, nil
}

// executeToolCalls executes all pending tool calls and returns the results.
// It returns tool result messages, the names of tools called, and any error.
// Note: If some tools fail, we still return partial results. This allows the
// LLM to work with what succeeded and potentially handle failures gracefully.
func (e *StreamingToolCallExample) executeToolCalls(
	ctx context.Context,
	toolCalls []schema.ToolCallChunk,
) ([]schema.Message, []string, error) {
	ctx, span := tracer.Start(ctx, "streaming.execute_tools")
	defer span.End()

	// Build a lookup map for efficient tool access
	toolMap := make(map[string]tools.Tool)
	for _, t := range e.tools {
		toolMap[t.Name()] = t
	}

	var results []schema.Message
	var toolNames []string
	var lastErr error

	for _, tc := range toolCalls {
		toolNames = append(toolNames, tc.Name)

		// Find the tool - if unknown, send an error result back to LLM
		tool, ok := toolMap[tc.Name]
		if !ok {
			lastErr = fmt.Errorf("unknown tool: %s", tc.Name)
			results = append(results, schema.NewToolMessage(
				fmt.Sprintf("Error: unknown tool %s", tc.Name),
				tc.ID,
			))
			continue
		}

		// Parse the JSON arguments
		var args map[string]any
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			lastErr = fmt.Errorf("failed to parse args for %s: %w", tc.Name, err)
			results = append(results, schema.NewToolMessage(
				fmt.Sprintf("Error: invalid arguments - %v", err),
				tc.ID,
			))
			continue
		}

		// Execute the tool with context (for cancellation support)
		toolResult, err := tool.Execute(ctx, args)
		if err != nil {
			lastErr = fmt.Errorf("tool %s failed: %w", tc.Name, err)
			results = append(results, schema.NewToolMessage(
				fmt.Sprintf("Error: tool execution failed - %v", err),
				tc.ID,
			))
			continue
		}

		// Success - add the result
		resultStr := fmt.Sprintf("%v", toolResult)
		results = append(results, schema.NewToolMessage(resultStr, tc.ID))
	}

	span.SetAttributes(
		attribute.Int("tools_executed", len(toolCalls)),
		attribute.StringSlice("tool_names", toolNames),
		attribute.Bool("had_errors", lastErr != nil),
	)

	return results, toolNames, lastErr
}

// createWeatherTool creates a sample weather tool for demonstration.
// In production, this would call a real weather API.
func createWeatherTool() tools.Tool {
	tool, _ := gofunc.NewGoFunctionTool(
		"get_weather",
		"Get the current weather for a location. Returns temperature and conditions.",
		`{"type": "object", "properties": {"location": {"type": "string", "description": "The city and state, e.g. 'San Francisco, CA'"}}, "required": ["location"]}`,
		func(ctx context.Context, args map[string]any) (any, error) {
			location, ok := args["location"].(string)
			if !ok || location == "" {
				return "", fmt.Errorf("location is required")
			}

			// In production, call a real weather API here
			// For this example, we return mock data
			weather := map[string]any{
				"location":   location,
				"temp_f":     72,
				"temp_c":     22,
				"conditions": "sunny",
				"humidity":   45,
			}

			data, err := json.Marshal(weather)
			if err != nil {
				return "", fmt.Errorf("failed to serialize weather data: %w", err)
			}

			return string(data), nil
		},
	)
	return tool
}

// createCalculatorTool creates a sample calculator tool for demonstration.
func createCalculatorTool() tools.Tool {
	tool, _ := gofunc.NewGoFunctionTool(
		"calculator",
		"Perform basic arithmetic calculations. Supports +, -, *, / operations.",
		`{"type": "object", "properties": {"expression": {"type": "string", "description": "The arithmetic expression to evaluate, e.g. '2 + 2'"}}, "required": ["expression"]}`,
		func(ctx context.Context, args map[string]any) (any, error) {
			expression, ok := args["expression"].(string)
			if !ok || expression == "" {
				return "", fmt.Errorf("expression is required")
			}

			// In production, use a proper expression parser
			// This is a simplified example
			return fmt.Sprintf(`{"expression": "%s", "result": "calculated result here"}`, expression), nil
		},
	)
	return tool
}

func main() {
	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create the LLM client with OpenAI
	// You can swap this for Anthropic, Ollama, etc. - the streaming pattern is the same
	client, err := llms.NewOpenAIChat(
		llms.WithAPIKey(apiKey),
		llms.WithModelName("gpt-4"),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	// Create available tools
	availableTools := []tools.Tool{
		createWeatherTool(),
		createCalculatorTool(),
	}

	// Create and run the example
	example := NewStreamingToolCallExample(client, availableTools)

	fmt.Println("=== Streaming LLM with Tool Calls Example ===")
	fmt.Println()

	// Example 1: A prompt that should trigger a tool call
	fmt.Println("Prompt: What's the weather like in San Francisco right now?")
	fmt.Println()

	result, err := example.Run(ctx, "What's the weather like in San Francisco right now?")
	if err != nil {
		log.Fatalf("Example failed: %v", err)
	}

	fmt.Printf("Response: %s\n", result.ResponseText)
	fmt.Printf("Tools called: %v\n", result.ToolsCalled)
	fmt.Printf("Chunks received: %d\n", result.ChunksReceived)
	fmt.Printf("Duration: %v\n", result.TotalDuration)
	fmt.Println()

	// Example 2: A prompt without tool calls
	fmt.Println("Prompt: What's 2 + 2?")
	fmt.Println()

	result2, err := example.Run(ctx, "What's 2 + 2? Answer briefly.")
	if err != nil {
		log.Fatalf("Example failed: %v", err)
	}

	fmt.Printf("Response: %s\n", result2.ResponseText)
	fmt.Printf("Tools called: %v\n", result2.ToolsCalled)
	fmt.Printf("Chunks received: %d\n", result2.ChunksReceived)
	fmt.Printf("Duration: %v\n", result2.TotalDuration)
}
