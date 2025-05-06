// docs/examples/llms/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/llms/anthropic"
	"github.com/lookatitude/beluga-ai/llms/bedrock"
	"github.com/lookatitude/beluga-ai/llms/cohere"
	"github.com/lookatitude/beluga-ai/llms/gemini"
	"github.com/lookatitude/beluga-ai/llms/ollama"
	"github.com/lookatitude/beluga-ai/llms/openai"
	"github.com/lookatitude/beluga-ai/schema"
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

func main() {
	err := config.LoadConfig() // Load configuration first
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()

	// --- OpenAI Example --- 
	fmt.Println("--- OpenAI Example ---")
	openAIClient, err := openai.NewOpenAIChat()
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
	anthropicClient, err := anthropic.NewAnthropicChat()
	if err != nil {
		log.Printf("Failed to create Anthropic client: %v", err)
	} else {
		messages := []schema.Message{schema.NewHumanMessage("Explain the concept of recursion briefly.")}
		aiMsg, err := anthropicClient.Generate(ctx, messages, llms.WithMaxTokens(100))
		if err != nil {
			log.Printf("Anthropic Generate failed: %v", err)
		} else {
			fmt.Printf("Anthropic Response: %s\n", aiMsg.GetContent())
			printUsage("Anthropic", aiMsg)
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
	cohereClient, err := cohere.NewCohereChat()
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
		bedrockClient, err := bedrock.NewBedrockLLM(ctx, modelID, bedrock.WithBedrockDefaultCallOptions(schema.CallOptions{MaxTokens: 150}))
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
		if strings.HasPrefix(modelID, "cohere.") || strings.HasPrefix(modelID, "anthropic.") || strings.HasPrefix(modelID, "mistral.") || strings.HasPrefix(modelID, "meta.") {
			messages = []schema.Message{
				schema.NewSystemMessage("You are a helpful AI assistant. If asked about tools, explain that tool descriptions would be in the prompt and you'd respond with a request to use them in text."),
				schema.NewHumanMessage(fmt.Sprintf("What is one key feature of the %s model family? Also, how would you use a hypothetical 'get_weather' tool if I provided its description?", name)),
			}
		} else { // For models like Titan, AI21 that strictly expect a single prompt string
			messages = []schema.Message{
				schema.NewHumanMessage(fmt.Sprintf("Describe one key feature of the %s model family. For RAG, context would be added here.", name)),
			}
		}

		aiMsg, err := bedrockClient.Generate(ctx, messages)
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

