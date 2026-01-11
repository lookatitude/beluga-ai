// Package pixtral provides Pixtral (Mistral AI) provider implementation for multimodal models.
package pixtral

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

// PixtralProvider implements the MultimodalModel interface for Pixtral.
type PixtralProvider struct {
	config       *Config
	capabilities *ModalityCapabilities
	httpClient   *http.Client
}

// NewPixtralProvider creates a new Pixtral multimodal provider.
func NewPixtralProvider(pixtralConfig *Config) (iface.MultimodalModel, error) {
	if err := pixtralConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Pixtral configuration: %w", err)
	}

	// Define Pixtral capabilities (supports text, image, audio, video)
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
		Timeout: pixtralConfig.Timeout,
	}

	provider := &PixtralProvider{
		config:       pixtralConfig,
		capabilities: capabilities,
		httpClient:   httpClient,
	}

	return provider, nil
}

// Process processes a multimodal input using Pixtral's API.
func (p *PixtralProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral")
	ctx, span := tracer.Start(ctx, "pixtral.Process",
		trace.WithAttributes(
			attribute.String("provider", "pixtral"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	startTime := time.Now()

	// Convert multimodal input to Pixtral API format
	messages, err := p.convertToPixtralMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Pixtral request
	req := &pixtralChatRequest{
		Model:    p.config.Model,
		Messages: messages,
		Stream:   false,
	}

	// Make API call with retry logic
	var resp *pixtralChatResponse
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

		logWithOTELContext(ctx, slog.LevelWarn, "Pixtral API call failed, retrying",
			"error", lastErr,
			"attempt", attempt+1,
			"max_retries", maxRetries)
	}

	if lastErr != nil {
		span.RecordError(lastErr)
		span.SetStatus(codes.Error, lastErr.Error())
		return nil, fmt.Errorf("pixtral API call failed: %w", lastErr)
	}

	// Convert response to multimodal output
	output, err := p.convertPixtralResponse(ctx, resp, input.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	duration := time.Since(startTime)
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
	span.SetStatus(codes.Ok, "")

	logWithOTELContext(ctx, slog.LevelInfo, "Pixtral multimodal processing completed",
		"input_id", input.ID,
		"output_id", output.ID,
		"duration_ms", duration.Milliseconds())

	return output, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
func (p *PixtralProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral")
	ctx, span := tracer.Start(ctx, "pixtral.ProcessStream",
		trace.WithAttributes(
			attribute.String("provider", "pixtral"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	// Convert multimodal input to Pixtral API format
	messages, err := p.convertToPixtralMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Pixtral streaming request
	req := &pixtralChatRequest{
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
				_, _ = accumulatedText.WriteString(chunk)

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
					Metadata:   map[string]any{"provider": "pixtral", "model": p.config.Model},
					Confidence: 0.95,
					Provider:   "pixtral",
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
				Metadata:   map[string]any{"provider": "pixtral", "model": p.config.Model},
				Confidence: 0.95,
				Provider:   "pixtral",
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

// GetCapabilities returns the capabilities of the Pixtral provider.
func (p *PixtralProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral")
	_, span := tracer.Start(ctx, "pixtral.GetCapabilities",
		trace.WithAttributes(
			attribute.String("provider", "pixtral"),
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

// SupportsModality checks if Pixtral supports a specific modality.
func (p *PixtralProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral")
	_, span := tracer.Start(ctx, "pixtral.SupportsModality",
		trace.WithAttributes(
			attribute.String("provider", "pixtral"),
			attribute.String("model", p.config.Model),
			attribute.String("modality", modality),
		))
	defer span.End()

	var supported bool
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
		err := fmt.Errorf("unknown modality: %s", modality)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	span.SetAttributes(attribute.Bool("supported", supported))
	span.SetStatus(codes.Ok, "")
	return supported, nil
}

// CheckHealth performs a health check and returns an error if the provider is unhealthy.
func (p *PixtralProvider) CheckHealth(ctx context.Context) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/pixtral")
	ctx, span := tracer.Start(ctx, "pixtral.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider", "pixtral"),
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

// Pixtral API request/response structures
type pixtralChatRequest struct {
	Model    string           `json:"model"`
	Messages []pixtralMessage `json:"messages"`
	Stream   bool             `json:"stream,omitempty"`
}

type pixtralMessage struct {
	Role    string           `json:"role"`
	Content []pixtralContent `json:"content"`
}

type pixtralContent struct {
	Type     string           `json:"type"` // "text" or "image_url"
	Text     string           `json:"text,omitempty"`
	ImageURL *pixtralImageURL `json:"image_url,omitempty"`
}

type pixtralImageURL struct {
	URL string `json:"url"` // data:image/png;base64,... or http://...
}

type pixtralChatResponse struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []pixtralChoice `json:"choices"`
	Usage   pixtralUsage    `json:"usage"`
}

type pixtralChoice struct {
	Index        int            `json:"index"`
	Message      pixtralMessage `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

type pixtralUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// convertToPixtralMessages converts multimodal content blocks to Pixtral message format.
func (p *PixtralProvider) convertToPixtralMessages(_ context.Context, blocks []*types.ContentBlock) ([]pixtralMessage, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no content blocks provided")
	}

	var contentParts []pixtralContent

	for _, block := range blocks {
		switch block.Type {
		case "text":
			contentParts = append(contentParts, pixtralContent{
				Type: "text",
				Text: string(block.Data),
			})

		case "image":
			var imageURL string
			switch {
			case len(block.Data) > 0:
				mimeType := block.MIMEType
				if mimeType == "" {
					mimeType = "image/png"
				}
				encoded := base64.StdEncoding.EncodeToString(block.Data)
				imageURL = fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
			case block.URL != "":
				imageURL = block.URL
			default:
				return nil, errors.New("image block has no URL or data")
			}

			contentParts = append(contentParts, pixtralContent{
				Type: "image_url",
				ImageURL: &pixtralImageURL{
					URL: imageURL,
				},
			})

		case "audio", "video":
			// Pixtral may support audio/video in future, for now convert to text description
			description := fmt.Sprintf("[%s content: %s]", block.Type, block.Format)
			contentParts = append(contentParts, pixtralContent{
				Type: "text",
				Text: description,
			})
		default:
			// Unknown content type, skip
		}
	}

	return []pixtralMessage{
		{
			Role:    "user",
			Content: contentParts,
		},
	}, nil
}

// makeAPIRequest makes an HTTP request to Pixtral API.
func (p *PixtralProvider) makeAPIRequest(ctx context.Context, req *pixtralChatRequest) (*pixtralChatResponse, error) {
	apiURL := p.config.BaseURL + "/chat/completions"

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("API request failed with status %d: failed to read error body: %w", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp pixtralChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}

// makeStreamingRequest makes a streaming HTTP request to Pixtral API.
func (p *PixtralProvider) makeStreamingRequest(ctx context.Context, req *pixtralChatRequest, onChunk func(string)) error {
	apiURL := p.config.BaseURL + "/chat/completions"

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("API request failed with status %d: failed to read error body: %w", resp.StatusCode, readErr)
		}
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

// convertPixtralResponse converts Pixtral API response to MultimodalOutput.
func (p *PixtralProvider) convertPixtralResponse(_ context.Context, resp *pixtralChatResponse, inputID string) (*types.MultimodalOutput, error) {
	if len(resp.Choices) == 0 {
		return nil, errors.New("pixtral response has no choices")
	}

	choice := resp.Choices[0]
	var contentBuilder strings.Builder

	// Extract text content from message
	for _, contentPart := range choice.Message.Content {
		if contentPart.Type == "text" && contentPart.Text != "" {
			_, _ = contentBuilder.WriteString(contentPart.Text)
		}
	}

	content := contentBuilder.String()
	if content == "" {
		return nil, errors.New("no content in pixtral response")
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
			"provider":      "pixtral",
			"model":         p.config.Model,
			"usage":         resp.Usage,
			"finish_reason": choice.FinishReason,
		},
		Confidence: 0.95,
		Provider:   "pixtral",
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
	default:
		logger.Info(msg, args...)
	}
}
