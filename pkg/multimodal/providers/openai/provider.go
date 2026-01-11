// Package openai provides OpenAI provider implementation for multimodal models.
package openai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	openaiClient "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OpenAIProvider implements the MultimodalModel interface for OpenAI.
type OpenAIProvider struct {
	config       *Config
	capabilities *ModalityCapabilities
	client       *openaiClient.Client
}

// NewOpenAIProvider creates a new OpenAI multimodal provider.
// Accepts the provider's Config directly to avoid import cycles.
func NewOpenAIProvider(openaiConfig *Config) (iface.MultimodalModel, error) {
	if err := openaiConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OpenAI configuration: %w", err)
	}

	// Define OpenAI capabilities (supports text, image, audio, video)
	capabilities := &ModalityCapabilities{
		Text:  true,
		Image: true,
		Audio: true,
		Video: true,

		MaxImageSize: 20 * 1024 * 1024,  // 20MB
		MaxAudioSize: 25 * 1024 * 1024,  // 25MB
		MaxVideoSize: 100 * 1024 * 1024, // 100MB

		SupportedImageFormats: []string{"png", "jpeg", "jpg", "gif", "webp"},
		SupportedAudioFormats: []string{"mp3", "wav", "m4a", "ogg"},
		SupportedVideoFormats: []string{"mp4", "webm", "mov"},
	}

	// Initialize OpenAI client
	clientConfig := openaiClient.DefaultConfig(openaiConfig.APIKey)
	if openaiConfig.BaseURL != "" {
		clientConfig.BaseURL = openaiConfig.BaseURL
	}
	if openaiConfig.APIVersion != "" {
		clientConfig.APIVersion = openaiConfig.APIVersion
	}

	client := openaiClient.NewClientWithConfig(clientConfig)

	provider := &OpenAIProvider{
		config:       openaiConfig,
		capabilities: capabilities,
		client:       client,
	}

	return provider, nil
}

// Process processes a multimodal input using OpenAI's API.
func (p *OpenAIProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/openai")
	ctx, span := tracer.Start(ctx, "openai.Process",
		trace.WithAttributes(
			attribute.String("provider", "openai"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	startTime := time.Now()

	// Convert multimodal input to OpenAI API format
	openaiMessages, err := p.convertToOpenAIMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build OpenAI chat completion request
	req := openaiClient.ChatCompletionRequest{
		Model:    p.config.Model,
		Messages: openaiMessages,
	}

	// Make API call with retry logic
	var resp openaiClient.ChatCompletionResponse
	maxRetries := p.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, lastErr = p.client.CreateChatCompletion(ctx, req)
		if lastErr == nil {
			break
		}

		// Check if error is retryable
		if !isRetryableError(lastErr) {
			break
		}

		span.AddEvent("retry_attempt",
			trace.WithAttributes(
				attribute.Int("attempt", attempt+1),
				attribute.String("error", lastErr.Error()),
			))
	}

	if lastErr != nil {
		span.RecordError(lastErr)
		span.SetStatus(codes.Error, lastErr.Error())
		return nil, fmt.Errorf("OpenAI API call failed after %d attempts: %w", maxRetries, lastErr)
	}

	// Convert OpenAI response to MultimodalOutput
	output, err := p.convertOpenAIResponse(ctx, &resp, input.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert OpenAI response: %w", err)
	}

	duration := time.Since(startTime)
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))

	span.SetStatus(codes.Ok, "")
	// Log with OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		slog.Info("OpenAI multimodal processing completed",
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
			"input_id", input.ID,
			"output_id", output.ID)
	} else {
		slog.Info("OpenAI multimodal processing completed",
			"input_id", input.ID,
			"output_id", output.ID)
	}

	return output, nil
}

