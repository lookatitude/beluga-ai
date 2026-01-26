// Package voice provides a facade registry for voice sub-package providers.
package voice

import (
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

// FacadeRegistry provides unified access to all voice sub-package registries.
// It follows the Facade pattern to simplify interaction with the voice system.
type FacadeRegistry struct {
	mu sync.RWMutex
}

// Global facade registry instance.
var (
	globalFacade *FacadeRegistry
	facadeOnce   sync.Once
)

// GetFacadeRegistry returns the global facade registry instance.
func GetFacadeRegistry() *FacadeRegistry {
	facadeOnce.Do(func() {
		globalFacade = &FacadeRegistry{}
	})
	return globalFacade
}

// STT returns the STT sub-package registry.
func (f *FacadeRegistry) STT() *stt.Registry {
	return stt.GetRegistry()
}

// TTS returns the TTS sub-package registry.
func (f *FacadeRegistry) TTS() *tts.Registry {
	return tts.GetRegistry()
}

// VAD returns the VAD sub-package registry.
func (f *FacadeRegistry) VAD() *vad.Registry {
	return vad.GetRegistry()
}

// Noise returns the Noise sub-package registry.
func (f *FacadeRegistry) Noise() *noise.Registry {
	return noise.GetRegistry()
}

// Transport returns the Transport sub-package registry.
func (f *FacadeRegistry) Transport() *transport.Registry {
	return transport.GetRegistry()
}

// TurnDetection returns the TurnDetection sub-package registry.
func (f *FacadeRegistry) TurnDetection() *turndetection.Registry {
	return turndetection.GetRegistry()
}

// CreateSTTProvider creates an STT provider using the sub-package registry.
func (f *FacadeRegistry) CreateSTTProvider(name string, config *stt.Config) (iface.STTProvider, error) {
	provider, err := f.STT().GetProvider(name, config)
	if err != nil {
		return nil, NewVoiceErrorWithMessage(
			"CreateSTTProvider",
			ErrCodeProviderNotFound,
			fmt.Sprintf("STT provider '%s' not found", name),
			err,
		)
	}
	return provider, nil
}

// CreateTTSProvider creates a TTS provider using the sub-package registry.
func (f *FacadeRegistry) CreateTTSProvider(name string, config *tts.Config) (iface.TTSProvider, error) {
	provider, err := f.TTS().GetProvider(name, config)
	if err != nil {
		return nil, NewVoiceErrorWithMessage(
			"CreateTTSProvider",
			ErrCodeProviderNotFound,
			fmt.Sprintf("TTS provider '%s' not found", name),
			err,
		)
	}
	return provider, nil
}

// CreateVADProvider creates a VAD provider using the sub-package registry.
func (f *FacadeRegistry) CreateVADProvider(name string, config *vad.Config) (iface.VADProvider, error) {
	provider, err := f.VAD().GetProvider(name, config)
	if err != nil {
		return nil, NewVoiceErrorWithMessage(
			"CreateVADProvider",
			ErrCodeProviderNotFound,
			fmt.Sprintf("VAD provider '%s' not found", name),
			err,
		)
	}
	return provider, nil
}

// CreateNoiseProvider creates a Noise provider using the sub-package registry.
func (f *FacadeRegistry) CreateNoiseProvider(name string, config *noise.Config) (iface.NoiseCancellation, error) {
	provider, err := f.Noise().GetProvider(name, config)
	if err != nil {
		return nil, NewVoiceErrorWithMessage(
			"CreateNoiseProvider",
			ErrCodeProviderNotFound,
			fmt.Sprintf("Noise provider '%s' not found", name),
			err,
		)
	}
	return provider, nil
}

// CreateTransportProvider creates a Transport provider using the sub-package registry.
func (f *FacadeRegistry) CreateTransportProvider(name string, config *transport.Config) (iface.Transport, error) {
	provider, err := f.Transport().GetProvider(name, config)
	if err != nil {
		return nil, NewVoiceErrorWithMessage(
			"CreateTransportProvider",
			ErrCodeProviderNotFound,
			fmt.Sprintf("Transport provider '%s' not found", name),
			err,
		)
	}
	return provider, nil
}

// CreateTurnDetectionProvider creates a TurnDetection provider using the sub-package registry.
func (f *FacadeRegistry) CreateTurnDetectionProvider(name string, config *turndetection.Config) (iface.TurnDetector, error) {
	provider, err := f.TurnDetection().GetProvider(name, config)
	if err != nil {
		return nil, NewVoiceErrorWithMessage(
			"CreateTurnDetectionProvider",
			ErrCodeProviderNotFound,
			fmt.Sprintf("TurnDetection provider '%s' not found", name),
			err,
		)
	}
	return provider, nil
}

// ListAllProviders returns a map of all registered providers across all sub-packages.
func (f *FacadeRegistry) ListAllProviders() map[string][]string {
	return map[string][]string{
		"stt":           f.STT().ListProviders(),
		"tts":           f.TTS().ListProviders(),
		"vad":           f.VAD().ListProviders(),
		"noise":         f.Noise().ListProviders(),
		"transport":     f.Transport().ListProviders(),
		"turndetection": f.TurnDetection().ListProviders(),
	}
}

// IsSTTProviderRegistered checks if an STT provider is registered.
func (f *FacadeRegistry) IsSTTProviderRegistered(name string) bool {
	return f.STT().IsRegistered(name)
}

// IsTTSProviderRegistered checks if a TTS provider is registered.
func (f *FacadeRegistry) IsTTSProviderRegistered(name string) bool {
	return f.TTS().IsRegistered(name)
}

// IsVADProviderRegistered checks if a VAD provider is registered.
func (f *FacadeRegistry) IsVADProviderRegistered(name string) bool {
	return f.VAD().IsRegistered(name)
}

// IsNoiseProviderRegistered checks if a Noise provider is registered.
func (f *FacadeRegistry) IsNoiseProviderRegistered(name string) bool {
	return f.Noise().IsRegistered(name)
}

// IsTransportProviderRegistered checks if a Transport provider is registered.
func (f *FacadeRegistry) IsTransportProviderRegistered(name string) bool {
	return f.Transport().IsRegistered(name)
}

// IsTurnDetectionProviderRegistered checks if a TurnDetection provider is registered.
func (f *FacadeRegistry) IsTurnDetectionProviderRegistered(name string) bool {
	return f.TurnDetection().IsRegistered(name)
}
