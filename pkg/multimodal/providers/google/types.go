// Package google provides Google Vertex AI provider implementation for multimodal models.
package google

// ModalityCapabilities represents the capabilities of a provider or model for different modalities.
// This is a local copy to avoid importing multimodal package (which would create import cycles).
type ModalityCapabilities struct {
	SupportedImageFormats []string
	SupportedAudioFormats []string
	SupportedVideoFormats []string
	MaxImageSize          int64
	MaxAudioSize          int64
	MaxVideoSize          int64
	Text                  bool
	Image                 bool
	Audio                 bool
	Video                 bool
}