// ProcessStream processes a multimodal input with streaming support.
func (p *OpenAIProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/openai")
	ctx, span := tracer.Start(ctx, "openai.ProcessStream",
		trace.WithAttributes(
			attribute.String("provider", "openai"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
		))
	defer span.End()

	// Convert multimodal input to OpenAI API format
	openaiMessages, err := p.convertToOpenAIMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build OpenAI chat completion request with streaming
	req := openaiClient.ChatCompletionRequest{
		Model:    p.config.Model,
		Messages: openaiMessages,
		Stream:   true,
	}

	// Create streaming response
	stream, err := p.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to create streaming request: %w", err)
	}

	outputChan := make(chan *types.MultimodalOutput, 10)

	go func() {
		defer close(outputChan)
		defer stream.Close()

		var accumulatedText strings.Builder
		outputID := uuid.New().String()

		for {
			response, err := stream.Recv()
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return
				}
				// Stream ended normally
				if strings.Contains(err.Error(), "stream closed") || strings.Contains(err.Error(), "EOF") {
					break
				}
				// Other errors
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return
			}

			// Extract text from response
			if len(response.Choices) > 0 {
				delta := response.Choices[0].Delta
				if delta.Content != "" {
					accumulatedText.WriteString(delta.Content)

					// Send incremental output
					output := &types.MultimodalOutput{
						ID:      outputID,
						InputID: input.ID,
						ContentBlocks: []*types.ContentBlock{
							{
								Type:     "text",
								Data:     []byte(accumulatedText.String()),
								Format:   "text/plain",
								MIMEType: "text/plain",
								Size:     int64(accumulatedText.Len()),
								Metadata: map[string]any{"incremental": true},
							},
						},
						Metadata:   map[string]any{"provider": "openai", "model": p.config.Model},
						Confidence: 0.95,
						Provider:   "openai",
						Model:      p.config.Model,
						CreatedAt:  time.Now(),
					}

					select {
					case outputChan <- output:
					case <-ctx.Done():
						return
					}
				}
			}
		}

		// Send final output
		finalText := accumulatedText.String()
		if finalText != "" {
			output := &types.MultimodalOutput{
				ID:      outputID,
				InputID: input.ID,
				ContentBlocks: []*types.ContentBlock{
					{
						Type:     "text",
						Data:     []byte(finalText),
						Format:   "text/plain",
						MIMEType: "text/plain",
						Size:     int64(len(finalText)),
						Metadata: map[string]any{"final": true},
					},
				},
				Metadata:   map[string]any{"provider": "openai", "model": p.config.Model},
				Confidence: 0.95,
				Provider:   "openai",
				Model:      p.config.Model,
				CreatedAt:  time.Now(),
			}

			select {
			case outputChan <- output:
			case <-ctx.Done():
			}
		}

		span.SetStatus(codes.Ok, "")
	}()

	return outputChan, nil
}

// GetCapabilities returns the capabilities of the OpenAI provider.
func (p *OpenAIProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/openai")
	_, span := tracer.Start(ctx, "openai.GetCapabilities",
		trace.WithAttributes(
			attribute.String("provider", "openai"),
			attribute.String("model", p.config.Model),
		))
	defer span.End()

	// Convert local capabilities to types.ModalityCapabilities
	result := &types.ModalityCapabilities{
		Text:                  p.capabilities.Text,
		Image:                 p.capabilities.Image,
		Audio:                 p.capabilities.Audio,
		Video:                 p.capabilities.Video,
		MaxImageSize:          p.capabilities.MaxImageSize,
		MaxAudioSize:          p.capabilities.MaxAudioSize,
		MaxVideoSize:          p.capabilities.MaxVideoSize,
		SupportedImageFormats: p.capabilities.SupportedImageFormats,
		SupportedAudioFormats: p.capabilities.SupportedAudioFormats,
		SupportedVideoFormats: p.capabilities.SupportedVideoFormats,
	}

	// Metrics recording would go here if needed
	// For now, we avoid importing multimodal package to prevent import cycles
	span.SetStatus(codes.Ok, "")
	return result, nil
}

// SupportsModality checks if OpenAI supports a specific modality.
func (p *OpenAIProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/openai")
	_, span := tracer.Start(ctx, "openai.SupportsModality",
		trace.WithAttributes(
			attribute.String("provider", "openai"),
			attribute.String("model", p.config.Model),
			attribute.String("modality", modality),
		))
	defer span.End()

	supported := false
	switch modality {
	case "text":
		supported = p.capabilities.Text
	case "image":
		supported = p.capabilities.Image
	case "audio":
		supported = p.capabilities.Audio
	case "video":
		supported = p.capabilities.Video
	}

	// Metrics recording would go here if needed
	// For now, we avoid importing multimodal package to prevent import cycles
	span.SetAttributes(attribute.Bool("supported", supported))
	span.SetStatus(codes.Ok, "")
	return supported, nil
}

