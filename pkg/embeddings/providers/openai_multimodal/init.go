package openai_multimodal

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
	// Register OpenAI multimodal provider with the global registry
	registry.GetRegistry().Register("openai_multimodal", func(ctx context.Context, config any) (embeddingsiface.Embedder, error) {
		// Use reflection to access config.OpenAI without importing embeddings
		configValue := reflect.ValueOf(config)
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		// Get the OpenAI field using reflection (reuse OpenAI config)
		openaiField := configValue.FieldByName("OpenAI")
		if !openaiField.IsValid() || openaiField.IsNil() {
			return nil, errors.New("OpenAI provider is not configured or disabled")
		}

		openaiValue := openaiField.Elem()

		// Check Enabled field
		enabledField := openaiValue.FieldByName("Enabled")
		if !enabledField.IsValid() || !enabledField.Bool() {
			return nil, errors.New("OpenAI provider is not configured or disabled")
		}

		// Extract config values using reflection
		apiKeyField := openaiValue.FieldByName("APIKey")
		modelField := openaiValue.FieldByName("Model")
		baseURLField := openaiValue.FieldByName("BaseURL")
		timeoutField := openaiValue.FieldByName("Timeout")
		maxRetriesField := openaiValue.FieldByName("MaxRetries")

		providerConfig := &Config{
			APIKey:     apiKeyField.String(),
			Model:      modelField.String(),
			BaseURL:    baseURLField.String(),
			Timeout:    timeoutField.Interface().(time.Duration),
			MaxRetries: int(maxRetriesField.Int()),
			Enabled:    enabledField.Bool(),
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai_multimodal")
		return NewOpenAIMultimodalEmbedder(providerConfig, tracer)
	})
}
