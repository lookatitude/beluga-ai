// Package bedrock provides an implementation of the llms.ChatModel interface
// using AWS Bedrock Runtime, with support for multiple model providers.
package bedrock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	brtypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	belugaConfig "github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"
)

// Constants for known Bedrock model providers
const (
	ProviderAnthropic = "anthropic"
	ProviderMeta      = "meta"
	ProviderCohere    = "cohere"
	ProviderAI21      = "ai21"
	ProviderAmazon    = "amazon" // For Titan models
	ProviderMistral   = "mistral"
)

// BedrockLLM represents a chat model client for AWS Bedrock Runtime.
// It dispatches to provider-specific logic based on the modelID.
type BedrockLLM struct {
	client               *bedrockruntime.Client
	modelID              string
	provider             string // Determined from modelID
	region               string
	maxConcurrentBatches int
	defaultCallOptions   schema.CallOptions // Store default call options (MaxTokens, Temp, etc.)
	boundTools           []tools.Tool       // Store tools in a generic format, to be mapped by provider logic
}

// Compile-time check to ensure BedrockLLM implements interfaces.
var _ llms.ChatModel = (*BedrockLLM)(nil)
var _ core.Runnable = (*BedrockLLM)(nil)

// BedrockOption is a function type for setting options on the BedrockLLM client.
type BedrockOption func(*BedrockLLM)

// WithBedrockMaxConcurrentBatches sets the concurrency limit for Batch.
func WithBedrockMaxConcurrentBatches(n int) BedrockOption {
	return func(bl *BedrockLLM) {
		if n > 0 {
			bl.maxConcurrentBatches = n
		}
	}
}

// WithBedrockDefaultCallOptions sets default call options for the Bedrock client.
func WithBedrockDefaultCallOptions(opts schema.CallOptions) BedrockOption {
	return func(bl *BedrockLLM) {
		bl.defaultCallOptions = opts
	}
}

// NewBedrockLLM creates a new Bedrock chat client.
func NewBedrockLLM(ctx context.Context, modelID string, options ...BedrockOption) (*BedrockLLM, error) {
	if modelID == "" {
		return nil, errors.New("AWS Bedrock model ID cannot be empty")
	}

	provider := determineProvider(modelID)
	if provider == "" {
		return nil, fmt.Errorf("could not determine provider from model ID: %s. Supported prefixes: anthropic, meta, cohere, ai21, amazon, mistral", modelID)
	}

	var cfgOpts []func(*awsconfig.LoadOptions) error
	region := belugaConfig.Cfg.LLMs.Bedrock.Region

	if region != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithRegion(region))
	}

	// TODO: Add explicit credential provider options from belugaConfig.Cfg.LLMs.Bedrock if set

	cfg, err := awsconfig.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	if region == "" {
		region = cfg.Region
	}
	if region == "" {
		return nil, errors.New("AWS region must be configured (Beluga config, AWS config, or environment)")
	}

	client := bedrockruntime.NewFromConfig(cfg)

	bl := &BedrockLLM{
		client:               client,
		modelID:              modelID,
		provider:             provider,
		region:               region,
		maxConcurrentBatches: 5, // Default concurrency
		defaultCallOptions:   schema.CallOptions{}, // Initialize empty
	}

	// Apply functional options
	for _, opt := range options {
		opt(bl)
	}

	// Set provider-specific default call options if not already set by user
	bl.setDefaultProviderCallOptions()

	return bl, nil
}

