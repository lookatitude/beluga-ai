// Package qwen provides Qwen (Alibaba) provider implementation for multimodal models.
package qwen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// QwenGenerateRequest represents the Qwen API request structure.
type QwenGenerateRequest struct {
	Model  string      `json:"model"`
	Input  interface{} `json:"input"` // Can be string or messages array
	Stream bool        `json:"stream,omitempty"`
}

// QwenMessage represents a message in Qwen API format.
type QwenMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or array of content parts
}

// QwenContentPart represents a content part (text or image).
type QwenContentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *QwenImageURL `json:"image_url,omitempty"`
}

// QwenImageURL represents an image URL in Qwen format.
type QwenImageURL struct {
	URL string `json:"url"`
}

// QwenGenerateResponse represents the Qwen API response structure.
type QwenGenerateResponse struct {
	Output QwenOutput `json:"output"`
	Usage  QwenUsage  `json:"usage,omitempty"`
}

// QwenOutput represents the output from Qwen API.
type QwenOutput struct {
	Choices []QwenChoice `json:"choices"`
}

// QwenChoice represents a choice in Qwen response.
type QwenChoice struct {
	Message QwenMessage `json:"message"`
}

// QwenUsage represents usage information.
type QwenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// QwenProvider implements the MultimodalModel interface for Qwen.
type QwenProvider struct {
	config       *Config
	capabilities *ModalityCapabilities
	httpClient   *http.Client
}

// NewQwenProvider creates a new Qwen multimodal provider.
func NewQwenProvider(qwenConfig *Config) (iface.MultimodalModel, error) {
	if err := qwenConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Qwen configuration: %w", err)
	}

	// Define Qwen capabilities (supports text, image, audio, video)
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

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: qwenConfig.Timeout,
	}

	provider := &QwenProvider{
		config:       qwenConfig,
		capabilities: capabilities,
		httpClient:   httpClient,
	}

	return provider, nil
}

