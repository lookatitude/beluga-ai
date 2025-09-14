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

	belugaConfig "github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
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
	defaultCallOptions   map[string]any // Store default call options (MaxTokens, Temp, etc.) as a map
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

// WithBedrockDefaultCallOptions sets default call options for the Bedrock client using core.Option.
func WithBedrockDefaultCallOptions(opts ...core.Option) BedrockOption {
	return func(bl *BedrockLLM) {
		if bl.defaultCallOptions == nil {
			bl.defaultCallOptions = make(map[string]any) // Corrected map initialization
		}
		for _, opt := range opts {
			opt.Apply(&bl.defaultCallOptions)
		}
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
		defaultCallOptions:   make(map[string]any), // Corrected map initialization
	}

	// Apply functional options
	for _, opt := range options {
		opt(bl)
	}

	bl.setDefaultProviderCallOptions() // Apply provider-specific defaults to the map

	return bl, nil
}

func (bl *BedrockLLM) setDefaultProviderCallOptions() {
    // Apply general defaults first if not set
    if _, ok := bl.defaultCallOptions["max_tokens"]; !ok {
        bl.defaultCallOptions["max_tokens"] = 1024 // A general default
    }

	switch bl.provider {
	case ProviderAnthropic:
		if _, ok := bl.defaultCallOptions["temperature"]; !ok { bl.defaultCallOptions["temperature"] = 0.7 }
	case ProviderMeta:
		if _, ok := bl.defaultCallOptions["temperature"]; !ok { bl.defaultCallOptions["temperature"] = 0.5 }
		if _, ok := bl.defaultCallOptions["top_p"]; !ok { bl.defaultCallOptions["top_p"] = 0.9 }
	case ProviderCohere:
		if _, ok := bl.defaultCallOptions["temperature"]; !ok { bl.defaultCallOptions["temperature"] = 0.75 }
	case ProviderAI21:
		if _, ok := bl.defaultCallOptions["temperature"]; !ok { bl.defaultCallOptions["temperature"] = 0.7 }
	case ProviderAmazon: // Titan
		if _, ok := bl.defaultCallOptions["temperature"]; !ok { bl.defaultCallOptions["temperature"] = 0.0 }
	case ProviderMistral:
		if _, ok := bl.defaultCallOptions["temperature"]; !ok { bl.defaultCallOptions["temperature"] = 0.7 }
		if _, ok := bl.defaultCallOptions["top_p"]; !ok { bl.defaultCallOptions["top_p"] = 1.0 }
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

// mergeCallOptions combines default, bound, and call-specific options into a map[string]any.
func (bl *BedrockLLM) mergeCallOptions(callOptions ...core.Option) map[string]any {
	finalOpts := make(map[string]any)
	// Start with client defaults
	for k, v := range bl.defaultCallOptions {
		finalOpts[k] = v
	}

	// Apply call-specific options, potentially overriding defaults
	for _, opt := range callOptions {
		opt.Apply(&finalOpts)
	}

	// If tools were bound to the client, and not overridden by call options, use them.
	if len(bl.boundTools) > 0 {
		if _, toolsOverridden := finalOpts["tools"]; !toolsOverridden {
			finalOpts["tools"] = bl.boundTools
		}
	}

	return finalOpts
}

// Generate implements the llms.ChatModel interface.
func (bl *BedrockLLM) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	finalOptsMap := bl.mergeCallOptions(options...)
	// Provider-specific invoke functions will handle message/prompt formatting.

	var responseBody json.RawMessage
	var err error

	switch bl.provider {
	case ProviderAnthropic:
		responseBody, err = bl.invokeAnthropicModel(ctx, messages, finalOptsMap, false)
	case ProviderMeta:
		responseBody, err = bl.invokeMetaLlamaModel(ctx, bl.modelID, messages, finalOptsMap)
	case ProviderCohere:
		responseBody, err = bl.invokeCohereModel(ctx, bl.modelID, messages, finalOptsMap)
	case ProviderAI21:
		responseBody, err = bl.invokeAI21Jurassic2Model(ctx, bl.modelID, messages, finalOptsMap)
	case ProviderAmazon: // Titan Text
		responseBody, err = bl.invokeTitanTextModel(ctx, bl.modelID, messages, options...)
	case ProviderMistral:
		responseBody, err = bl.invokeMistralModel(ctx, bl.modelID, messages, finalOptsMap)
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
	finalOptsMap := bl.mergeCallOptions(options...)
	// Provider-specific invoke functions will handle message/prompt formatting.

	var streamOutputHandler *bedrockruntime.InvokeModelWithResponseStreamEventStream
	var err error

	switch bl.provider {
	case ProviderAnthropic:
		streamOutputHandler, err = bl.invokeAnthropicModelStream(ctx, messages, finalOptsMap)
	case ProviderMeta:
		streamOutputHandler, err = bl.invokeMetaLlamaModelStream(ctx, bl.modelID, messages, finalOptsMap)
	case ProviderCohere:
		streamOutputHandler, err = bl.invokeCohereModelStream(ctx, bl.modelID, messages, finalOptsMap)
	case ProviderAI21:
		streamOutputHandler, err = bl.invokeAI21Jurassic2ModelStream(ctx, bl.modelID, messages, finalOptsMap)
	case ProviderAmazon:
		streamOutputHandler, err = bl.invokeTitanTextModelStream(ctx, bl.modelID, messages, options...)
	case ProviderMistral:
		streamOutputHandler, err = bl.invokeMistralModelStream(ctx, bl.modelID, messages, finalOptsMap)
	default:
		return nil, fmt.Errorf("provider %s not supported for StreamChat", bl.provider)
	}

	if err != nil {
		return nil, err
	}

	chunkChan := make(chan llms.AIMessageChunk, 1)
	eventChan := streamOutputHandler.Events()

	go func() {
		defer close(chunkChan)

		for {
			select {
			case <-ctx.Done():
				chunkChan <- llms.AIMessageChunk{Err: ctx.Err()}
				return
			case event, ok := <-eventChan:
				if !ok { // Channel closed
					if streamErr := streamOutputHandler.Err(); streamErr != nil && !errors.Is(streamErr, io.EOF) {
						chunkChan <- llms.AIMessageChunk{Err: fmt.Errorf("bedrock stream error: %w", streamErr)}
					}
					return
				}

				switch v := event.(type) {
				case *brtypes.ResponseStreamMemberChunk:
					var chunk *llms.AIMessageChunk
					var parseErr error
					switch bl.provider {
					case ProviderAnthropic:
						chunk, parseErr = bl.anthropicStreamChunkToAIMessageChunk(v.Value.Bytes)
					case ProviderMeta:
						chunk, parseErr = bl.metaLlamaStreamChunkToAIMessageChunk(v.Value.Bytes)
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
						continue
					}
					if chunk != nil {
						select {
						case chunkChan <- *chunk:
						case <-ctx.Done():
							return
						}
					}
			  // Removed cases for specific ResponseStreamMember...Exception types as they are not part of the union.
			  // The streamOutputHandler.Err() check when the channel closes is the primary way to catch terminal stream errors.
				case *brtypes.UnknownUnionMember:
					log.Printf("Warning: Unknown Bedrock stream event union member: %s. Value: %v", v.Tag, v.Value)
				default:
					// This case handles any other event types that might appear in the stream but are not explicitly handled above.
					// This could include future additions to the SDK or unexpected event types.
					// We log it for now. If specific error types *are* part of the ResponseStream union and need explicit handling,
					// they should be added as specific cases above.
					log.Printf("Warning: Unexpected Bedrock stream event type: %T. Value: %v", v, v)
				}
			}
		}
	}()

	return chunkChan, nil
}

// BindTools implements the llms.ChatModel interface.
// Note: This method on BedrockLLM binds tools at the generic BedrockLLM level.
// The actual conversion and usage of these tools will happen in the provider-specific invoke methods.
func (bl *BedrockLLM) BindTools(toolsToBind []tools.Tool) llms.ChatModel {
	newClient := *bl // Create a shallow copy
	newClient.boundTools = make([]tools.Tool, len(toolsToBind))
	copy(newClient.boundTools, toolsToBind)

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

