// docs/examples/agents/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/agents"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llms/ollama"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
	"github.com/lookatitude/beluga-ai/tools/gofunc"
)

// --- Tool Definition (from tools example) ---
func getCurrentWeather(location string, unit string) (string, error) {
	// Mock implementation
	 if location == "London" {
	 	 if unit == "celsius" {
	 	 	 return `{"temperature": 15, "unit": "celsius", "description": "Cloudy"}`, nil
	 	 }
	 	 return `{"temperature": 59, "unit": "fahrenheit", "description": "Cloudy"}`, nil
	 }
	 if location == "Tokyo" {
	 	 if unit == "celsius" {
	 	 	 return `{"temperature": 22, "unit": "celsius", "description": "Sunny"}`, nil
	 	 }
	 	 return `{"temperature": 72, "unit": "fahrenheit", "description": "Sunny"}`, nil
	 }
	 return `{"error": "Location not found"}`, fmt.Errorf("location not found: %s", location)
}

var weatherSchema = `{ 
    "type": "object",
    "properties": {
        "location": {
            "type": "string",
            "description": "The city and state, e.g. San Francisco, CA"
        },
        "unit": { 
            "type": "string", 
            "enum": ["celsius", "fahrenheit"],
            "description": "The temperature unit to use. Infer this from the user's location."
        }
    },
    "required": ["location", "unit"]
}`

// --- Main Agent Example --- 
func main() {
	 ctx := context.Background()

	 // Load configuration
	 err := config.LoadConfig()
	 if err != nil {
	 	 log.Fatalf("Failed to load configuration: %v", err)
	 }

	 // 1. Initialize LLM (using Ollama for local testing)
	 llm, err := ollama.NewOllamaChat(
	 	 ollama.WithOllamaBaseURL(config.Cfg.LLMs.Ollama.BaseURL),
	 	 ollama.WithOllamaModel(config.Cfg.LLMs.Ollama.Model),
	 )
	 if err != nil {
	 	 log.Fatalf("Failed to create Ollama client: %v", err)
	 }

	 // 2. Define Tools
	 weatherTool, err := gofunc.NewGoFuncTool(
	 	 "get_current_weather",
	 	 "Get the current weather in a given location",
	 	 weatherSchema,
	 	 getCurrentWeather,
	 )
	 if err != nil {
	 	 log.Fatalf("Failed to create weather tool: %v", err)
	 }
	 agentTools := []tools.Tool{weatherTool}

	 // 3. Initialize Memory
	 mem := memory.NewBufferMemory()

	 // 4. Create Agent Executor
	 // Using the basic Executor for this example
	 agentExecutor, err := agents.NewExecutor(
	 	 llm, 
	 	 agentTools,
	 	 agents.WithMemory(mem),
	 	 agents.WithMaxIterations(5), // Limit agent loops
	 	 // agents.WithVerbose(true), // Uncomment for detailed logging
	 )
	 if err != nil {
	 	 log.Fatalf("Failed to create agent executor: %v", err)
	 }

	 // 5. Run the Agent
	 fmt.Println("--- Running Agent Executor ---")
	 input := "What's the weather like in London in Celsius?"
	 fmt.Printf("User Input: %s\n", input)

	 result, err := agentExecutor.Run(ctx, input)
	 if err != nil {
	 	 log.Printf("Agent execution failed: %v", err)
	 } else {
	 	 fmt.Printf("\nAgent Final Output: %s\n", result)
	 }

	 // --- Inspect Memory --- 
	 fmt.Println("\n--- Agent Memory --- ")
	 memVars, _ := mem.LoadMemoryVariables(ctx, nil)
	 history, _ := memVars[mem.MemoryKey()].([]schema.Message)
	 for _, msg := range history {
	 	 fmt.Printf(" - %s: %s", msg.GetType(), msg.GetContent())
	 	 // Print tool calls/results if they exist
	 	 if aiMsg, ok := msg.(*schema.AIMessage); ok && len(aiMsg.ToolCalls) > 0 {
	 	 	 tcJSON, _ := json.Marshal(aiMsg.ToolCalls)
	 	 	 fmt.Printf(" [Tool Calls: %s]", string(tcJSON))
	 	 }
	 	 if toolMsg, ok := msg.(*schema.ToolMessage); ok {
	 	 	 fmt.Printf(" [Tool Call ID: %s]", toolMsg.ToolCallID)
	 	 }
	 	 fmt.Println()
	 }

	 // Note: ReAct agent example would involve creating a ReAct agent instance
	 // and potentially a different prompt structure.
}

