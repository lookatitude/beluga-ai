// Package iface defines interfaces for multimodal content operations.
package iface

// ContentBlock defines the interface for a single content block within multimodal input/output.
// A content block represents a single piece of content (text, image, audio, video).
type ContentBlock interface {
	// GetType returns the content type ("text", "image", "audio", "video").
	GetType() string

	// GetData returns the raw content data (base64 encoded or raw bytes).
	GetData() []byte

	// GetURL returns the URL to the content (if applicable).
	GetURL() string

	// GetFilePath returns the file path to the content (if applicable).
	GetFilePath() string

	// GetMIMEType returns the MIME type of the content (e.g., "image/png", "audio/mpeg").
	GetMIMEType() string

	// GetSize returns the size of the content in bytes.
	GetSize() int64

	// GetMetadata returns additional metadata for the content block.
	GetMetadata() map[string]any
}
