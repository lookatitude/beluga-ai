// Package openai provides OpenAI provider implementation for multimodal models.
package openai

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
)

func init() {
	// Register OpenAI provider with the global registry
	// Use registry package directly to avoid import cycles
	// Use reflection to access config fields without importing multimodal package
	registry.GetRegistry().Register("openai", func(ctx context.Context, config any) (iface.MultimodalModel, error) {
		// Use reflection to access config fields without importing multimodal package
		configValue := reflect.ValueOf(config)
		if configValue.Kind() == reflect.Ptr {
			configValue = configValue.Elem()
		}

		// Extract config values using reflection
		providerField := configValue.FieldByName("Provider")
		modelField := configValue.FieldByName("Model")
		apiKeyField := configValue.FieldByName("APIKey")
		baseURLField := configValue.FieldByName("BaseURL")
		timeoutField := configValue.FieldByName("Timeout")
		maxRetriesField := configValue.FieldByName("MaxRetries")
		providerSpecificField := configValue.FieldByName("ProviderSpecific")

		if !providerField.IsValid() || !modelField.IsValid() || !apiKeyField.IsValid() {
			return nil, errors.New("invalid config structure for OpenAI provider")
		}

		multimodalConfig := MultimodalConfig{
			Provider:         providerField.String(),
			Model:            modelField.String(),
			APIKey:           apiKeyField.String(),
			BaseURL:          baseURLField.String(),
			Timeout:          getDurationFromValue(timeoutField),
			MaxRetries:       int(maxRetriesField.Int()),
			ProviderSpecific: getMapFromValue(providerSpecificField),
		}

		openaiConfig := FromMultimodalConfig(multimodalConfig)
		if err := openaiConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid OpenAI configuration: %w", err)
		}

		return NewOpenAIProvider(openaiConfig)
	})
}

// getDurationFromValue extracts a time.Duration from a reflect.Value.
func getDurationFromValue(v reflect.Value) time.Duration {
	if !v.IsValid() {
		return 0
	}
	if v.CanInterface() {
		if d, ok := v.Interface().(time.Duration); ok {
			return d
		}
	}
	return 0
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
