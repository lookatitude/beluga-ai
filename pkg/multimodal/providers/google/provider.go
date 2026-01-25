// Package google provides Google Vertex AI provider implementation for multimodal models.
package google

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

// GoogleProvider implements the MultimodalModel interface for Google Vertex AI.
type GoogleProvider struct {
	config       *Config
	capabilities *ModalityCapabilities
	httpClient   *http.Client
}

// NewGoogleProvider creates a new Google Vertex AI multimodal provider.
func NewGoogleProvider(googleConfig *Config) (iface.MultimodalModel, error) {
	if err := googleConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Google Vertex AI configuration: %w", err)
	}

	// Define Google Vertex AI capabilities (supports text, image, audio, video)
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
		Timeout: googleConfig.Timeout,
	}

	provider := &GoogleProvider{
		config:       googleConfig,
		capabilities: capabilities,
		httpClient:   httpClient,
	}

	return provider, nil
}

// Process processes a multimodal input using Google Vertex AI's API.
func (p *GoogleProvider) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/google")
	ctx, span := tracer.Start(ctx, "google.Process",
		trace.WithAttributes(
			attribute.String("provider", "google"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	startTime := time.Now()

	// Convert multimodal input to Vertex AI format
	contents, err := p.convertToVertexAIContents(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Vertex AI request
	req := &vertexAIGenerateRequest{
		Contents:         contents,
		GenerationConfig: &vertexAIGenerationConfig{},
	}

	// Make API call with retry logic
	var resp *vertexAIGenerateResponse
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

		logWithOTELContext(ctx, slog.LevelWarn, "Google Vertex AI API call failed, retrying",
			"error", lastErr,
			"attempt", attempt+1,
			"max_retries", maxRetries)
	}

	if lastErr != nil {
		span.RecordError(lastErr)
		span.SetStatus(codes.Error, lastErr.Error())
		return nil, fmt.Errorf("Google Vertex AI API call failed: %w", lastErr)
	}

	// Convert Vertex AI response to MultimodalOutput
	output, err := p.convertVertexAIResponse(ctx, resp, input.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	duration := time.Since(startTime)
	span.SetAttributes(attribute.Int64("duration_ms", duration.Milliseconds()))
	span.SetStatus(codes.Ok, "")

	logWithOTELContext(ctx, slog.LevelInfo, "Google Vertex AI multimodal processing completed",
		"input_id", input.ID,
		"output_id", output.ID,
		"duration_ms", duration.Milliseconds())

	return output, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
func (p *GoogleProvider) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/google")
	ctx, span := tracer.Start(ctx, "google.ProcessStream",
		trace.WithAttributes(
			attribute.String("provider", "google"),
			attribute.String("model", p.config.Model),
			attribute.String("input_id", input.ID),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	// Convert multimodal input to Vertex AI format
	contents, err := p.convertToVertexAIContents(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Build Vertex AI streaming request
	req := &vertexAIGenerateRequest{
		Contents:         contents,
		GenerationConfig: &vertexAIGenerationConfig{},
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
					Metadata:   map[string]any{"provider": "google", "model": p.config.Model},
					Confidence: 0.95,
					Provider:   "google",
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
				Metadata:   map[string]any{"provider": "google", "model": p.config.Model},
				Confidence: 0.95,
				Provider:   "google",
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

// GetCapabilities returns the capabilities of the Google Vertex AI provider.
func (p *GoogleProvider) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/google")
	_, span := tracer.Start(ctx, "google.GetCapabilities",
		trace.WithAttributes(
			attribute.String("provider", "google"),
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

// SupportsModality checks if Google Vertex AI supports a specific modality.
func (p *GoogleProvider) SupportsModality(ctx context.Context, modality string) (bool, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/google")
	_, span := tracer.Start(ctx, "google.SupportsModality",
		trace.WithAttributes(
			attribute.String("provider", "google"),
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
func (p *GoogleProvider) CheckHealth(ctx context.Context) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/google")
	ctx, span := tracer.Start(ctx, "google.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider", "google"),
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

// Vertex AI API request/response structures.
type vertexAIGenerateRequest struct {
	GenerationConfig *vertexAIGenerationConfig `json:"generationConfig,omitempty"`
	Contents         []vertexAIContent         `json:"contents"`
}

type vertexAIContent struct {
	Role  string         `json:"role"`
	Parts []vertexAIPart `json:"parts"`
}

type vertexAIPart struct {
	InlineData *vertexAIInlineData `json:"inlineData,omitempty"`
	FileData   *vertexAIFileData   `json:"fileData,omitempty"`
	Text       string              `json:"text,omitempty"`
}

type vertexAIInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

type vertexAIFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

type vertexAIGenerationConfig struct {
	Temperature     *float32 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	TopP            *float32 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type vertexAIGenerateResponse struct {
	UsageMetadata *vertexAIUsageMetadata `json:"usageMetadata,omitempty"`
	Candidates    []vertexAICandidate    `json:"candidates"`
}

type vertexAICandidate struct {
	FinishReason string          `json:"finishReason,omitempty"`
	Content      vertexAIContent `json:"content"`
}

type vertexAIUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// convertToVertexAIContents converts multimodal content blocks to Vertex AI format.
func (p *GoogleProvider) convertToVertexAIContents(ctx context.Context, blocks []*types.ContentBlock) ([]vertexAIContent, error) {
	if len(blocks) == 0 {
		return nil, errors.New("no content blocks provided")
	}

	var parts []vertexAIPart

	for _, block := range blocks {
		switch block.Type {
		case "text":
			parts = append(parts, vertexAIPart{
				Text: string(block.Data),
			})

		case "image":
			var inlineData *vertexAIInlineData
			if len(block.Data) > 0 {
				mimeType := block.MIMEType
				if mimeType == "" {
					mimeType = "image/png"
				}
				encoded := base64.StdEncoding.EncodeToString(block.Data)
				inlineData = &vertexAIInlineData{
					MimeType: mimeType,
					Data:     encoded,
				}
			} else if block.URL != "" {
				return nil, errors.New("image URL support requires file upload API (not implemented)")
			}

			if inlineData != nil {
				parts = append(parts, vertexAIPart{
					InlineData: inlineData,
				})
			}

		case "audio", "video":
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
				parts = append(parts, vertexAIPart{
					InlineData: &vertexAIInlineData{
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

	contents := []vertexAIContent{
		{
			Role:  "user",
			Parts: parts,
		},
	}

	return contents, nil
}

// makeAPIRequest makes an HTTP request to the Vertex AI API.
func (p *GoogleProvider) makeAPIRequest(ctx context.Context, req *vertexAIGenerateRequest) (*vertexAIGenerateResponse, error) {
	// Vertex AI endpoint format: projects/{project}/locations/{location}/publishers/google/models/{model}:generateContent
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		p.config.ProjectID, p.config.Location, p.config.Model)

	// Build base URL from location
	baseURL := p.config.BaseURL
	if baseURL == "" || strings.Contains(baseURL, "us-central1") {
		// Default to location-specific endpoint
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1", p.config.Location)
	}

	url := fmt.Sprintf("%s/%s", baseURL, endpoint)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add authentication - Vertex AI uses OAuth2 tokens or API keys
	if p.config.APIKey != "" {
		url = fmt.Sprintf("%s?key=%s", url, p.config.APIKey)
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
	} else {
		// For service account authentication, the token should be in the Authorization header
		// This would typically be handled by Google Cloud client libraries
		// For now, we'll require API key or assume credentials are set via environment
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var vertexAIResp vertexAIGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&vertexAIResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &vertexAIResp, nil
}

// makeStreamingRequest makes a streaming HTTP request to Vertex AI API.
func (p *GoogleProvider) makeStreamingRequest(ctx context.Context, req *vertexAIGenerateRequest, onChunk func(string)) error {
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s:streamGenerateContent",
		p.config.ProjectID, p.config.Location, p.config.Model)

	baseURL := p.config.BaseURL
	if baseURL == "" || strings.Contains(baseURL, "us-central1") {
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1", p.config.Location)
	}

	url := fmt.Sprintf("%s/%s", baseURL, endpoint)
	if p.config.APIKey != "" {
		url = fmt.Sprintf("%s?key=%s", url, p.config.APIKey)
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse streaming response
	decoder := json.NewDecoder(resp.Body)
	for {
		var streamResp struct {
			Candidates []vertexAICandidate `json:"candidates"`
		}

		if err := decoder.Decode(&streamResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode stream chunk: %w", err)
		}

		if len(streamResp.Candidates) > 0 {
			candidate := streamResp.Candidates[0]
			if len(candidate.Content.Parts) > 0 {
				text := candidate.Content.Parts[0].Text
				if text != "" {
					onChunk(text)
				}
			}
		}
	}

	return nil
}

// convertVertexAIResponse converts Vertex AI API response to MultimodalOutput.
func (p *GoogleProvider) convertVertexAIResponse(ctx context.Context, resp *vertexAIGenerateResponse, inputID string) (*types.MultimodalOutput, error) {
	if len(resp.Candidates) == 0 {
		return nil, errors.New("Vertex AI response has no candidates")
	}

	candidate := resp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return nil, errors.New("Vertex AI candidate has no parts")
	}

	var textParts []string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
	}

	content := strings.Join(textParts, "")

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
			"provider":      "google",
			"model":         p.config.Model,
			"usage":         resp.UsageMetadata,
			"finish_reason": candidate.FinishReason,
		},
		Confidence: 0.95,
		Provider:   "google",
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
