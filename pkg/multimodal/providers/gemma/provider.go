// Package gemma provides Google Gemma provider implementation for multimodal models.
package gemma

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

// GemmaProvider implements the MultimodalModel interface for Google Gemma.
type GemmaProvider struct {
	config       *Config
	capabilities *ModalityCapabilities
	httpClient   *http.Client
}

// NewGemmaProvider creates a new Gemma multimodal provider.
func NewGemmaProvider(gemmaConfig *Config) (iface.MultimodalModel, error) {
	if err := gemmaConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Gemma configuration: %w", err)
	}

	// Define Gemma capabilities (supports text, image, audio, video)
	capabilities := &ModalityCapabilities{
		Text:  true,
		Image: true,
		Audio: true,
		Video: true,

		MaxImageSize: 20 * 1024 * 1024,  // 20MB
		MaxAudioSize: 25 * 1024 * 1024,  // 25MB
		MaxVideoSize: 100 * 1024 * 1024, // 100MB

		SupportedImageFormats: []string{"png", "jpeg", "jpg", "webp"},
		SupportedAudioFormats: []string{"mp3", "wav", "m4a", "ogg"},
		SupportedVideoFormats: []string{"mp4", "webm", "mov"},
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: gemmaConfig.Timeout,
	}

	provider := &GemmaProvider{
		config:       gemmaConfig,
		capabilities: capabilities,
		httpClient:   httpClient,
	}

	return provider, nil
}

