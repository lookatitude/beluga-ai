package google_multimodal

import (
	"context"
	"errors"
	"reflect"
	"time"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/registry"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register Google multimodal provider with the global registry
	// Note: This uses the same config structure as the base Google provider
	// For now, we'll use a placeholder config structure
	registry.GetRegistry().Register("google_multimodal", func(ctx context.Context, config any) (embeddingsiface.Embedder, error) {
		// Use reflection to access config fields
		configValue := reflect.ValueOf(config)
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		// Try to find Google or Gemini config field
		// For now, we'll create a basic config from available fields
		apiKeyField := configValue.FieldByName("APIKey")
		modelField := configValue.FieldByName("Model")
		baseURLField := configValue.FieldByName("BaseURL")
		timeoutField := configValue.FieldByName("Timeout")
		maxRetriesField := configValue.FieldByName("MaxRetries")
		enabledField := configValue.FieldByName("Enabled")

		if !apiKeyField.IsValid() || apiKeyField.String() == "" {
			return nil, errors.New("Google API key is required")
		}

		providerConfig := &Config{
			APIKey:     apiKeyField.String(),
			Model:      "text-embedding-004", // Default multimodal model
			BaseURL:    "",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			Enabled:    true,
		}

		if modelField.IsValid() {
			providerConfig.Model = modelField.String()
		}
		if baseURLField.IsValid() {
			providerConfig.BaseURL = baseURLField.String()
		}
		if timeoutField.IsValid() {
			providerConfig.Timeout = timeoutField.Interface().(time.Duration)
		}
		if maxRetriesField.IsValid() {
			providerConfig.MaxRetries = int(maxRetriesField.Int())
		}
		if enabledField.IsValid() {
			providerConfig.Enabled = enabledField.Bool()
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings/providers/google_multimodal")
		return NewGoogleMultimodalEmbedder(providerConfig, tracer)
	})
}
