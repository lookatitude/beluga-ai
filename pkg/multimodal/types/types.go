// Package types defines core types for multimodal operations.
// This package is separate from the main multimodal package to avoid import cycles.
package types

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MultimodalInput represents a multimodal input containing one or more content types.
type MultimodalInput struct {
	// ID is a unique identifier for this input
	ID string

	// ContentBlocks contains the content blocks (text, images, audio, video)
	ContentBlocks []*ContentBlock

	// Metadata contains additional metadata
	Metadata map[string]any

	// Format is the preferred format ("base64", "url", "file_path")
	Format string

	// Routing contains routing instructions for content blocks
	Routing map[string]any // Using map to avoid importing multimodal.Config

	// CreatedAt is the timestamp when this input was created
	CreatedAt time.Time
}

// MultimodalOutput represents the result of processing a multimodal input.
type MultimodalOutput struct {
	// ID is a unique identifier for this output
	ID string

	// InputID is the ID of the input that generated this output
	InputID string

	// ContentBlocks contains the output content blocks
	ContentBlocks []*ContentBlock

	// Metadata contains additional metadata
	Metadata map[string]any

	// Confidence is the confidence score (0.0 to 1.0)
	Confidence float32

	// Provider is the name of the provider that generated this output
	Provider string

	// Model is the name of the model that generated this output
	Model string

	// CreatedAt is the timestamp when this output was created
	CreatedAt time.Time
}

// ContentBlock represents a single piece of content (text, image, audio, video).
type ContentBlock struct {
	// Type is the content type: "text", "image", "audio", "video"
	Type string

	// Data is the raw content data (base64-encoded for binary content)
	Data []byte

	// URL is the URL to the content (if using URL format)
	URL string

	// FilePath is the file path to the content (if using file path format)
	FilePath string

	// Format is the content format (e.g., "png", "mp3", "mp4")
	Format string

	// MIMEType is the MIME type of the content
	MIMEType string

	// Size is the size in bytes
	Size int64

	// Metadata contains additional metadata for this content block
	Metadata map[string]any
}

// ModalityCapabilities represents the capabilities of a provider or model for different modalities.
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

// NewContentBlock creates a new content block from raw data.
func NewContentBlock(contentType string, data []byte) (*ContentBlock, error) {
	if contentType == "" {
		return nil, fmt.Errorf("content type cannot be empty")
	}

	validTypes := []string{"text", "image", "audio", "video"}
	isValid := false
	for _, t := range validTypes {
		if contentType == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, fmt.Errorf("invalid content type: %s (must be one of: text, image, audio, video)", contentType)
	}

	block := &ContentBlock{
		Type:     contentType,
		Data:     data,
		Size:     int64(len(data)),
		Metadata: make(map[string]any),
	}

	// Detect MIME type if not provided
	if contentType == "text" {
		block.MIMEType = "text/plain"
	} else {
		// Try to detect MIME type from data
		mimeType := http.DetectContentType(data)
		block.MIMEType = mimeType
	}

	return block, nil
}

// NewContentBlockFromURL creates a new content block from a URL.
func NewContentBlockFromURL(ctx context.Context, contentType, url string) (*ContentBlock, error) {
	if contentType == "" {
		return nil, fmt.Errorf("content type cannot be empty")
	}
	if url == "" {
		return nil, fmt.Errorf("URL cannot be empty")
	}

	// Fetch the content from URL
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch URL: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	block := &ContentBlock{
		Type:     contentType,
		URL:      url,
		Data:     data,
		Size:     int64(len(data)),
		MIMEType: resp.Header.Get("Content-Type"),
		Metadata: make(map[string]any),
	}

	// Extract format from MIME type or URL
	if block.MIMEType != "" {
		exts, _ := mime.ExtensionsByType(block.MIMEType)
		if len(exts) > 0 {
			block.Format = strings.TrimPrefix(exts[0], ".")
		}
	}

	return block, nil
}

// NewContentBlockFromFile creates a new content block from a file path.
func NewContentBlockFromFile(ctx context.Context, contentType, filePath string) (*ContentBlock, error) {
	if contentType == "" {
		return nil, fmt.Errorf("content type cannot be empty")
	}
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}

	// Check if file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect MIME type from file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	block := &ContentBlock{
		Type:     contentType,
		FilePath: filePath,
		Data:     data,
		Size:     info.Size(),
		MIMEType: mimeType,
		Format:   strings.TrimPrefix(ext, "."),
		Metadata: make(map[string]any),
	}

	return block, nil
}

// NewMultimodalInput creates a new multimodal input with the given content blocks and options.
func NewMultimodalInput(contentBlocks []*ContentBlock, opts ...MultimodalInputOption) (*MultimodalInput, error) {
	if len(contentBlocks) == 0 {
		return nil, fmt.Errorf("must have at least one content block")
	}

	// Validate all content blocks
	for i, block := range contentBlocks {
		if err := block.Validate(); err != nil {
			return nil, fmt.Errorf("content block %d validation failed: %w", i, err)
		}
	}

	input := &MultimodalInput{
		ID:            uuid.New().String(),
		ContentBlocks: contentBlocks,
		Metadata:      make(map[string]any),
		Format:        "base64", // default
		CreatedAt:     time.Now(),
	}

	// Apply options
	for _, opt := range opts {
		opt(input)
	}

	return input, nil
}

// MultimodalInputOption defines a function type for MultimodalInput options.
type MultimodalInputOption func(*MultimodalInput)

// Validate validates the content block.
func (cb *ContentBlock) Validate() error {
	// Validate type
	validTypes := []string{"text", "image", "audio", "video"}
	isValid := false
	for _, t := range validTypes {
		if cb.Type == t {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid content type: %s (must be one of: text, image, audio, video)", cb.Type)
	}

	// Must have at least one data source
	if len(cb.Data) == 0 && cb.URL == "" && cb.FilePath == "" {
		return fmt.Errorf("content block must have at least one of: Data, URL, or FilePath")
	}

	// Validate size
	if cb.Size < 0 {
		return fmt.Errorf("size must be >= 0")
	}

	return nil
}