func (bl *BedrockLLM) setDefaultProviderCallOptions() {
    // Apply general defaults first if not set
    if bl.defaultCallOptions.MaxTokens == 0 {
        bl.defaultCallOptions.MaxTokens = 1024 // A general default
    }
    if bl.defaultCallOptions.Temperature == 0 {
        // Provider specific defaults might be better, but have a fallback
    }

	switch bl.provider {
	case ProviderAnthropic:
		if bl.defaultCallOptions.Temperature == 0 { bl.defaultCallOptions.Temperature = 0.7 }
		// Anthropic uses "max_tokens_to_sample" or "max_tokens" in API, handled by provider logic
	case ProviderMeta:
		if bl.defaultCallOptions.Temperature == 0 { bl.defaultCallOptions.Temperature = 0.5 }
		if bl.defaultCallOptions.TopP == 0 { bl.defaultCallOptions.TopP = 0.9 }
	case ProviderCohere:
		if bl.defaultCallOptions.Temperature == 0 { bl.defaultCallOptions.Temperature = 0.75 }
	case ProviderAI21:
		if bl.defaultCallOptions.Temperature == 0 { bl.defaultCallOptions.Temperature = 0.7 }
	case ProviderAmazon: // Titan
		if bl.defaultCallOptions.Temperature == 0 { bl.defaultCallOptions.Temperature = 0.0 }
	case ProviderMistral:
		if bl.defaultCallOptions.Temperature == 0 { bl.defaultCallOptions.Temperature = 0.7 }
		if bl.defaultCallOptions.TopP == 0 { bl.defaultCallOptions.TopP = 1.0 }
	}
}

func determineProvider(modelID string) string {
	parts := strings.Split(modelID, ".")
	if len(parts) > 0 {
		providerPrefix := strings.ToLower(parts[0])
		switch providerPrefix {
		case ProviderAnthropic, ProviderMeta, ProviderCohere, ProviderAI21, ProviderAmazon, ProviderMistral:
			return providerPrefix
		}
	}
	return ""
}

func (bl *BedrockLLM) createInvokeModelInput(body []byte) *bedrockruntime.InvokeModelInput {
	return &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(bl.modelID),
		Body:        body,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	}
}

func (bl *BedrockLLM) createInvokeModelWithResponseStreamInput(body []byte) *bedrockruntime.InvokeModelWithResponseStreamInput {
	return &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(bl.modelID),
		Body:        body,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	}
}

// mergeCallOptions combines default, bound, and call-specific options.
func (bl *BedrockLLM) mergeCallOptions(callOptions ...core.Option) schema.CallOptions {
	finalOpts := bl.defaultCallOptions // Start with client defaults

	// Apply call-specific options, potentially overriding defaults
	emphemeralCoreOpts := make(map[string]any)
	for _, opt := range callOptions {
		opt.Apply(&ephemeralCoreOpts)
	}

	// Convert core.Option map to schema.CallOptions struct
	if mt, ok := ephemeralCoreOpts["max_tokens"].(int); ok {
		finalOpts.MaxTokens = mt
	}
	if temp, ok := ephemeralCoreOpts["temperature"].(float32); ok {
		finalOpts.Temperature = temp
	}
	if topP, ok := ephemeralCoreOpts["top_p"].(float32); ok {
		finalOpts.TopP = topP
	}
	if topK, ok := ephemeralCoreOpts["top_k"].(int); ok {
		finalOpts.TopK = topK
	}
	if stop, ok := ephemeralCoreOpts["stop_words"].([]string); ok {
		finalOpts.StopWords = stop
	}
    if tools, ok := ephemeralCoreOpts["tools"].([]tools.Tool); ok {
        finalOpts.Tools = tools
    }
    if toolChoice, ok := ephemeralCoreOpts["tool_choice"].(string); ok {
        finalOpts.ToolChoice = toolChoice
    }
    // Note: ToolResults are usually dynamic and passed directly if needed by the provider logic

	// If tools were bound to the client, and not overridden by call options, use them.
	if len(bl.boundTools) > 0 && len(finalOpts.Tools) == 0 {
		finalOpts.Tools = bl.boundTools
	}

	return finalOpts
}

