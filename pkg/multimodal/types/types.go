// Package types defines core types for multimodal operations.
// This package is separate from the main multimodal package to avoid import cycles.
package types

import (
	"context"
	"errors"
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
	CreatedAt     time.Time
	Metadata      map[string]any
	Routing       map[string]any
	ID            string
	Format        string
	ContentBlocks []*ContentBlock
}

// MultimodalOutput represents the result of processing a multimodal input.
type MultimodalOutput struct {
	CreatedAt     time.Time
	Metadata      map[string]any
	ID            string
	InputID       string
	Provider      string
	Model         string
	ContentBlocks []*ContentBlock
	Confidence    float32
}

// ContentBlock represents a single piece of content (text, image, audio, video).
type ContentBlock struct {
	Metadata map[string]any
	Type     string
	URL      string
	FilePath string
	Format   string
	MIMEType string
	Data     []byte
	Size     int64
}

// ModalityCapabilities represents the capabilities of a provider or model for different modalities.
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

// NewContentBlock creates a new content block from raw data.
func NewContentBlock(contentType string, data []byte) (*ContentBlock, error) {
	if contentType == "" {
		return nil, errors.New("content type cannot be empty")
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
		return nil, errors.New("content type cannot be empty")
	}
	if url == "" {
		return nil, errors.New("URL cannot be empty")
	}

	// Fetch the content from URL using a client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := client.Do(req)
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
		return nil, errors.New("content type cannot be empty")
	}
	if filePath == "" {
		return nil, errors.New("file path cannot be empty")
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
		return nil, errors.New("must have at least one content block")
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
		return errors.New("content block must have at least one of: Data, URL, or FilePath")
	}

	// Validate size
	if cb.Size < 0 {
		return errors.New("size must be >= 0")
	}

	return nil
}
