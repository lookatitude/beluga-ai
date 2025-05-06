// Package ollama provides an implementation of the llms.ChatModel interface
// using a local Ollama instance.
package ollama

import (
	"context"
	"errors"
	"fmt"
	"log" // Added for logging warnings
	"net/url"
	"sync" // Added for Batch method
	"time" // Added for timeout

	belugaConfig "github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llms"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tools"

	"github.com/ollama/ollama/api"
)

// OllamaOption is a function type for setting options on the OllamaChat client.
type OllamaOption func(*OllamaChat)

// WithDefaultOptions sets default API options for the Ollama client.
func WithDefaultOptions(opts api.Options) OllamaOption {
	return func(oc *OllamaChat) {
		oc.defaultOptions = opts
	}
}

// WithHost sets a custom host URL for the Ollama client.
func WithHost(host string) OllamaOption {
	return func(oc *OllamaChat) {
		oc.host = host
	}
}

// WithOllamaMaxConcurrentBatches sets the concurrency limit for Batch.
func WithOllamaMaxConcurrentBatches(n int) OllamaOption {
	return func(oc *OllamaChat) {
		if n > 0 {
			oc.maxConcurrentBatches = n
		}
	}
}

// OllamaChat represents a chat model client for a local Ollama instance.
type OllamaChat struct {
	client               *api.Client
	modelName            string
	host                 string // Optional custom host
	defaultOptions       api.Options
	maxConcurrentBatches int
}

// NewOllamaChat creates a new Ollama chat client.
// It requires a model name and accepts functional options for customization.
func NewOllamaChat(modelName string, options ...OllamaOption) (*OllamaChat, error) {
	if modelName == "" {
		return nil, errors.New("Ollama model name cannot be empty")
	}

	// Corrected to use BaseURL from config
	host := belugaConfig.Cfg.LLMs.Ollama.BaseURL 
	if host == "" {
		host = "http://127.0.0.1:11434" // Default Ollama host
		log.Printf("Ollama host not found in configuration, defaulting to %s", host)
	}

	oc := &OllamaChat{
		modelName:            modelName,
		host:                 host,
		defaultOptions:       api.Options{},
		maxConcurrentBatches: 1, // Default to sequential batch processing for Ollama
	}

	for _, opt := range options {
		opt(oc)
	}

	var client *api.Client
	var err error

	client, err = api.ClientFromEnvironment()
	if err != nil {
		log.Printf("Could not create Ollama client from environment (OLLAMA_HOST not set or invalid?): %v. Falling back to configured/default host: %s", err, oc.host)
		parsedURL, parseErr := url.Parse(oc.host)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid Ollama host URL %q: %w", oc.host, parseErr)
		}
		client = api.NewClient(parsedURL, nil) 
	}
	oc.client = client

	// Define DefaultShortTimeout if not available in core, or use a local constant
	const defaultShortTimeout = 5 * time.Second 

	ctxShow, cancelShow := context.WithTimeout(context.Background(), defaultShortTimeout)
	defer cancelShow()

	// Get the actual host the client is configured to use for the error message
	clientHost := oc.host // Fallback to configured host
	if envHost := oc.client.Host(); envHost != nil && envHost.String() != "" {
	    clientHost = envHost.String()
	}

	_, err = oc.client.Show(ctxShow, &api.ShowRequest{Name: oc.modelName})
	if err != nil {
		return nil, fmt.Errorf("failed to find Ollama model %q (is Ollama running at %s and the model pulled?): %w", oc.modelName, clientHost, err)
	}

	return oc, nil
}

// mapMessages converts Beluga-ai schema messages to Ollama chat messages.
func mapOllamaMessages(messages []schema.Message) []api.Message {
	ollamaMessages := make([]api.Message, 0, len(messages))
	for _, msg := range messages {
		var role string
		switch msg.GetType() {
		case schema.MessageTypeSystem:
			role = "system"
		case schema.MessageTypeHuman:
			role = "user"
		case schema.MessageTypeAI:
			role = "assistant"
		case schema.MessageTypeTool:
			log.Printf("Warning: Skipping Tool message for Ollama API call as it_s not natively supported for input.")
			continue
		default:
			log.Printf("Warning: Skipping message of unknown type %s for Ollama API call.", msg.GetType())
			continue
		}
		
		var imagesData []api.ImageData
		if msgWithParts, ok := msg.(schema.MessageWithParts); ok { // Assuming schema.MessageWithParts exists
			for _, part := range msgWithParts.GetParts() {
				if part.MIMEType == "image/jpeg" || part.MIMEType == "image/png" {
					imagesData = append(imagesData, api.ImageData(part.Data)) // Correctly cast to api.ImageData
				}
			}
		}

		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    role,
			Content: msg.GetContent(),
			Images:  imagesData,
		})
	}
	return ollamaMessages
}