// Generate implements the llms.ChatModel interface.
func (bl *BedrockLLM) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	finalOpts := bl.mergeCallOptions(options...)
	prompt, _ := llms.GetSystemAndHumanPrompts(messages) // Simplified prompt extraction for some models

	var responseBody json.RawMessage
	var err error

	switch bl.provider {
	case ProviderAnthropic:
		responseBody, err = bl.invokeAnthropicModel(ctx, messages, finalOpts, false)
	case ProviderMeta:
		responseBody, err = bl.invokeMetaLlamaModel(ctx, prompt, finalOpts) // Meta Llama uses a single prompt string
	case ProviderCohere:
		responseBody, err = bl.invokeCohereModel(ctx, prompt, messages, finalOpts)
	case ProviderAI21:
		responseBody, err = bl.invokeAI21Jurassic2Model(ctx, prompt, finalOpts) // AI21 J2 uses a single prompt string
	case ProviderAmazon: // Titan Text
		responseBody, err = bl.invokeTitanTextModel(ctx, prompt, finalOpts) // Titan Text uses a single prompt string
	case ProviderMistral:
		responseBody, err = bl.invokeMistralModel(ctx, prompt, finalOpts) // Mistral uses a single prompt string
	default:
		return nil, fmt.Errorf("provider %s not supported for Generate", bl.provider)
	}

	if err != nil {
		return nil, err // Error already wrapped by provider-specific invoke
	}

	switch bl.provider {
	case ProviderAnthropic:
		return bl.anthropicResponseToAIMessage(responseBody)
	case ProviderMeta:
		return bl.metaLlamaResponseToAIMessage(responseBody)
	case ProviderCohere:
		return bl.cohereResponseToAIMessage(responseBody)
	case ProviderAI21:
		return bl.ai21Jurassic2ResponseToAIMessage(responseBody)
	case ProviderAmazon:
		return bl.titanTextResponseToAIMessage(responseBody)
	case ProviderMistral:
		return bl.mistralResponseToAIMessage(responseBody)
	default:
		return nil, fmt.Errorf("provider %s not supported for response parsing", bl.provider)
	}
}

// StreamChat implements the llms.ChatModel interface.
func (bl *BedrockLLM) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llms.AIMessageChunk, error) {
	finalOpts := bl.mergeCallOptions(options...)
	prompt, _ := llms.GetSystemAndHumanPrompts(messages) // Simplified for some models

	var bedrockStream *brtypes.ResponseStream
	var err error

	switch bl.provider {
	case ProviderAnthropic:
		bedrockStream, err = bl.invokeAnthropicModelStream(ctx, messages, finalOpts)
	case ProviderMeta:
		bedrockStream, err = bl.invokeMetaLlamaModelStream(ctx, prompt, finalOpts)
	case ProviderCohere:
		bedrockStream, err = bl.invokeCohereModelStream(ctx, prompt, messages, finalOpts)
	case ProviderAI21:
		bedrockStream, err = bl.invokeAI21Jurassic2ModelStream(ctx, prompt, finalOpts)
	case ProviderAmazon:
		bedrockStream, err = bl.invokeTitanTextModelStream(ctx, prompt, finalOpts)
	case ProviderMistral:
		bedrockStream, err = bl.invokeMistralModelStream(ctx, prompt, finalOpts)
	default:
		return nil, fmt.Errorf("provider %s not supported for StreamChat", bl.provider)
	}

	if err != nil {
		return nil, err
	}

	chunkChan := make(chan llms.AIMessageChunk, 1)
	eventStream := bedrockStream.GetStream()

	go func() {
		defer close(chunkChan)
		defer func() {
			if err := eventStream.Close(); err != nil && !errors.Is(err, io.EOF) {
				log.Printf("Error closing Bedrock event stream: %v", err)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				chunkChan <- llms.AIMessageChunk{Err: ctx.Err()}
				return
			default:
				event, ok := <-eventStream.Events()
				if !ok {
					if streamErr := eventStream.Err(); streamErr != nil && !errors.Is(streamErr, io.EOF) {
						chunkChan <- llms.AIMessageChunk{Err: fmt.Errorf("bedrock stream error: %w", streamErr)}
					}
					return
				}

				switch v := event.(type) {
				case *brtypes.ResponseStreamMemberChunk:
					var chunk *llms.AIMessageChunk // Use pointer to handle nil for non-content chunks
					var parseErr error
					switch bl.provider {
					case ProviderAnthropic:
						chunk, parseErr = bl.anthropicStreamChunkToAIMessageChunk(v.Value.Bytes)
					case ProviderMeta:
						chunk, parseErr = bl.metaLlamaStreamChunkToAIMessageChunk(v.Value)
					case ProviderCohere:
						chunk, parseErr = bl.cohereStreamChunkToAIMessageChunk(v.Value.Bytes)
					case ProviderAI21:
						chunk, parseErr = bl.ai21Jurassic2StreamChunkToAIMessageChunk(v.Value.Bytes)
					case ProviderAmazon:
						chunk, parseErr = bl.titanTextStreamChunkToAIMessageChunk(v.Value.Bytes)
					case ProviderMistral:
						chunk, parseErr = bl.mistralStreamChunkToAIMessageChunk(v.Value.Bytes)
					default:
						parseErr = fmt.Errorf("stream chunk parsing not supported for provider %s", bl.provider)
					}

					if parseErr != nil {
						chunkChan <- llms.AIMessageChunk{Err: fmt.Errorf("chunk parse error: %w", parseErr)}
						continue // Or return, depending on desired error handling
					}
					if chunk != nil { // Only send if chunk is not nil (e.g. Cohere might return nil for non-content events)
						select {
						case chunkChan <- *chunk:
						case <-ctx.Done():
							return
						}
					}
				case *brtypes.UnknownUnionMember:
					log.Printf("Warning: Unknown Bedrock stream event type: %s", v.Tag)
				default:
					log.Printf("Warning: Unexpected Bedrock stream event type: %T", v)
				}
			}
		}
	}()

	return chunkChan, nil
}

