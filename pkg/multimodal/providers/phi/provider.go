// Package phi provides Phi (Mistral AI) provider implementation for multimodal models.
package phi

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

// PhiProvider implements the MultimodalModel interface for Phi.
type PhiProvider struct {
	config       *Config
	capabilities *ModalityCapabilities
	httpClient   *http.Client
}

// NewPhiProvider creates a new Phi multimodal provider.
func NewPhiProvider(phiConfig *Config) (iface.MultimodalModel, error) {
	if err := phiConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Phi configuration: %w", err)
	}

	// Define Phi capabilities (supports text, image, audio, video)
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
		Timeout: phiConfig.Timeout,
	}

	provider := &PhiProvider{
		config:       phiConfig,
		capabilities: capabilities,
		httpClient:   httpClient,
	}

	return provider, nil
}

// Process processes a multimodal input using Phi's API.
func (p *PhiProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/phi")
	ctx, span := tracer.Start(ctx, "phi.Process",
		trace.WithAttributes(
			attribute.String("provider", "phi"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	startTime := time.Now()

	// Convert multimodal input to Phi API format
	messages, err := p.convertToPhiMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Phi request
	req := &phiChatRequest{
		Model:    p.config.Model,
		Messages: messages,
		Stream:   false,
	}

	// Make API call with retry logic
	var resp *phiChatResponse
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

		logWithOTELContext(ctx, slog.LevelWarn, "Phi API call failed, retrying",
			"error", lastErr,
			"attempt", attempt+1,
			"max_retries", maxRetries)
	}

	if lastErr != nil {
		span.RecordError(lastErr)
		span.SetStatus(codes.Error, lastErr.Error())
		return nil, fmt.Errorf("Phi API call failed: %w", lastErr)
	}

	// Convert response to multimodal output
	output, err := p.convertPhiResponse(ctx, resp, input.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	duration := time.Since(startTime)
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
	span.SetStatus(codes.Ok, "")

	logWithOTELContext(ctx, slog.LevelInfo, "Phi multimodal processing completed",
		"input_id", input.ID,
		"output_id", output.ID,
		"duration_ms", duration.Milliseconds())

	return output, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
func (p *PhiProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/phi")
	ctx, span := tracer.Start(ctx, "phi.ProcessStream",
		trace.WithAttributes(
			attribute.String("provider", "phi"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	// Convert multimodal input to Phi API format
	messages, err := p.convertToPhiMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Phi streaming request
	req := &phiChatRequest{
		Model:    p.config.Model,
		Messages: messages,
		Stream:   true,
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
					Metadata:   map[string]any{"provider": "phi", "model": p.config.Model},
					Confidence: 0.95,
					Provider:   "phi",
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
				Metadata:   map[string]any{"provider": "phi", "model": p.config.Model},
				Confidence: 0.95,
				Provider:   "phi",
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

// GetCapabilities returns the capabilities of the Phi provider.
func (p *PhiProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/phi")
	_, span := tracer.Start(ctx, "phi.GetCapabilities",
		trace.WithAttributes(
			attribute.String("provider", "phi"),
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

// SupportsModality checks if Phi supports a specific modality.
func (p *PhiProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/phi")
	_, span := tracer.Start(ctx, "phi.SupportsModality",
		trace.WithAttributes(
			attribute.String("provider", "phi"),
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
		span.SetStatus(codes.Error, "unknown modality: "+modality)
		return false, fmt.Errorf("unknown modality: %s", modality)
	}

	span.SetAttributes(attribute.Bool("supported", supported))
	span.SetStatus(codes.Ok, "")
	return supported, nil
}

// CheckHealth performs a health check and returns an error if the provider is unhealthy.
func (p *PhiProvider) CheckHealth(ctx context.Context) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/phi")
	ctx, span := tracer.Start(ctx, "phi.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider", "phi"),
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

// Phi API request/response structures.
type phiChatRequest struct {
	Model    string       `json:"model"`
	Messages []phiMessage `json:"messages"`
	Stream   bool         `json:"stream,omitempty"`
}

type phiMessage struct {
	Role    string       `json:"role"`
	Content []phiContent `json:"content"`
}

type phiContent struct {
	ImageURL *phiImageURL `json:"image_url,omitempty"`
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
}

type phiImageURL struct {
	URL string `json:"url"` // data:image/png;base64,... or http://...
}

type phiChatResponse struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	Model   string      `json:"model"`
	Choices []phiChoice `json:"choices"`
	Usage   phiUsage    `json:"usage"`
	Created int64       `json:"created"`
}

type phiChoice struct {
	FinishReason string     `json:"finish_reason"`
	Message      phiMessage `json:"message"`
	Index        int        `json:"index"`
}

type phiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// convertToPhiMessages converts multimodal content blocks to Phi message format.
func (p *PhiProvider) convertToPhiMessages(ctx context.Context, blocks []*types.ContentBlock) ([]phiMessage, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no content blocks provided")
	}

	var contentParts []phiContent

	for _, block := range blocks {
		switch block.Type {
		case "text":
			contentParts = append(contentParts, phiContent{
				Type: "text",
				Text: string(block.Data),
			})

		case "image":
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
				return nil, errors.New("image block has no URL or data")
			}

			contentParts = append(contentParts, phiContent{
				Type: "image_url",
				ImageURL: &phiImageURL{
					URL: imageURL,
				},
			})

		case "audio", "video":
			// Phi may support audio/video in future, for now convert to text description
			description := fmt.Sprintf("[%s content: %s]", block.Type, block.Format)
			contentParts = append(contentParts, phiContent{
				Type: "text",
				Text: description,
			})
		}
	}

	return []phiMessage{
		{
			Role:    "user",
			Content: contentParts,
		},
	}, nil
}

// makeAPIRequest makes an HTTP request to Phi API (Hugging Face Inference).
func (p *PhiProvider) makeAPIRequest(ctx context.Context, req *phiChatRequest) (*phiChatResponse, error) {
	// Hugging Face format: https://api-inference.huggingface.co/models/{model}
	url := fmt.Sprintf("%s/%s", p.config.BaseURL, p.config.Model)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp phiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}

// makeStreamingRequest makes a streaming HTTP request to Phi API (Hugging Face Inference).
func (p *PhiProvider) makeStreamingRequest(ctx context.Context, req *phiChatRequest, onChunk func(string)) error {
	// Hugging Face format: https://api-inference.huggingface.co/models/{model}
	url := fmt.Sprintf("%s/%s", p.config.BaseURL, p.config.Model)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode stream chunk: %w", err)
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			onChunk(chunk.Choices[0].Delta.Content)
		}
	}

	return nil
}

// convertPhiResponse converts Phi API response to MultimodalOutput.
func (p *PhiProvider) convertPhiResponse(ctx context.Context, resp *phiChatResponse, inputID string) (*types.MultimodalOutput, error) {
	if len(resp.Choices) == 0 {
		return nil, errors.New("Phi response has no choices")
	}

	choice := resp.Choices[0]
	var content string

	// Extract text content from message
	var contentSb549 strings.Builder
	for _, contentPart := range choice.Message.Content {
		if contentPart.Type == "text" && contentPart.Text != "" {
			contentSb549.WriteString(contentPart.Text)
		}
	}
	content += contentSb549.String()

	if content == "" {
		return nil, errors.New("no content in Phi response")
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
			"provider":      "phi",
			"model":         p.config.Model,
			"usage":         resp.Usage,
			"finish_reason": choice.FinishReason,
		},
		Confidence: 0.95,
		Provider:   "phi",
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