// CheckHealth performs a health check and returns an error if the provider is unhealthy.
func (p *OpenAIProvider) CheckHealth(ctx context.Context) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/openai")
	ctx, span := tracer.Start(ctx, "openai.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider", "openai"),
			attribute.String("model", p.config.Model),
		))
	defer span.End()

	// Basic health check: verify capabilities can be retrieved
	_, err := p.GetCapabilities(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// convertToOpenAIMessages converts multimodal content blocks to OpenAI message format.
func (p *OpenAIProvider) convertToOpenAIMessages(ctx context.Context, blocks []*types.ContentBlock) ([]openaiClient.ChatCompletionMessage, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no content blocks provided")
	}

	// Group blocks into messages (text and images can be combined in one message)
	var messages []openaiClient.ChatCompletionMessage
	var currentParts []openaiClient.ChatMessagePart
	var currentRole string

	for _, block := range blocks {
		switch block.Type {
		case "text":
			// Add text part
			currentParts = append(currentParts, openaiClient.ChatMessagePart{
				Type: openaiClient.ChatMessagePartTypeText,
				Text: string(block.Data),
			})
			currentRole = openaiClient.ChatMessageRoleUser

		case "image":
			// Add image part
			var imageURL string
			if block.URL != "" {
				imageURL = block.URL
			} else if len(block.Data) > 0 {
				// Encode as base64 data URL
				mimeType := block.MIMEType
				if mimeType == "" {
					mimeType = "image/png" // Default
				}
				encoded := base64.StdEncoding.EncodeToString(block.Data)
				imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
			} else {
				return nil, fmt.Errorf("image block has no URL or data")
			}

			currentParts = append(currentParts, openaiClient.ChatMessagePart{
				Type: openaiClient.ChatMessagePartTypeImageURL,
				ImageURL: &openaiClient.ChatMessageImageURL{
					URL: imageURL,
				},
			})
			currentRole = openaiClient.ChatMessageRoleUser

		case "audio", "video":
			// OpenAI's chat API doesn't directly support audio/video in the same way
			// For now, we'll convert to text description
			// In production, you might need to use a different endpoint or transcribe first
			description := fmt.Sprintf("[%s content: %s]", block.Type, block.Format)
			currentParts = append(currentParts, openaiClient.ChatMessagePart{
				Type: openaiClient.ChatMessagePartTypeText,
				Text: description,
			})
			currentRole = openaiClient.ChatMessageRoleUser
		}
	}

	// Create final message with all parts
	if len(currentParts) > 0 {
		messages = append(messages, openaiClient.ChatCompletionMessage{
			Role:         currentRole,
			MultiContent: currentParts,
		})
	}

	if len(messages) == 0 {
		return nil, errors.New("no valid messages created from content blocks")
	}

	return messages, nil
}

// convertOpenAIResponse converts OpenAI API response to MultimodalOutput.
func (p *OpenAIProvider) convertOpenAIResponse(ctx context.Context, resp *openaiClient.ChatCompletionResponse, inputID string) (*types.MultimodalOutput, error) {
	if len(resp.Choices) == 0 {
		return nil, errors.New("OpenAI response has no choices")
	}

	choice := resp.Choices[0]

	// Extract content - can be from Content (string) or MultiContent (parts)
	var content string
	if choice.Message.Content != "" {
		content = choice.Message.Content
	} else if len(choice.Message.MultiContent) > 0 {
		// Extract text from multimodal content
		var textParts []string
		for _, part := range choice.Message.MultiContent {
			if part.Text != "" {
				textParts = append(textParts, part.Text)
			}
		}
		content = strings.Join(textParts, "")
	}

	// Create content block from response
	responseBlock := &types.ContentBlock{
		Type:     "text",
		Data:     []byte(content),
		Format:   "text/plain",
		MIMEType: "text/plain",
		Size:     int64(len(content)),
		Metadata: map[string]any{
			"finish_reason": choice.FinishReason,
		},
	}

	output := &types.MultimodalOutput{
		ID:            uuid.New().String(),
		InputID:       inputID,
		ContentBlocks: []*types.ContentBlock{responseBlock},
		Metadata: map[string]any{
			"provider":      "openai",
			"model":         p.config.Model,
			"usage":         resp.Usage,
			"finish_reason": choice.FinishReason,
		},
		Confidence: 0.95, // OpenAI doesn't provide confidence scores
		Provider:   "openai",
		Model:      p.config.Model,
		CreatedAt:  time.Now(),
	}

	return output, nil
}

// isRetryableError checks if an error is retryable.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Retry on rate limits, timeouts, and server errors
	retryablePatterns := []string{
		"rate limit",
		"timeout",
		"503",
		"502",
		"500",
		"429",
		"connection",
		"temporary",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}
