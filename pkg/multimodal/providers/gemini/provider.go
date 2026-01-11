// Package gemini provides Google Gemini provider implementation for multimodal models.
package gemini

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

// GeminiProvider implements the MultimodalModel interface for Google Gemini.
type GeminiProvider struct {
	config       *Config
	capabilities *ModalityCapabilities
	httpClient   *http.Client
}

// NewGeminiProvider creates a new Gemini multimodal provider.
func NewGeminiProvider(geminiConfig *Config) (iface.MultimodalModel, error) {
	if err := geminiConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Gemini configuration: %w", err)
	}

	// Define Gemini capabilities (supports text, image, audio, video)
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
		Timeout: geminiConfig.Timeout,
	}

	provider := &GeminiProvider{
		config:       geminiConfig,
		capabilities: capabilities,
		httpClient:   httpClient,
	}

	return provider, nil
}

// Process processes a multimodal input using Gemini's API.
func (p *GeminiProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemini")
	ctx, span := tracer.Start(ctx, "gemini.Process",
		trace.WithAttributes(
			attribute.String("provider", "gemini"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	startTime := time.Now()

	// Convert multimodal input to Gemini API format
	contents, err := p.convertToGeminiContents(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Gemini request
	req := &geminiGenerateRequest{
		Contents:         contents,
		GenerationConfig: &geminiGenerationConfig{},
	}

	// Make API call with retry logic
	var resp *geminiGenerateResponse
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
		return nil, fmt.Errorf("Gemini API call failed after %d attempts: %w", maxRetries, lastErr)
	}

	// Convert Gemini response to MultimodalOutput
	output, err := p.convertGeminiResponse(ctx, resp, input.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert Gemini response: %w", err)
	}

	duration := time.Since(startTime)
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
	span.SetStatus(codes.Ok, "")

	// Log with OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		slog.Info("Gemini multimodal processing completed",
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
			"input_id", input.ID,
			"output_id", output.ID)
	} else {
		slog.Info("Gemini multimodal processing completed",
			"input_id", input.ID,
			"output_id", output.ID)
	}

	return output, nil
}

// ProcessStream processes a multimodal input with streaming support.
func (p *GeminiProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemini")
	ctx, span := tracer.Start(ctx, "gemini.ProcessStream",
		trace.WithAttributes(
			attribute.String("provider", "gemini"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
		))
	defer span.End()

	// Convert multimodal input to Gemini API format
	contents, err := p.convertToGeminiContents(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Gemini request
	req := &geminiGenerateRequest{
		Contents:         contents,
		GenerationConfig: &geminiGenerationConfig{},
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

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
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
				Candidates []geminiCandidate `json:"candidates"`
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
							Metadata:   map[string]any{"provider": "gemini", "model": p.config.Model},
							Confidence: 0.95,
							Provider:   "gemini",
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
				Metadata:   map[string]any{"provider": "gemini", "model": p.config.Model},
				Confidence: 0.95,
				Provider:   "gemini",
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

// GetCapabilities returns the capabilities of the Gemini provider.
func (p *GeminiProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemini")
	_, span := tracer.Start(ctx, "gemini.GetCapabilities",
		trace.WithAttributes(
			attribute.String("provider", "gemini"),
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

// SupportsModality checks if Gemini supports a specific modality.
func (p *GeminiProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemini")
	_, span := tracer.Start(ctx, "gemini.SupportsModality",
		trace.WithAttributes(
			attribute.String("provider", "gemini"),
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

	span.SetAttributes(attribute.Bool("supported", supported))
	span.SetStatus(codes.Ok, "")
	return supported, nil
}

// CheckHealth performs a health check and returns an error if the provider is unhealthy.
func (p *GeminiProvider) CheckHealth(ctx context.Context) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/gemini")
	ctx, span := tracer.Start(ctx, "gemini.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider", "gemini"),
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

// Gemini API request/response structures
type geminiGenerateRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
	FileData   *geminiFileData   `json:"fileData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

type geminiFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

type geminiGenerationConfig struct {
	Temperature     *float32 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	TopP            *float32 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiGenerateResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// convertToGeminiContents converts multimodal content blocks to Gemini format.
func (p *GeminiProvider) convertToGeminiContents(ctx context.Context, blocks []*types.ContentBlock) ([]geminiContent, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no content blocks provided")
	}

	// Group blocks into a single user message with multiple parts
	var parts []geminiPart

	for _, block := range blocks {
		switch block.Type {
		case "text":
			parts = append(parts, geminiPart{
				Text: string(block.Data),
			})

		case "image":
			var inlineData *geminiInlineData
			if len(block.Data) > 0 {
				mimeType := block.MIMEType
				if mimeType == "" {
					mimeType = "image/png" // Default
				}
				encoded := base64.StdEncoding.EncodeToString(block.Data)
				inlineData = &geminiInlineData{
					MimeType: mimeType,
					Data:     encoded,
				}
			} else if block.URL != "" {
				// For URLs, we'd need to use FileData with a file URI
				// For now, we'll use inline data by fetching the URL
				// In production, you might want to use Gemini's file upload API
				return nil, fmt.Errorf("image URL support requires file upload API (not implemented)")
			}

			if inlineData != nil {
				parts = append(parts, geminiPart{
					InlineData: inlineData,
				})
			}

		case "audio", "video":
			// Gemini supports audio/video through inline data or file URIs
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
				parts = append(parts, geminiPart{
					InlineData: &geminiInlineData{
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
	contents := []geminiContent{
		{
			Role:  "user",
			Parts: parts,
		},
	}

	return contents, nil
}

// makeAPIRequest makes an HTTP request to the Gemini API.
func (p *GeminiProvider) makeAPIRequest(ctx context.Context, endpoint string, req *geminiGenerateRequest) (*geminiGenerateResponse, error) {
	url := fmt.Sprintf("%s/models/%s:%s?key=%s", p.config.BaseURL, p.config.Model, endpoint, p.config.APIKey)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
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

	var geminiResp geminiGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &geminiResp, nil
}

// convertGeminiResponse converts Gemini API response to MultimodalOutput.
func (p *GeminiProvider) convertGeminiResponse(ctx context.Context, resp *geminiGenerateResponse, inputID string) (*types.MultimodalOutput, error) {
	if len(resp.Candidates) == 0 {
		return nil, errors.New("Gemini response has no candidates")
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, errors.New("Gemini candidate has no parts")
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
			"provider":      "gemini",
			"model":         p.config.Model,
			"usage":         resp.UsageMetadata,
			"finish_reason": candidate.FinishReason,
		},
		Confidence: 0.95, // Gemini doesn't provide confidence scores
		Provider:   "gemini",
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