// Process processes a multimodal input using Qwen's API.
func (p *QwenProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/qwen")
	ctx, span := tracer.Start(ctx, "qwen.Process",
		trace.WithAttributes(
			attribute.String("provider", "qwen"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	startTime := time.Now()

	// Convert multimodal input to Qwen API format
	messages, err := p.convertToQwenMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Qwen request
	req := &QwenGenerateRequest{
		Model:  p.config.Model,
		Input:  messages,
		Stream: false,
	}

	// Make API call with retry logic
	var resp *QwenGenerateResponse
	maxRetries := p.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, lastErr = p.makeAPIRequest(ctx, req)
		if lastErr == nil {
			break
		}

		// Check if error is retryable
		if !isRetryableError(lastErr) {
			break
		}

		logWithOTELContext(ctx, slog.LevelWarn, "Qwen API call failed, retrying",
			"error", lastErr,
			"attempt", attempt+1,
			"max_retries", maxRetries)
	}

	if lastErr != nil {
		span.RecordError(lastErr)
		span.SetStatus(codes.Error, lastErr.Error())
		return nil, fmt.Errorf("Qwen API call failed: %w", lastErr)
	}

	// Convert response to multimodal output
	output, err := p.convertQwenResponse(ctx, resp, input.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	duration := time.Since(startTime)
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
	span.SetStatus(codes.Ok, "")

	logWithOTELContext(ctx, slog.LevelInfo, "Qwen multimodal processing completed",
		"input_id", input.ID,
		"output_id", output.ID,
		"duration_ms", duration.Milliseconds())

	return output, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
func (p *QwenProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/qwen")
	ctx, span := tracer.Start(ctx, "qwen.ProcessStream",
		trace.WithAttributes(
			attribute.String("provider", "qwen"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	// Convert multimodal input to Qwen API format
	messages, err := p.convertToQwenMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Qwen streaming request
	req := &QwenGenerateRequest{
		Model:  p.config.Model,
		Input:  messages,
		Stream: true,
	}

	outputChan := make(chan *types.MultimodalOutput, 10)

	go func() {
		defer close(outputChan)

		var accumulatedText strings.Builder
		outputID := uuid.New().String()

		// Make streaming request
		err := p.makeStreamingRequest(ctx, req, func(chunk string) {
			if chunk != "" {
				accumulatedText.WriteString(chunk)

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
					Metadata:   map[string]any{"provider": "qwen", "model": p.config.Model},
					Confidence: 0.95,
					Provider:   "qwen",
					Model:      p.config.Model,
					CreatedAt:  time.Now(),
				}

				select {
				case outputChan <- output:
				case <-ctx.Done():
					return
				}
			}
		})

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
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
				Metadata:   map[string]any{"provider": "qwen", "model": p.config.Model},
				Confidence: 0.95,
				Provider:   "qwen",
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

// GetCapabilities returns the capabilities of the Qwen provider.
func (p *QwenProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/qwen")
	_, span := tracer.Start(ctx, "qwen.GetCapabilities",
		trace.WithAttributes(
			attribute.String("provider", "qwen"),
			attribute.String("model", p.config.Model),
		))
	defer span.End()

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

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// SupportsModality checks if Qwen supports a specific modality.
func (p *QwenProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/qwen")
	_, span := tracer.Start(ctx, "qwen.SupportsModality",
		trace.WithAttributes(
			attribute.String("provider", "qwen"),
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
	default:
		span.SetStatus(codes.Error, fmt.Sprintf("unknown modality: %s", modality))
		return false, fmt.Errorf("unknown modality: %s", modality)
	}

	span.SetAttributes(attribute.Bool("supported", supported))
	span.SetStatus(codes.Ok, "")
	return supported, nil
}

// CheckHealth performs a health check and returns an error if the provider is unhealthy.
func (p *QwenProvider) CheckHealth(ctx context.Context) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/qwen")
	ctx, span := tracer.Start(ctx, "qwen.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider", "qwen"),
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

// convertToQwenMessages converts multimodal content blocks to Qwen message format.
func (p *QwenProvider) convertToQwenMessages(ctx context.Context, blocks []*types.ContentBlock) ([]QwenMessage, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no content blocks provided")
	}

	var contentParts []QwenContentPart

	for _, block := range blocks {
		switch block.Type {
		case "text":
			contentParts = append(contentParts, QwenContentPart{
				Type: "text",
				Text: string(block.Data),
			})

		case "image":
			// Qwen supports base64 images via data URLs
			var imageURL string
			if len(block.Data) > 0 {
				mimeType := block.MIMEType
				if mimeType == "" {
					mimeType = "image/png"
				}
				encoded := base64.StdEncoding.EncodeToString(block.Data)
				imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
			} else if block.URL != "" {
				imageURL = block.URL
			} else {
				return nil, fmt.Errorf("image block has no URL or data")
			}

			contentParts = append(contentParts, QwenContentPart{
				Type: "image_url",
				ImageURL: &QwenImageURL{
					URL: imageURL,
				},
			})

		case "audio", "video":
			// Qwen may support audio/video in future, for now convert to text description
			description := fmt.Sprintf("[%s content: %s]", block.Type, block.Format)
			contentParts = append(contentParts, QwenContentPart{
				Type: "text",
				Text: description,
			})
		}
	}

	// If only one text part, use simple string format
	if len(contentParts) == 1 && contentParts[0].Type == "text" {
		return []QwenMessage{
			{
				Role:    "user",
				Content: contentParts[0].Text,
			},
		}, nil
	}

	// Otherwise use array format
	return []QwenMessage{
		{
			Role:    "user",
			Content: contentParts,
		},
	}, nil
}

// makeAPIRequest makes an HTTP request to Qwen API.
func (p *QwenProvider) makeAPIRequest(ctx context.Context, req *QwenGenerateRequest) (*QwenGenerateResponse, error) {
	url := fmt.Sprintf("%s/services/aigc/multimodal-generation/generation", p.config.BaseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp QwenGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}

// makeStreamingRequest makes a streaming HTTP request to Qwen API.
func (p *QwenProvider) makeStreamingRequest(ctx context.Context, req *QwenGenerateRequest, onChunk func(string)) error {
	url := fmt.Sprintf("%s/services/aigc/multimodal-generation/generation", p.config.BaseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk struct {
			Output struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			} `json:"output"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode stream chunk: %w", err)
		}

		if len(chunk.Output.Choices) > 0 && chunk.Output.Choices[0].Message.Content != "" {
			onChunk(chunk.Output.Choices[0].Message.Content)
		}
	}

	return nil
}

// convertQwenResponse converts Qwen API response to MultimodalOutput.
func (p *QwenProvider) convertQwenResponse(ctx context.Context, resp *QwenGenerateResponse, inputID string) (*types.MultimodalOutput, error) {
	if len(resp.Output.Choices) == 0 {
		return nil, errors.New("Qwen response has no choices")
	}

	choice := resp.Output.Choices[0]
	var content string

	// Extract content from message
	switch msgContent := choice.Message.Content.(type) {
	case string:
		content = msgContent
	case []interface{}:
		// Handle array of content parts
		for _, part := range msgContent {
			if partMap, ok := part.(map[string]interface{}); ok {
				if text, ok := partMap["text"].(string); ok {
					content += text
				}
			}
		}
	default:
		return nil, fmt.Errorf("unexpected content type: %T", msgContent)
	}

	if content == "" {
		return nil, errors.New("no content in Qwen response")
	}

	output := &types.MultimodalOutput{
		ID:      uuid.New().String(),
		InputID: inputID,
		ContentBlocks: []*types.ContentBlock{
			{
				Type:     "text",
				Data:     []byte(content),
				Format:   "text/plain",
				MIMEType: "text/plain",
				Size:     int64(len(content)),
			},
		},
		Metadata: map[string]any{
			"provider": "qwen",
			"model":    p.config.Model,
			"usage":    resp.Usage,
		},
		Confidence: 0.95,
		Provider:   "qwen",
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
	return strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "429")
}

// logWithOTELContext logs with OTEL context information.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, args ...any) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		args = append(args,
			"trace_id", span.SpanContext().TraceID().String(),
			"span_id", span.SpanContext().SpanID().String(),
		)
	}

	logger := slog.Default()
	switch level {
	case slog.LevelDebug:
		logger.Debug(msg, args...)
	case slog.LevelInfo:
		logger.Info(msg, args...)
	case slog.LevelWarn:
		logger.Warn(msg, args...)
	case slog.LevelError:
		logger.Error(msg, args...)
	}
}
