// Package multimodal provides factory functions for creating multimodal model instances.
package multimodal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/internal"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// NewMultimodalModel creates a new multimodal model instance using the global registry.
// The providerName must match a registered provider (e.g., "openai", "gemini", "anthropic").
// The config is validated before creating the model. If the provider is not found in the registry,
// a base model with default text-only capabilities is created as a fallback.
//
// Example:
//
//	model, err := NewMultimodalModel(ctx, "openai", Config{
//	    Provider: "openai",
//	    Model:    "gpt-4o",
//	    APIKey:   os.Getenv("OPENAI_API_KEY"),
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewMultimodalModel(ctx context.Context, providerName string, config Config) (iface.MultimodalModel, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal")
	ctx, span := tracer.Start(ctx, "multimodal.NewMultimodalModel",
		trace.WithAttributes(
			attribute.String("provider", providerName),
			attribute.String("model", config.Model),
		))
	defer span.End()

	// Validate config
	if err := config.Validate(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Config validation failed", "error", err)
		return nil, err
	}

	// Try to create from registry first
	model, err := GetRegistry().Create(ctx, providerName, config)
	if err != nil {
		// If provider not found, create a base model with default capabilities
		if err.Error() == fmt.Sprintf("multimodal provider '%s' not found", providerName) {
			logWithOTELContext(ctx, slog.LevelWarn, "Provider not found in registry, creating base model",
				"provider", providerName)

			// Convert Config to map[string]any for internal.NewBaseMultimodalModel
			configMap := map[string]any{
				"Provider":          config.Provider,
				"Model":             config.Model,
				"APIKey":            config.APIKey,
				"stream_chunk_size": int64(1024 * 1024), // 1MB default
			}

			// Create base model with default capabilities using types package
			capabilities := &types.ModalityCapabilities{
				Text:  true,
				Image: false,
				Audio: false,
				Video: false,
			}

			baseModel := internal.NewBaseMultimodalModel(providerName, config.Model, configMap, capabilities)
			span.SetStatus(codes.Ok, "")
			logWithOTELContext(ctx, slog.LevelInfo, "Base multimodal model created",
				"provider", providerName, "model", config.Model)
			return baseModel, nil
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Failed to create multimodal model", "error", err)
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Multimodal model created successfully",
		"provider", providerName, "model", config.Model)
	return model, nil
}

// logWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}
