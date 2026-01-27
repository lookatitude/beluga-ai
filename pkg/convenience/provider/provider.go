// Package provider provides unified provider creation utilities.
// It offers a simplified interface for creating providers of various types
// with automatic discovery and fallback support.
//
// Note: This package provides simplified wrappers around the existing
// package-specific providers (llms, embeddings, stt, tts, etc.).
//
// Example usage:
//
//	// List available providers
//	llmProviders := provider.ListLLMs()
//
// For creating actual provider instances, use the respective packages:
//   - llms.NewProvider() for LLM providers
//   - embeddings.NewProvider() for embedding providers
//   - stt.NewProvider() for speech-to-text providers
//   - tts.NewProvider() for text-to-speech providers
package provider

import (
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/stt"
	"github.com/lookatitude/beluga-ai/pkg/tts"
)

// ListLLMs returns all available LLM provider names.
func ListLLMs() []string {
	return llms.GetRegistry().ListProviders()
}

// ListSTTs returns all available STT provider names.
func ListSTTs() []string {
	return stt.GetRegistry().ListProviders()
}

// ListTTSs returns all available TTS provider names.
func ListTTSs() []string {
	return tts.GetRegistry().ListProviders()
}

// ProviderInfo contains information about an available provider.
type ProviderInfo struct {
	Name        string
	Type        string
	Description string
}

// GetAllProviders returns information about all available providers across all types.
func GetAllProviders() map[string][]ProviderInfo {
	result := make(map[string][]ProviderInfo)

	for _, name := range ListLLMs() {
		result["llm"] = append(result["llm"], ProviderInfo{
			Name: name,
			Type: "llm",
		})
	}

	for _, name := range ListSTTs() {
		result["stt"] = append(result["stt"], ProviderInfo{
			Name: name,
			Type: "stt",
		})
	}

	for _, name := range ListTTSs() {
		result["tts"] = append(result["tts"], ProviderInfo{
			Name: name,
			Type: "tts",
		})
	}

	return result
}