// applyOllamaOptions converts core.Option into Ollama API options,
// layering them over the default options.
func applyOllamaOptions(defaults api.Options, options ...core.Option) api.Options {
	opts := defaults
	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}

	// Corrected assignments to use pointers where the API expects them
	if temp, ok := config["temperature"].(float64); ok { f32 := float32(temp); opts.Temperature = &f32 }
	if maxTokens, ok := config["max_tokens"].(int); ok { opts.NumPredict = &maxTokens }
	if stops, ok := config["stop_sequences"].([]string); ok { opts.Stop = stops }
	if topP, ok := config["top_p"].(float64); ok { f32 := float32(topP); opts.TopP = &f32 }
	if topK, ok := config["top_k"].(int); ok { opts.TopK = &topK }
	if presPenalty, ok := config["presence_penalty"].(float64); ok { f32 := float32(presPenalty); opts.PresencePenalty = &f32 }
	if freqPenalty, ok := config["frequency_penalty"].(float64); ok { f32 := float32(freqPenalty); opts.FrequencyPenalty = &f32 }
	if seed, ok := config["seed"].(int); ok { opts.Seed = &seed }
	if numCtx, ok := config["num_ctx"].(int); ok { opts.NumCtx = &numCtx }
	if repeatLastN, ok := config["repeat_last_n"].(int); ok { opts.RepeatLastN = &repeatLastN }
	if repeatPenalty, ok := config["repeat_penalty"].(float64); ok { f32 := float32(repeatPenalty); opts.RepeatPenalty = &f32 }
	if tfsz, ok := config["tfs_z"].(float64); ok { f32 := float32(tfsz); opts.TFSZ = &f32 }
	if mirostat, ok := config["mirostat"].(int); ok { opts.Mirostat = &mirostat }
	if mirostatEta, ok := config["mirostat_eta"].(float64); ok { f32 := float32(mirostatEta); opts.MirostatEta = &f32 }
	if mirostatTau, ok := config["mirostat_tau"].(float64); ok { f32 := float32(mirostatTau); opts.MirostatTau = &f32 }
	if numGPU, ok := config["num_gpu"].(int); ok { opts.NumGPU = &numGPU }
	if mainGPU, ok := config["main_gpu"].(int); ok { opts.MainGPU = &mainGPU }
	if numThread, ok := config["num_thread"].(int); ok { opts.NumThread = &numThread }

	return opts
}

// Generate implements the llms.ChatModel interface.
func (o *OllamaChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	ollamaMessages := mapOllamaMessages(messages)
	if len(ollamaMessages) == 0 {
		return nil, errors.New("no valid messages provided for Ollama generation")
	}

	apiOptions := applyOllamaOptions(o.defaultOptions, options...)

	req := &api.ChatRequest{
		Model:    o.modelName,
		Messages: ollamaMessages,
		Options:  apiOptions,
		Stream:   core.BoolPtr(false),
	}

	var finalResponse api.ChatResponse
	respFunc := func(resp api.ChatResponse) error {
		finalResponse = resp
		return nil
	}

	err := o.client.Chat(ctx, req, respFunc)
	if err != nil {
		return nil, fmt.Errorf("ollama chat completion failed: %w", err)
	}

	if finalResponse.Message.Role != "assistant" {
		return nil, fmt.Errorf("ollama returned unexpected role: %s", finalResponse.Message.Role)
	}

	aiMsg := schema.NewAIMessage(finalResponse.Message.Content)
	aiMsg.AdditionalArgs = make(map[string]any)

	if finalResponse.PromptEvalCount > 0 || finalResponse.EvalCount > 0 {
		aiMsg.AdditionalArgs["usage"] = map[string]int{
			"prompt_tokens":     int(finalResponse.PromptEvalCount),
			"completion_tokens": int(finalResponse.EvalCount),
			"total_tokens":      int(finalResponse.PromptEvalCount + finalResponse.EvalCount),
		}
	}
	if finalResponse.DoneReason != "" {
		aiMsg.AdditionalArgs["finish_reason"] = finalResponse.DoneReason
	}

	return aiMsg, nil
}

