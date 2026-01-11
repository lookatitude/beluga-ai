package xai

import (
	"context"
	"reflect"
	"time"

	multimodaliface "github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
	"go.opentelemetry.io/otel"
)

func init() {
	// Register xAI provider with the global registry
	// Use reflection to avoid import cycles
	registry.GetRegistry().Register("xai", func(ctx context.Context, config any) (multimodaliface.MultimodalModel, error) {
		tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/providers/xai")
		ctx, span := tracer.Start(ctx, "xai.init")
		defer span.End()

		// Use reflection to extract config fields
		configValue := reflect.ValueOf(config)
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		// Extract xAI config from multimodal config
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

		// Try to get ProviderSpecific field
		providerSpecificField := configValue.FieldByName("ProviderSpecific")
		if providerSpecificField.IsValid() && providerSpecificField.Kind() == reflect.Map {
			multimodalConfig.ProviderSpecific = getMapFromValue(providerSpecificField)
		}

		// Convert to xAI config
		xaiConfig := FromMultimodalConfig(multimodalConfig)

		// Create provider
		return NewXAIProvider(xaiConfig)
	})
}

// getMapFromValue extracts a map[string]any from a reflect.Value.
func getMapFromValue(v reflect.Value) map[string]any {
	if !v.IsValid() || v.IsNil() {
		return nil
	}
	if v.Kind() == reflect.Map {
		result := make(map[string]any)
		for _, key := range v.MapKeys() {
			result[key.String()] = v.MapIndex(key).Interface()
		}
		return result
	}
	return nil
}
