package pixtral

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	multimodaliface "github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
)

// errInvalidConfigStructure is returned when the config structure is invalid.
var errInvalidConfigStructure = errors.New("invalid config structure for Pixtral provider")

func init() {
	// Register Pixtral provider with the global registry
	// Use reflection to avoid import cycles with multimodal package
	registry.GetRegistry().Register("pixtral",
		func(_ context.Context, config any) (multimodaliface.MultimodalModel, error) {
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
				return nil, errInvalidConfigStructure
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

			pixtralConfig := FromMultimodalConfig(multimodalConfig)
			if err := pixtralConfig.Validate(); err != nil {
				return nil, fmt.Errorf("invalid Pixtral configuration: %w", err)
			}

			return NewPixtralProvider(pixtralConfig)
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
