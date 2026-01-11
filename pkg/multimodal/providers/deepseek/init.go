package deepseek

import (
	"context"
	"reflect"
	"time"

	multimodaliface "github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register DeepSeek provider with the global registry
	// Use reflection to avoid import cycles
	registry.GetRegistry().Register("deepseek", func(ctx context.Context, config any) (multimodaliface.MultimodalModel, error) {
		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/deepseek")
		ctx, span := tracer.Start(ctx, "deepseek.init")
		defer span.End()

		// Use reflection to extract config fields
		configValue := reflect.ValueOf(config)
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		// Extract DeepSeek config from multimodal config
		multimodalConfig := MultimodalConfig{
			ProviderSpecific: make(map[string]any),
		}

		// Try to get provider field
		providerField := configValue.FieldByName("Provider")
		if providerField.IsValid() {
			multimodalConfig.Provider = providerField.String()
		}

		// Try to get model field
		modelField := configValue.FieldByName("Model")
		if modelField.IsValid() {
			multimodalConfig.Model = modelField.String()
		}

		// Try to get APIKey field
		apiKeyField := configValue.FieldByName("APIKey")
		if apiKeyField.IsValid() {
			multimodalConfig.APIKey = apiKeyField.String()
		}

		// Try to get BaseURL field
		baseURLField := configValue.FieldByName("BaseURL")
		if baseURLField.IsValid() {
			multimodalConfig.BaseURL = baseURLField.String()
		}

		// Try to get Timeout field
		timeoutField := configValue.FieldByName("Timeout")
		if timeoutField.IsValid() {
			multimodalConfig.Timeout = timeoutField.Interface().(time.Duration)
		}

		// Try to get MaxRetries field
		maxRetriesField := configValue.FieldByName("MaxRetries")
		if maxRetriesField.IsValid() {
			multimodalConfig.MaxRetries = int(maxRetriesField.Int())
		}

		// Convert to DeepSeek config
		deepseekConfig := FromMultimodalConfig(multimodalConfig)

		// Create provider
		return NewDeepSeekProvider(deepseekConfig)
	})
}
