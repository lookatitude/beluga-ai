// Package phi provides Phi (Microsoft) provider implementation for multimodal models.
package phi

// ModalityCapabilities represents the capabilities of a provider or model for different modalities.
// This is a local copy to avoid importing multimodal package (which would create import cycles).
type ModalityCapabilities struct {
	// Text processing support
	Text bool

	// Image processing support
	Image bool

	// Audio processing support
	Audio bool

	// Video processing support
	Video bool

	// Maximum image size in bytes
	MaxImageSize int64

	// Maximum audio size in bytes
	MaxAudioSize int64

	// Maximum video size in bytes
	MaxVideoSize int64

	// Supported image formats
	SupportedImageFormats []string

	// Supported audio formats
	SupportedAudioFormats []string

	// Supported video formats
	SupportedVideoFormats []string
}