// BindTools implements the llms.ChatModel interface.
func (bl *BedrockLLM) BindTools(toolsToBind []tools.Tool) llms.ChatModel {
	newClient := *bl // Create a shallow copy
	newClient.boundTools = make([]tools.Tool, len(toolsToBind))
	copy(newClient.boundTools, toolsToBind)

	// Provider-specific tool mapping/validation can happen during request building
	// or here if there's a common structure to pre-process.
	// For now, we store the generic tools.Tool and let provider logic handle it.
	// Example: Anthropic maps these to its specific tool format during request build.
	// Cohere also has its own tool format.

	log.Printf("Tools bound to Bedrock client for provider %s. Actual tool use support depends on the specific model and provider logic.", bl.provider)
	return &newClient
}

// --- core.Runnable Implementation ---

func (bl *BedrockLLM) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}
	return bl.Generate(ctx, messages, options...)
}

func (bl *BedrockLLM) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	errorsList := make([]error, len(inputs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, bl.maxConcurrentBatches)

	for i, input := range inputs {
		wg.Add(1)
		sem <- struct{}{}
		go func(index int, currentInput any) {
			defer wg.Done()
			defer func() { <-sem }()
			// TODO: Handle per-request options if core.Option can be made to target specific batch items
			result, err := bl.Invoke(ctx, currentInput, options...)
			results[index] = result
			errorsList[index] = err
		}(i, input)
	}
	wg.Wait()

	var combinedError error
	for _, err := range errorsList {
		if err != nil {
			if combinedError == nil {
				combinedError = fmt.Errorf("batch errors: %w", err)
			} else {
				combinedError = fmt.Errorf("%w; %w", combinedError, err)
			}
		}
	}
	return results, combinedError
}

func (bl *BedrockLLM) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, err
	}

	chunkChan, err := bl.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	outputChan := make(chan any, 1)
	go func() {
		defer close(outputChan)
		for chunk := range chunkChan {
			if chunk.Err != nil {
				select {
				case outputChan <- chunk.Err:
				case <-ctx.Done():
				}
				return
			}
			select {
			case outputChan <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()
	return outputChan, nil
}