// StreamChat implements the llms.ChatModel interface.
func (o *OllamaChat) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llms.AIMessageChunk, error) {
	ollamaMessages := mapOllamaMessages(messages)
	if len(ollamaMessages) == 0 {
		return nil, errors.New("no valid messages provided for Ollama streaming")
	}

	apiOptions := applyOllamaOptions(o.defaultOptions, options...)

	req := &api.ChatRequest{
		Model:    o.modelName,
		Messages: ollamaMessages,
		Options:  apiOptions,
		Stream:   core.BoolPtr(true),
	}

	chunkChan := make(chan llms.AIMessageChunk, 1)

	go func() {
		defer close(chunkChan)

		respFunc := func(resp api.ChatResponse) error {
			chunk := llms.AIMessageChunk{Content: resp.Message.Content, AdditionalArgs: make(map[string]any)}
			if resp.Done {
				if resp.PromptEvalCount > 0 || resp.EvalCount > 0 {
					chunk.AdditionalArgs["usage"] = map[string]int{
						"prompt_tokens":     int(resp.PromptEvalCount),
						"completion_tokens": int(resp.EvalCount),
						"total_tokens":      int(resp.PromptEvalCount + resp.EvalCount),
					}
				}
				if resp.DoneReason != "" {
					chunk.AdditionalArgs["finish_reason"] = resp.DoneReason
				}
				select {
				case chunkChan <- chunk:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil 
			}

			select {
			case chunkChan <- chunk:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		}

		err := o.client.Chat(ctx, req, respFunc)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			errChunk := llms.AIMessageChunk{Err: fmt.Errorf("ollama stream error: %w", err)}
			select {
			case chunkChan <- errChunk:
			case <-ctx.Done():
			}
		}
	}()

	return chunkChan, nil
}

// BindTools implements the llms.ChatModel interface.
func (o *OllamaChat) BindTools(toolsToBind []tools.Tool) llms.ChatModel {
	log.Println("Warning: BindTools called on OllamaChat. Ollama does not natively support API-level tool binding. Tool usage requires agent-level prompting strategies (e.g., ReAct) or specific model fine-tuning.")
	newClient := *o
	return &newClient
}

// --- core.Runnable Implementation ---

// Invoke implements the core.Runnable interface.
func (o *OllamaChat) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, fmt.Errorf("invalid input type for OllamaChat invoke: %w", err)
	}
	return o.Generate(ctx, messages, options...)
}

// Batch implements the core.Runnable interface.
func (o *OllamaChat) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	var firstErr error
	var wg sync.WaitGroup
	sem := make(chan struct{}, o.maxConcurrentBatches)
	var mu sync.Mutex

	for i, input := range inputs {
		wg.Add(1)
		sem <- struct{}{}
		go func(index int, currentInput any) {
			defer wg.Done()
			defer func() { <-sem }()

			output, err := o.Invoke(ctx, currentInput, options...)
			mu.Lock()
			if err != nil {
				results[index] = err 
				if firstErr == nil {
					firstErr = fmt.Errorf("error in Ollama batch item %d: %w", index, err)
				}
			} else {
				results[index] = output
			}
			mu.Unlock()
		}(i, input)
	}

	wg.Wait()
	return results, firstErr
}

// Stream implements the core.Runnable interface.
func (o *OllamaChat) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	messages, err := llms.EnsureMessages(input)
	if err != nil {
		return nil, fmt.Errorf("invalid input type for OllamaChat stream: %w", err)
	}

	chunkChan, err := o.StreamChat(ctx, messages, options...)
	if err != nil {
		return nil, err
	}

	resultChan := make(chan any, 1) 
	go func() {
		defer close(resultChan)
		for chunk := range chunkChan {
			if chunk.Err != nil {
				select {
				case resultChan <- chunk.Err: 
				case <-ctx.Done():
				}
				return 
			}
			select {
			case resultChan <- chunk:
			case <-ctx.Done():
				return
			}
		}
	}()

	return resultChan, nil
}

// Compile-time check to ensure OllamaChat implements interfaces.
var _ llms.ChatModel = (*OllamaChat)(nil)
var _ core.Runnable = (*OllamaChat)(nil)