// Process processes a multimodal input using Gemma's API.
func (p *GemmaProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemma")
	ctx, span := tracer.Start(ctx, "gemma.Process",
		trace.WithAttributes(
			attribute.String("provider", "gemma"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	startTime := time.Now()

	// Convert multimodal input to Gemma API format
	contents, err := p.convertToGemmaContents(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Gemma request
	req := &gemmaGenerateRequest{
		Contents:         contents,
		GenerationConfig: &gemmaGenerationConfig{},
	}

	// Make API call with retry logic
	var resp *gemmaGenerateResponse
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

		resp, lastErr = p.makeAPIRequest(ctx, "generateContent", req)
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
		return nil, fmt.Errorf("Gemma API call failed after %d attempts: %w", maxRetries, lastErr)
	}

	// Convert Gemma response to MultimodalOutput
	output, err := p.convertGemmaResponse(ctx, resp, input.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert Gemma response: %w", err)
	}

	duration := time.Since(startTime)
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
	span.SetStatus(codes.Ok, "")

	// Log with OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		slog.Info("Gemma multimodal processing completed",
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
			"input_id", input.ID,
			"output_id", output.ID)
	} else {
		slog.Info("Gemma multimodal processing completed",
			"input_id", input.ID,
			"output_id", output.ID)
	}

	return output, nil
}

// ProcessStream processes a multimodal input with streaming support.
func (p *GemmaProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemma")
	ctx, span := tracer.Start(ctx, "gemma.ProcessStream",
		trace.WithAttributes(
			attribute.String("provider", "gemma"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
		))
	defer span.End()

	// Convert multimodal input to Gemma API format
	contents, err := p.convertToGemmaContents(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Gemma request
	req := &gemmaGenerateRequest{
		Contents:         contents,
		GenerationConfig: &gemmaGenerationConfig{},
	}

	outputChan := make(chan *types.MultimodalOutput, 10)

	go func() {
		defer close(outputChan)

		// Make streaming API request
		url := fmt.Sprintf("%s/models/%s:streamGenerateContent?key=%s", p.config.BaseURL, p.config.Model, p.config.APIKey)

		reqBody, err := json.Marshal(req)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}

		var accumulatedText strings.Builder
		outputID := uuid.New().String()

		// Parse streaming response
		decoder := json.NewDecoder(resp.Body)
		for {
			var streamResp struct {
				Candidates []gemmaCandidate `json:"candidates"`
			}

			if err := decoder.Decode(&streamResp); err != nil {
				if err == io.EOF {
					break
				}
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return
			}

			if len(streamResp.Candidates) > 0 {
				candidate := streamResp.Candidates[0]
				if len(candidate.Content.Parts) > 0 {
					text := candidate.Content.Parts[0].Text
					if text != "" {
						accumulatedText.WriteString(text)

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
							Metadata:   map[string]any{"provider": "gemma", "model": p.config.Model},
							Confidence: 0.95,
							Provider:   "gemma",
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
				Metadata:   map[string]any{"provider": "gemma", "model": p.config.Model},
				Confidence: 0.95,
				Provider:   "gemma",
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

// GetCapabilities returns the capabilities of the Gemma provider.
func (p *GemmaProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemma")
	_, span := tracer.Start(ctx, "gemma.GetCapabilities",
		trace.WithAttributes(
			attribute.String("provider", "gemma"),
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

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// SupportsModality checks if Gemma supports a specific modality.
func (p *GemmaProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemma")
	_, span := tracer.Start(ctx, "gemma.SupportsModality",
		trace.WithAttributes(
			attribute.String("provider", "gemma"),
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
func (p *GemmaProvider) CheckHealth(ctx context.Context) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemma")
	ctx, span := tracer.Start(ctx, "gemma.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider", "gemma"),
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

// Gemma API request/response structures.
type gemmaGenerateRequest struct {
	GenerationConfig *gemmaGenerationConfig `json:"generationConfig,omitempty"`
	Contents         []gemmaContent         `json:"contents"`
}

type gemmaContent struct {
	Role  string      `json:"role"`
	Parts []gemmaPart `json:"parts"`
}

type gemmaPart struct {
	InlineData *gemmaInlineData `json:"inlineData,omitempty"`
	FileData   *gemmaFileData   `json:"fileData,omitempty"`
	Text       string           `json:"text,omitempty"`
}

type gemmaInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

type gemmaFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

type gemmaGenerationConfig struct {
	Temperature     *float32 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	TopP            *float32 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type gemmaGenerateResponse struct {
	UsageMetadata *gemmaUsageMetadata `json:"usageMetadata,omitempty"`
	Candidates    []gemmaCandidate    `json:"candidates"`
}

type gemmaCandidate struct {
	FinishReason string       `json:"finishReason,omitempty"`
	Content      gemmaContent `json:"content"`
}

type gemmaUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// convertToGemmaContents converts multimodal content blocks to Gemma format.
func (p *GemmaProvider) convertToGemmaContents(ctx context.Context, blocks []*types.ContentBlock) ([]gemmaContent, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no content blocks provided")
	}

	// Group blocks into a single user message with multiple parts
	var parts []gemmaPart

	for _, block := range blocks {
		switch block.Type {
		case "text":
			parts = append(parts, gemmaPart{
				Text: string(block.Data),
			})

		case "image":
			var inlineData *gemmaInlineData
			if len(block.Data) > 0 {
				mimeType := block.MIMEType
				if mimeType == "" {
					mimeType = "image/png" // Default
				}
				encoded := base64.StdEncoding.EncodeToString(block.Data)
				inlineData = &gemmaInlineData{
					MimeType: mimeType,
					Data:     encoded,
				}
			} else if block.URL != "" {
				// For URLs, we'd need to use FileData with a file URI
				// For now, we'll use inline data by fetching the URL
				// In production, you might want to use Gemma's file upload API
				return nil, errors.New("image URL support requires file upload API (not implemented)")
			}

			if inlineData != nil {
				parts = append(parts, gemmaPart{
					InlineData: inlineData,
				})
			}

		case "audio", "video":
			// Gemma supports audio/video through inline data or file URIs
			if len(block.Data) > 0 {
				mimeType := block.MIMEType
				if mimeType == "" {
					if block.Type == "audio" {
						mimeType = "audio/mpeg"
					} else {
						mimeType = "video/mp4"
					}
				}
				encoded := base64.StdEncoding.EncodeToString(block.Data)
				parts = append(parts, gemmaPart{
					InlineData: &gemmaInlineData{
						MimeType: mimeType,
						Data:     encoded,
					},
				})
			} else if block.URL != "" {
				return nil, fmt.Errorf("%s URL support requires file upload API (not implemented)", block.Type)
			}
		}
	}

	if len(parts) == 0 {
		return nil, errors.New("no valid parts created from content blocks")
	}

	// Create a single user message with all parts
	contents := []gemmaContent{
		{
			Role:  "user",
			Parts: parts,
		},
	}

	return contents, nil
}

// makeAPIRequest makes an HTTP request to the Gemma API.
func (p *GemmaProvider) makeAPIRequest(ctx context.Context, endpoint string, req *gemmaGenerateRequest) (*gemmaGenerateResponse, error) {
	url := fmt.Sprintf("%s/models/%s:%s?key=%s", p.config.BaseURL, p.config.Model, endpoint, p.config.APIKey)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var gemmaResp gemmaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&gemmaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &gemmaResp, nil
}

// convertGemmaResponse converts Gemma API response to MultimodalOutput.
func (p *GemmaProvider) convertGemmaResponse(ctx context.Context, resp *gemmaGenerateResponse, inputID string) (*types.MultimodalOutput, error) {
	if len(resp.Candidates) == 0 {
		return nil, errors.New("Gemma response has no candidates")
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, errors.New("Gemma candidate has no parts")
	}

	// Extract text from response
	var textParts []string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
	}

	content := strings.Join(textParts, "")

	// Create content block from response
	responseBlock := &types.ContentBlock{
		Type:     "text",
		Data:     []byte(content),
		Format:   "text/plain",
		MIMEType: "text/plain",
		Size:     int64(len(content)),
		Metadata: map[string]any{
			"finish_reason": candidate.FinishReason,
		},
	}

	output := &types.MultimodalOutput{
		ID:            uuid.New().String(),
		InputID:       inputID,
		ContentBlocks: []*types.ContentBlock{responseBlock},
		Metadata: map[string]any{
			"provider":      "gemma",
			"model":         p.config.Model,
			"usage":         resp.UsageMetadata,
			"finish_reason": candidate.FinishReason,
		},
		Confidence: 0.95, // Gemma doesn't provide confidence scores
		Provider:   "gemma",
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
