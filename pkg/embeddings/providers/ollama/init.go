//go:build experimental

package ollama

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
	// Register Ollama provider with the global registry (experimental build only)
	// Ollama contains high-severity CVEs (GO-2025-3824, GO-2025-3695) that allow
	// cross-domain token exposure and DoS. This provider is only available when
	// building with the 'experimental' tag: go build -tags experimental
	registry.GetRegistry().Register("ollama", func(ctx context.Context, config any) (embeddingsiface.Embedder, error) {
		// Use reflection to access config.Ollama without importing embeddings
		configValue := reflect.ValueOf(config)
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		// Get the Ollama field using reflection
		ollamaField := configValue.FieldByName("Ollama")
		if !ollamaField.IsValid() || ollamaField.IsNil() {
			return nil, errors.New("ollama provider is not configured or disabled")
		}

		ollamaValue := ollamaField.Elem()

		// Check Enabled field
		enabledField := ollamaValue.FieldByName("Enabled")
		if !enabledField.IsValid() || !enabledField.Bool() {
			return nil, errors.New("ollama provider is not configured or disabled")
		}

		// Extract config values using reflection
		serverURLField := ollamaValue.FieldByName("ServerURL")
		modelField := ollamaValue.FieldByName("Model")
		timeoutField := ollamaValue.FieldByName("Timeout")
		maxRetriesField := ollamaValue.FieldByName("MaxRetries")
		keepAliveField := ollamaValue.FieldByName("KeepAlive")

		ollamaConfig := &Config{
			ServerURL:  serverURLField.String(),
			Model:      modelField.String(),
			Timeout:    timeoutField.Interface().(time.Duration),
			MaxRetries: int(maxRetriesField.Int()),
			KeepAlive:  keepAliveField.String(),
			Enabled:    enabledField.Bool(),
		}

		// Validate using reflection (call Validate method if it exists)
		validateMethod := ollamaValue.MethodByName("Validate")
		if validateMethod.IsValid() {
			results := validateMethod.Call(nil)
			if len(results) > 0 && !results[0].IsNil() {
				if err, ok := results[0].Interface().(error); ok && err != nil {
					return nil, fmt.Errorf("invalid Ollama configuration: %w", err)
				}
			}
		}

		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/embeddings")
		return NewOllamaEmbedder(ollamaConfig, tracer)
	})
}
