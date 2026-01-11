// Package openai provides OpenAI provider implementation for multimodal models.
package openai

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OpenAIProvider implements the MultimodalModel interface for OpenAI.
type OpenAIProvider struct {
	config       *Config
	capabilities *ModalityCapabilities
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

		MaxImageSize: 20 * 1024 * 1024, // 20MB
		MaxAudioSize: 25 * 1024 * 1024, // 25MB
		MaxVideoSize: 100 * 1024 * 1024, // 100MB

		SupportedImageFormats: []string{"png", "jpeg", "jpg", "gif", "webp"},
		SupportedAudioFormats: []string{"mp3", "wav", "m4a", "ogg"},
		SupportedVideoFormats: []string{"mp4", "webm", "mov"},
	}

	provider := &OpenAIProvider{
		config:       openaiConfig,
		capabilities: capabilities,
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

	// Metrics recording would go here if needed
	// For now, we avoid importing multimodal package to prevent import cycles
	startTime := time.Now()
	_ = startTime // Suppress unused variable warning

	// Convert multimodal input to OpenAI API format
	_, err := p.convertToOpenAIMessages(ctx, input.ContentBlocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// TODO: Make actual OpenAI API call
	// For now, return a placeholder response
	// In production, this would call OpenAI's chat completions API with multimodal support

	responseText := fmt.Sprintf("OpenAI multimodal processing (placeholder) for input %s", input.ID)
	responseBlock := &types.ContentBlock{
		Type:     "text",
		Data:     []byte(responseText),
		Format:   "text/plain",
		MIMEType: "text/plain",
		Size:     int64(len(responseText)),
		Metadata: make(map[string]any),
	}

	output := &types.MultimodalOutput{
		ID:            uuid.New().String(),
		InputID:       input.ID,
		ContentBlocks: []*types.ContentBlock{responseBlock},
		Metadata:      map[string]any{"provider": "openai", "model": p.config.Model},
		Confidence:    0.95,
		Provider:      "openai",
		Model:         p.config.Model,
		CreatedAt:     time.Now(),
	}

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

	// TODO: Implement actual streaming
	// For now, return a channel with a single output
	outputChan := make(chan *types.MultimodalOutput, 1)
	
	go func() {
		defer close(outputChan)
		output, err := p.Process(ctx, input)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return
		}
		outputChan <- output
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

// convertToOpenAIMessages converts multimodal content blocks to OpenAI message format.
func (p *OpenAIProvider) convertToOpenAIMessages(ctx context.Context, blocks []*types.ContentBlock) ([]schema.Message, error) {
	messages := make([]schema.Message, 0, len(blocks))

	for _, block := range blocks {
		switch block.Type {
		case "text":
			msg := schema.NewHumanMessage(string(block.Data))
			messages = append(messages, msg)
		case "image":
			// Convert image to OpenAI format (base64 or URL)
			var imageURL string
			var imageData string

			if block.URL != "" {
				imageURL = block.URL
			} else if len(block.Data) > 0 {
				imageData = base64.StdEncoding.EncodeToString(block.Data)
			}

			// Create ImageMessage
			imgMsg := &schema.ImageMessage{
				ImageURL:    imageURL,
				ImageData:   []byte(imageData),
				ImageFormat: block.Format,
			}
			messages = append(messages, imgMsg)
		case "audio":
			// Convert audio to OpenAI format
			audioData := base64.StdEncoding.EncodeToString(block.Data)
			voiceDoc := schema.NewVoiceDocumentWithData(
				[]byte(audioData),
				block.Format,
				"", // No transcript initially
				map[string]string{
					"audio_format": block.Format,
					"mime_type":     block.MIMEType,
				},
			)
			messages = append(messages, voiceDoc)
		case "video":
			// Convert video to OpenAI format
			vidMsg := &schema.VideoMessage{
				VideoURL:    block.URL,
				VideoData:   block.Data,
				VideoFormat: block.Format,
			}
			messages = append(messages, vidMsg)
		}
	}

	return messages, nil
}
