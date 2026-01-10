package mock

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/registry"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register Mock provider with the global registry
	// Use registry package directly to avoid import cycles in tests
	// Use reflection to access config fields without importing embeddings package
	registry.GetRegistry().Register("mock", func(ctx context.Context, config any) (embeddingsiface.Embedder, error) {
		// Use reflection to access config.Mock without importing embeddings
		configValue := reflect.ValueOf(config)
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		// Get the Mock field using reflection
		mockField := configValue.FieldByName("Mock")
		if !mockField.IsValid() || mockField.IsNil() {
			return nil, errors.New("mock provider is not configured or disabled")
		}

		mockValue := mockField.Elem()

		// Check Enabled field
		enabledField := mockValue.FieldByName("Enabled")
		if !enabledField.IsValid() || !enabledField.Bool() {
			return nil, errors.New("mock provider is not configured or disabled")
		}

		// Extract config values using reflection
		dimensionField := mockValue.FieldByName("Dimension")
		seedField := mockValue.FieldByName("Seed")
		randomizeNilField := mockValue.FieldByName("RandomizeNil")

		mockConfig := &Config{
			Dimension:    int(dimensionField.Int()),
			Seed:         seedField.Int(),
			RandomizeNil: randomizeNilField.Bool(),
			Enabled:      enabledField.Bool(),
		}

		// Validate using reflection (call Validate method if it exists)
		validateMethod := mockValue.MethodByName("Validate")
		if validateMethod.IsValid() {
			results := validateMethod.Call(nil)
			if len(results) > 0 && !results[0].IsNil() {
				if err, ok := results[0].Interface().(error); ok && err != nil {
					return nil, fmt.Errorf("invalid Mock configuration: %w", err)
				}
			}
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
		return NewMockEmbedder(mockConfig, tracer)
	})
}
