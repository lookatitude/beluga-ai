// docs/examples/llms/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/anthropic"
	"github.com/lookatitude/beluga-ai/pkg/llms/bedrock"
	"github.com/lookatitude/beluga-ai/pkg/llms/gemini"
	"github.com/lookatitude/beluga-ai/pkg/llms/ollama"
	"github.com/lookatitude/beluga-ai/pkg/llms/openai"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

func printUsage(providerName string, aiMsg schema.Message) {
	if aiMsg == nil {
		fmt.Printf("%s Token Usage: Message was nil\n", providerName)
		return
	}
	usage, ok := aiMsg.GetAdditionalArgs()["usage"].(map[string]int)
	if ok {
		fmt.Printf("%s Token Usage: Input=%d, Output=%d, Total=%d\n",
			providerName,
			usage["input_tokens"],
			usage["output_tokens"],
			usage["total_tokens"],
		)
	} else {
		usageAny, okAny := aiMsg.GetAdditionalArgs()["usage"].(map[string]any)
		if okAny {
			inputTokens, inputOk := usageAny["input_tokens"].(int)
			outputTokens, outputOk := usageAny["output_tokens"].(int)
			totalTokens, totalOk := usageAny["total_tokens"].(int)
			if inputOk && outputOk && totalOk {
				fmt.Printf("%s Token Usage (from map[string]any): Input=%d, Output=%d, Total=%d\n",
					providerName,
					inputTokens,
					outputTokens,
					totalTokens,
				)
			} else {
				fmt.Printf("%s Token Usage: N/A (structure mismatch in map[string]any)\n", providerName)
			}
		} else {
			note, noteOk := aiMsg.GetAdditionalArgs()["usage_note"].(string)
			if noteOk {
				fmt.Printf("%s Token Usage: %s\n", providerName, note)
			} else {
				fmt.Printf("%s Token Usage: N/A\n", providerName)
			}
		}
	}
	if stopReason, ok := aiMsg.GetAdditionalArgs()["finish_reason"].(string); ok {
		fmt.Printf("%s Stop Reason: %s\n", providerName, stopReason)
	}
}

// CalculatorTool implements a simple calculator that can perform basic arithmetic operations
type CalculatorTool struct {
	name        string
	description string
	schema      string
}

// Definition returns the tool definition for the calculator
func (t *CalculatorTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        t.name,
		Description: t.description,
		InputSchema: t.schema,
	}
}

// Execute performs the calculation based on the operation and arguments
func (t *CalculatorTool) Execute(ctx context.Context, input any) (any, error) {
	// Convert input to map[string]interface{}
	var args map[string]interface{}
	
	switch v := input.(type) {
	case map[string]interface{}:
		args = v
	default:
		// Try to marshal and unmarshal to get a map
		bytes, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal input: %v", err)
		}
		
		if err := json.Unmarshal(bytes, &args); err != nil {
			return nil, fmt.Errorf("failed to convert input to map: %v", err)
		}
	}
	
	// Extract operation and operands
	operation, _ := args["operation"].(string)
	a, _ := args["a"].(float64)
	b, _ := args["b"].(float64)
	
	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
	
	return fmt.Sprintf("Result: %.2f", result), nil
}

// Description returns a description of the calculator tool
func (t *CalculatorTool) Description() string {
	return t.description
}

// Name returns the name of the calculator tool
func (t *CalculatorTool) Name() string {
	return t.name
}

// Batch implements the tools.Tool interface for batch processing
func (t *CalculatorTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	
	for i, input := range inputs {
		// Convert input to map[string]interface{} for Execute
		argsMap, ok := input.(map[string]interface{})
		if !ok {
			// Try to convert via JSON if it's not already the right type
			bytes, err := json.Marshal(input)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal input at index %d: %v", i, err)
			}
			
			var convertedMap map[string]interface{}
			if err := json.Unmarshal(bytes, &convertedMap); err != nil {
				return nil, fmt.Errorf("failed to unmarshal input at index %d: %v", i, err)
			}
			argsMap = convertedMap
		}
		
		result, err := t.Execute(ctx, argsMap)
		if err != nil {
			return nil, fmt.Errorf("failed to execute tool at index %d: %v", i, err)
		}
		results[i] = result
	}
	
	return results, nil
}

