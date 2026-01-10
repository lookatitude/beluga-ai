package cohere

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/registry"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register Cohere provider with the global registry
	// Use registry package directly to avoid import cycles in tests
	// Use reflection to access config fields without importing embeddings package
	registry.GetRegistry().Register("cohere", func(ctx context.Context, config any) (embeddingsiface.Embedder, error) {
		// Use reflection to access config.Cohere without importing embeddings
		configValue := reflect.ValueOf(config)
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		// Get the Cohere field using reflection
		cohereField := configValue.FieldByName("Cohere")
		if !cohereField.IsValid() || cohereField.IsNil() {
			return nil, errors.New("cohere provider is not configured or disabled")
		}

		cohereValue := cohereField.Elem()

		// Check Enabled field
		enabledField := cohereValue.FieldByName("Enabled")
		if !enabledField.IsValid() || !enabledField.Bool() {
			return nil, errors.New("cohere provider is not configured or disabled")
		}

		// Extract config values using reflection
		apiKeyField := cohereValue.FieldByName("APIKey")
		modelField := cohereValue.FieldByName("Model")
		baseURLField := cohereValue.FieldByName("BaseURL")
		timeoutField := cohereValue.FieldByName("Timeout")
		maxRetriesField := cohereValue.FieldByName("MaxRetries")

		cohereConfig := &Config{
			APIKey:     apiKeyField.String(),
			Model:      modelField.String(),
			BaseURL:    baseURLField.String(),
			Timeout:    timeoutField.Interface().(time.Duration),
			MaxRetries: int(maxRetriesField.Int()),
			Enabled:    enabledField.Bool(),
		}

		// Validate using reflection (call Validate method if it exists)
		validateMethod := cohereValue.MethodByName("Validate")
		if validateMethod.IsValid() {
			results := validateMethod.Call(nil)
			if len(results) > 0 && !results[0].IsNil() {
				if err, ok := results[0].Interface().(error); ok && err != nil {
					return nil, fmt.Errorf("invalid Cohere configuration: %w", err)
				}
			}
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
		return NewCohereEmbedder(cohereConfig, tracer)
	})
}
