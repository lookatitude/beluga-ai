// docs/examples/agents/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/agents"
	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/llms/ollama"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/prompts"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools/gofunc"
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
	 	 config.Cfg.LLMs.Ollama.Model,
	 	 ollama.WithHost(config.Cfg.LLMs.Ollama.BaseURL),
	 )
	 if err != nil {
	 	 log.Fatalf("Failed to create Ollama client: %v", err)
	 }

	 // 2. Define Tools
	 weatherTool, err := gofunc.NewGoFunctionTool(
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
	 // Create an in-memory chat history
	 chatHistory := &inMemoryChatHistory{messages: []schema.Message{}}
	 mem := memory.NewChatMessageBufferMemory(chatHistory)

	 // 4. Create Agent Executor
	 // Using the basic Executor for this example
	 // Create agent that implements the Agent interface
	 // Create a prompt template for the agent
	 templateString := "You are a helpful assistant. Use the following tools to answer the user's question: {{.agent_scratchpad}}\nQuestion: {{.input}}"
	 promptTemplate, err := prompts.NewStringPromptTemplate(templateString)
	 if err != nil {
	 	 log.Fatalf("Failed to create prompt template: %v", err)
	 }
	 
	 // Create the agent
	 agent, err := agents.NewReActAgent(llm, agentTools, promptTemplate)
	 if err != nil {
	 	 log.Fatalf("Failed to create agent: %v", err)
	 }
	 
	 // Create the executor
	 agentExecutor, err := agents.NewAgentExecutor(
	 	 agent,
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

	 result, err := agentExecutor.Invoke(ctx, input)
	 if err != nil {
	 	 log.Printf("Agent execution failed: %v", err)
	 } else {
	 	 fmt.Printf("\nAgent Final Output: %s\n", result)
	 }

	 // --- Inspect Memory --- 
	 fmt.Println("\n--- Agent Memory --- ")
	 memVars, _ := mem.LoadMemoryVariables(ctx, nil)
	 history, _ := memVars[mem.MemoryKey].([]schema.Message)
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

// Simple in-memory implementation of ChatMessageHistory
type inMemoryChatHistory struct {
	messages []schema.Message
}

func (h *inMemoryChatHistory) AddMessage(ctx context.Context, message schema.Message) error {
	h.messages = append(h.messages, message)
	return nil
}

func (h *inMemoryChatHistory) AddUserMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewHumanMessage(content))
}

func (h *inMemoryChatHistory) AddAIMessage(ctx context.Context, content string) error {
	return h.AddMessage(ctx, schema.NewAIMessage(content))
}

func (h *inMemoryChatHistory) GetMessages(ctx context.Context) ([]schema.Message, error) {
	return h.messages, nil
}

func (h *inMemoryChatHistory) Clear(ctx context.Context) error {
	h.messages = []schema.Message{}
	return nil
}

