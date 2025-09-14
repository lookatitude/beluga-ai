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

	belugaConfig "github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"

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

	// Use the configured host for error messages
	clientHost := oc.host // We'll use the configured host for error messages

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
		
		// Check for message parts with images
		var imagesData []api.ImageData
		
		// Try to access message parts using type assertion
		// This is a more generic approach that doesn't require a specific interface
		additionalArgs := make(map[string]interface{})
		if argsMsg, ok := msg.(interface{ GetAdditionalArgs() map[string]any }); ok {
			additionalArgs = argsMsg.GetAdditionalArgs()
		}
		
		if parts, ok := additionalArgs["parts"].([]map[string]interface{}); ok {
			for _, part := range parts {
				mimeType, hasMimeType := part["mime_type"].(string)
				data, hasData := part["data"].([]byte)
				
				if hasMimeType && hasData && (mimeType == "image/jpeg" || mimeType == "image/png") {
					imagesData = append(imagesData, api.ImageData(data))
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

	// Create proper pointer variables for each option
	if temp, ok := config["temperature"].(float64); ok {
		tempF32 := float32(temp)
		opts.Temperature = tempF32
	}
	if maxTokens, ok := config["max_tokens"].(int); ok {
		mtCopy := maxTokens
		opts.NumPredict = mtCopy
	}
	if stops, ok := config["stop_sequences"].([]string); ok {
		opts.Stop = stops // This is already a slice, not a pointer
	}
	if topP, ok := config["top_p"].(float64); ok {
		tpF32 := float32(topP)
		opts.TopP = tpF32
	}
	if topK, ok := config["top_k"].(int); ok {
		tkCopy := topK
		opts.TopK = tkCopy
	}
	if presPenalty, ok := config["presence_penalty"].(float64); ok {
		ppF32 := float32(presPenalty)
		opts.PresencePenalty = ppF32
	}
	if freqPenalty, ok := config["frequency_penalty"].(float64); ok {
		fpF32 := float32(freqPenalty)
		opts.FrequencyPenalty = fpF32
	}
	if seed, ok := config["seed"].(int); ok {
		seedCopy := seed
		opts.Seed = seedCopy
	}
	if numCtx, ok := config["num_ctx"].(int); ok {
		ncCopy := numCtx
		opts.NumCtx = ncCopy
	}
	if repeatLastN, ok := config["repeat_last_n"].(int); ok {
		rlnCopy := repeatLastN
		opts.RepeatLastN = rlnCopy
	}
	if repeatPenalty, ok := config["repeat_penalty"].(float64); ok {
		rpF32 := float32(repeatPenalty)
		opts.RepeatPenalty = rpF32
	}
	// TFSZ is not in the current Ollama API, so we'll skip this setting
	// if tfsz, ok := config["tfs_z"].(float64); ok {
	//    tfszF32 := float32(tfsz)
	//    // opts.TFSZ = tfszF32  // This field doesn't exist
	// }
	if mirostat, ok := config["mirostat"].(int); ok {
		msCopy := mirostat
		opts.Mirostat = msCopy
	}
	if mirostatEta, ok := config["mirostat_eta"].(float64); ok {
		metaF32 := float32(mirostatEta)
		opts.MirostatEta = metaF32
	}
	if mirostatTau, ok := config["mirostat_tau"].(float64); ok {
		mtauF32 := float32(mirostatTau)
		opts.MirostatTau = mtauF32
	}
	if numGPU, ok := config["num_gpu"].(int); ok {
		ngCopy := numGPU
		opts.NumGPU = ngCopy
	}
	if mainGPU, ok := config["main_gpu"].(int); ok {
		mgCopy := mainGPU
		opts.MainGPU = mgCopy
	}
	if numThread, ok := config["num_thread"].(int); ok {
		ntCopy := numThread
		opts.NumThread = ntCopy
	}

	return opts
}

// convertOptionsToMap converts api.Options to map[string]any for use in ChatRequest
func convertOptionsToMap(opts api.Options) map[string]any {
	optionsMap := make(map[string]any)
	
	// Only include non-zero values
	if opts.Temperature != 0 {
		optionsMap["temperature"] = opts.Temperature
	}
	if opts.NumPredict != 0 {
		optionsMap["num_predict"] = opts.NumPredict
	}
	if len(opts.Stop) > 0 {
		optionsMap["stop"] = opts.Stop
	}
	if opts.TopP != 0 {
		optionsMap["top_p"] = opts.TopP
	}
	if opts.TopK != 0 {
		optionsMap["top_k"] = opts.TopK
	}
	if opts.PresencePenalty != 0 {
		optionsMap["presence_penalty"] = opts.PresencePenalty
	}
	if opts.FrequencyPenalty != 0 {
		optionsMap["frequency_penalty"] = opts.FrequencyPenalty
	}
	if opts.Seed != 0 {
		optionsMap["seed"] = opts.Seed
	}
	if opts.NumCtx != 0 {
		optionsMap["num_ctx"] = opts.NumCtx
	}
	if opts.RepeatLastN != 0 {
		optionsMap["repeat_last_n"] = opts.RepeatLastN
	}
	if opts.RepeatPenalty != 0 {
		optionsMap["repeat_penalty"] = opts.RepeatPenalty
	}
	if opts.Mirostat != 0 {
		optionsMap["mirostat"] = opts.Mirostat
	}
	if opts.MirostatEta != 0 {
		optionsMap["mirostat_eta"] = opts.MirostatEta
	}
	if opts.MirostatTau != 0 {
		optionsMap["mirostat_tau"] = opts.MirostatTau
	}
	if opts.NumGPU != 0 {
		optionsMap["num_gpu"] = opts.NumGPU
	}
	if opts.MainGPU != 0 {
		optionsMap["main_gpu"] = opts.MainGPU
	}
	if opts.NumThread != 0 {
		optionsMap["num_thread"] = opts.NumThread
	}

	return optionsMap
}

// Generate implements the llms.ChatModel interface.
func (o *OllamaChat) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	ollamaMessages := mapOllamaMessages(messages)
	if len(ollamaMessages) == 0 {
		return nil, errors.New("no valid messages provided for Ollama generation")
	}

	apiOptions := applyOllamaOptions(o.defaultOptions, options...)
	
	// Convert options to map[string]any
	optionsMap := convertOptionsToMap(apiOptions)

	// Create a chat request with the processed options
	req := &api.ChatRequest{
		Model:    o.modelName,
		Messages: ollamaMessages,
		Options:  optionsMap,
		Stream:   boolPtr(false),
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
	
	// Convert options to map[string]any
	optionsMap := convertOptionsToMap(apiOptions)

	// Create a streaming chat request with the processed options
	req := &api.ChatRequest{
		Model:    o.modelName,
		Messages: ollamaMessages,
		Options:  optionsMap,
		Stream:   boolPtr(true),
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

// Helper function to replace core.BoolPtr
func boolPtr(b bool) *bool {
	return &b
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