func main() {
	err := config.LoadConfig() // Load configuration first
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()

	// --- OpenAI Example --- 
	fmt.Println("--- OpenAI Example ---")
	openAIClient, err := openai.NewOpenAIChat("gpt-3.5-turbo")
	if err != nil {
		log.Printf("Failed to create OpenAI client: %v", err)
	} else {
		messages := []schema.Message{schema.NewHumanMessage("Tell me a short joke about Go.")}
		aiMsg, err := openAIClient.Generate(ctx, messages, llms.WithMaxTokens(50))
		if err != nil {
			log.Printf("OpenAI Generate failed: %v", err)
		} else {
			fmt.Printf("OpenAI Response: %s\n", aiMsg.GetContent())
			printUsage("OpenAI", aiMsg)
		}
	}

	// --- Anthropic Example --- 
	fmt.Println("\n--- Anthropic Example ---")
	anthropicClient, err := anthropic.NewAnthropicChat(
		anthropic.WithAnthropicModel("claude-3-haiku-20240307"),
	)
	if err != nil {
		log.Printf("Failed to create Anthropic client: %v", err)
	} else {
		// Basic chat completion
		fmt.Println("1. Basic Chat Completion:")
		messages := []schema.Message{
			schema.NewSystemMessage("You are a helpful AI assistant that provides concise explanations."),
			schema.NewHumanMessage("Explain the concept of recursion briefly."),
		}
		aiMsg, err := anthropicClient.Generate(ctx, messages, llms.WithMaxTokens(100))
		if err != nil {
			log.Printf("Anthropic Generate failed: %v", err)
		} else {
			fmt.Printf("Anthropic Response: %s\n", aiMsg.GetContent())
			printUsage("Anthropic", aiMsg)
		}

		// Streaming example
		fmt.Println("\n2. Anthropic Streaming Example:")
		streamMessages := []schema.Message{
			schema.NewHumanMessage("Write a very short poem about AI."),
		}
		streamChan, streamErr := anthropicClient.StreamChat(ctx, streamMessages, llms.WithMaxTokens(75))
		if streamErr != nil {
			log.Printf("Anthropic StreamChat failed: %v", streamErr)
		} else {
			var fullStreamedResponse strings.Builder
			var lastChunk llms.AIMessageChunk
			for chunk := range streamChan {
				if chunk.Err != nil {
					log.Printf("Anthropic stream error: %v", chunk.Err)
					break
				}
				fmt.Print(chunk.Content)
				fullStreamedResponse.WriteString(chunk.Content)
				lastChunk = chunk
			}
			fmt.Println()
			if lastChunk.AdditionalArgs != nil {
				if usage, ok := lastChunk.AdditionalArgs["usage"].(map[string]int); ok {
					fmt.Printf("Anthropic Stream Token Usage (from last chunk): Input=%d, Output=%d, Total=%d\n",
						usage["input_tokens"], usage["output_tokens"], usage["total_tokens"],
					)
				}
				if stopReason, ok := lastChunk.AdditionalArgs["finish_reason"].(string); ok {
					fmt.Printf("Anthropic Stream Stop Reason: %s\n", stopReason)
				}
			}
		}
		
		// Tool usage example
		fmt.Println("\n3. Anthropic Tool Usage Example:")
		// Define a simple calculator tool
		calcToolSchema := `{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"enum": ["add", "subtract", "multiply", "divide"],
					"description": "The mathematical operation to perform"
				},
				"a": {
					"type": "number",
					"description": "The first operand"
				},
				"b": {
					"type": "number",
					"description": "The second operand"
				}
			},
			"required": ["operation", "a", "b"]
		}`
		
		// Create a calculator tool instance
		calcTool := &CalculatorTool{
			name:        "calculator",
			description: "Perform basic arithmetic operations",
			schema:      calcToolSchema,
		}
		
		// Bind the tool to the client
		toolClient := anthropicClient.BindTools([]tools.Tool{calcTool})
		
		toolMessages := []schema.Message{
			schema.NewSystemMessage("You are a helpful assistant that can perform calculations."),
			schema.NewHumanMessage("What is 42 multiplied by 56?"),
		}
		
		toolMsg, err := toolClient.Generate(ctx, toolMessages, 
			llms.WithMaxTokens(250),
			llms.WithTemperature(0),
		)
		
		if err != nil {
			log.Printf("Anthropic Tool Generate failed: %v", err)
		} else {
			aiToolMsg, ok := toolMsg.(*schema.AIMessage)
			fmt.Printf("Anthropic Tool Response: %s\n", toolMsg.GetContent())
			
			if ok && len(aiToolMsg.ToolCalls) > 0 {
				fmt.Println("Tool was called with:")
				for _, tc := range aiToolMsg.ToolCalls {
					fmt.Printf("  Name: %s\n  Arguments: %s\n", tc.Name, tc.Arguments)
					
					// Execute the tool
					var args map[string]interface{}
					json.Unmarshal([]byte(tc.Arguments), &args)
					result, err := calcTool.Execute(ctx, args)
					if err != nil {
						fmt.Printf("Tool execution error: %v\n", err)
					} else {
						fmt.Printf("  Result: %s\n", result)
						
						// Send the result back to the model
						resultStr, _ := result.(string) // Add type assertion
						toolResultMsg := schema.NewToolMessage(resultStr, tc.ID)
						toolMessages = append(toolMessages, aiToolMsg, toolResultMsg)
						
						finalMsg, err := toolClient.Generate(ctx, toolMessages, 
							llms.WithMaxTokens(250),
							llms.WithTemperature(0),
						)
						if err != nil {
							log.Printf("Final tool response failed: %v", err)
						} else {
							fmt.Printf("Final assistant response: %s\n", finalMsg.GetContent())
							printUsage("Anthropic Tool Final", finalMsg)
						}
					}
				}
			} else {
				fmt.Println("No tool calls were made in the response.")
			}
			printUsage("Anthropic Tool", toolMsg)
		}
	}

	// --- Ollama Example --- 
	fmt.Println("\n--- Ollama Example ---")
	ollamaModel := config.Cfg.LLMs.Ollama.Model
	if ollamaModel == "" {
		ollamaModel = "llama3" 
		log.Printf("Ollama Model not in config, using default: %s", ollamaModel)
	}
	fmt.Printf("Using Ollama Model: %s\n", ollamaModel)
	ollamaClient, err := ollama.NewOllamaChat(ollamaModel)
	if err != nil {
		log.Printf("Failed to create Ollama client: %v (Is Ollama running and model 	%s	 pulled?)", err, ollamaModel)
	} else {
		messages := []schema.Message{schema.NewHumanMessage("What is the capital of France?")}
		aiMsg, err := ollamaClient.Generate(ctx, messages)
		if err != nil {
			log.Printf("Ollama Generate failed: %v", err)
		} else {
			fmt.Printf("Ollama Response: %s\n", aiMsg.GetContent())
			printUsage("Ollama", aiMsg)
		}
	}

	// --- Google Gemini Example --- 
	fmt.Println("\n--- Google Gemini Example ---")
	fmt.Printf("Using Gemini Model: %s\n", config.Cfg.LLMs.Gemini.Model)
	geminiClient, err := gemini.NewGeminiChat(ctx)
	if err != nil {
		log.Printf("Failed to create Gemini client: %v (Ensure BELUGA_LLMS_GEMINI_APIKEY is set)", err)
	} else {
		messages := []schema.Message{schema.NewHumanMessage("What are the main features of Google Gemini?")}
		aiMsg, err := geminiClient.Generate(ctx, messages, llms.WithMaxTokens(150))
		if err != nil {
			log.Printf("Gemini Generate failed: %v", err)
		} else {
			fmt.Printf("Gemini Response: %s\n", aiMsg.GetContent())
			printUsage("Gemini", aiMsg)
		}
	}

	// --- Cohere Example --- 
	fmt.Println("\n--- Cohere Example ---")
	cohereModel := config.Cfg.LLMs.Cohere.Model
	if cohereModel == "" {
		cohereModel = "command-r"
		log.Printf("Cohere Model not in config, using default: %s", cohereModel)
	}
	fmt.Printf("Using Cohere Model: %s\n", cohereModel)
	// Try to create Cohere client or skip if constructor not available
	// Skipping Cohere initialization for now until we have a clear implementation
	var cohereClient llms.ChatModel
	err = fmt.Errorf("Cohere client initialization skipped")
	if err != nil {
		log.Printf("Failed to create Cohere client: %v (Ensure BELUGA_LLMS_COHERE_APIKEY is set)", err)
	} else {
		messages := []schema.Message{
			schema.NewSystemMessage("You are a helpful assistant that provides concise answers."),
			schema.NewHumanMessage("What is the main purpose of the Cohere platform?"),
		}
		aiMsg, err := cohereClient.Generate(ctx, messages, llms.WithMaxTokens(100))
		if err != nil {
			log.Printf("Cohere Generate failed: %v", err)
		} else {
			fmt.Printf("Cohere Response: %s\n", aiMsg.GetContent())
			printUsage("Cohere", aiMsg)
		}
	}

	// --- AWS Bedrock Examples (Multi-Provider) --- 
	fmt.Println("\n--- AWS Bedrock Examples (Multi-Provider) ---")
	bedrockRegion := config.Cfg.LLMs.Bedrock.Region
	if bedrockRegion == "" {
		bedrockRegion = "us-east-1" // Default if not set
		log.Printf("Bedrock Region not in config, using default: %s", bedrockRegion)
	}
	fmt.Printf("Using Bedrock Region: %s\n", bedrockRegion)

	bedrockTestModels := map[string]string{
		"Anthropic Claude 3 Haiku": "anthropic.claude-3-haiku-20240307-v1:0",
		"Meta Llama 3 8B Instruct":  "meta.llama3-8b-instruct-v1:0",
		"Cohere Command R":         "cohere.command-r-v1:0",
		"AI21 Jurassic-2 Ultra":    "ai21.j2-ultra-v1",
		"Amazon Titan Text G1 Express": "amazon.titan-text-express-v1",
		"Mistral 7B Instruct":      "mistral.mistral-7b-instruct-v0:2",
	}

	for name, modelID := range bedrockTestModels {
		fmt.Printf("\n--- Bedrock Provider: %s (Model: %s) ---\n", name, modelID)
		// Define a bedrock option for max tokens
		bedrockClient, err := bedrock.NewBedrockLLM(ctx, modelID)
		if err != nil {
			log.Printf("Failed to create Bedrock client for %s: %v", name, err)
			continue
		}

		// Cohere, Anthropic, Mistral (with appropriate prompting), and Meta Llama (with appropriate prompting) can handle chat messages.
		// Other models might prefer a single concatenated prompt string, which BedrockLLM's Generate method attempts to handle
		// by extracting system and human prompts or by using the provider-specific prompt formatting (e.g., for Llama and Mistral).
		// For tool use with Mistral/Llama via InvokeModel: this typically requires careful prompt engineering to include tool descriptions
		// and to parse tool invocation requests from the model's text response. The Beluga framework's `BindTools` feature
		// stores tools, but their direct invocation by these Bedrock models (Mistral/Llama via InvokeModel) isn't as structured as with Anthropic or the Bedrock Converse API.
		// For RAG: Retrieved documents should be formatted into the prompt context for these models.
		var bedrockMessages []schema.Message
		if strings.HasPrefix(modelID, "cohere.") || strings.HasPrefix(modelID, "anthropic.") || strings.HasPrefix(modelID, "mistral.") || strings.HasPrefix(modelID, "meta.") {
			bedrockMessages = []schema.Message{
				schema.NewSystemMessage("You are a helpful AI assistant. If asked about tools, explain that tool descriptions would be in the prompt and you'd respond with a request to use them in text."),
				schema.NewHumanMessage(fmt.Sprintf("What is one key feature of the %s model family? Also, how would you use a hypothetical 'get_weather' tool if I provided its description?", name)),
			}
		} else { // For models like Titan, AI21 that strictly expect a single prompt string
			bedrockMessages = []schema.Message{
				schema.NewHumanMessage(fmt.Sprintf("Describe one key feature of the %s model family. For RAG, context would be added here.", name)),
			}
		}

		aiMsg, err := bedrockClient.Generate(ctx, bedrockMessages)
		if err != nil {
			log.Printf("Bedrock Generate failed for %s: %v", name, err)
		} else {
			fmt.Printf("Bedrock Response (%s): %s\n", name, aiMsg.GetContent())
			printUsage(fmt.Sprintf("Bedrock-%s", name), aiMsg)
		}

		// Streaming example for one Bedrock provider (e.g., Anthropic)
		if strings.HasPrefix(modelID, "anthropic.") {
			fmt.Printf("\nBedrock Streaming Response (%s):\n", name)
			streamMessages := []schema.Message{schema.NewHumanMessage("Write a very short poem about AWS Bedrock.")}
			streamChan, streamErr := bedrockClient.StreamChat(ctx, streamMessages, llms.WithMaxTokens(70))
			if streamErr != nil {
				log.Printf("Bedrock StreamChat failed for %s: %v", name, streamErr)
			} else {
				var fullStreamedResponse strings.Builder
				var lastChunk llms.AIMessageChunk
				for chunk := range streamChan {
					if chunk.Err != nil {
						log.Printf("Bedrock stream error for %s: %v", name, chunk.Err)
						break
					}
					fmt.Print(chunk.Content)
					fullStreamedResponse.WriteString(chunk.Content)
					lastChunk = chunk
				}
				fmt.Println()
				if lastChunk.AdditionalArgs != nil {
					if usage, ok := lastChunk.AdditionalArgs["usage"].(map[string]int); ok {
						fmt.Printf("Bedrock Stream Token Usage (%s, from last chunk): Input=%d, Output=%d, Total=%d\n",
							name, usage["input_tokens"], usage["output_tokens"], usage["total_tokens"],
						)
					} else {
						fmt.Printf("Bedrock Stream Token Usage (%s): N/A in last chunk\n", name)
					}
					if stopReason, ok := lastChunk.AdditionalArgs["finish_reason"].(string); ok {
						fmt.Printf("Bedrock Stream Stop Reason (%s): %s\n", name, stopReason)
					}
				} else {
					fmt.Printf("Bedrock Stream Token Usage (%s): N/A (last chunk had no AdditionalArgs)\n", name)
				}
			}
		}
	}
}

