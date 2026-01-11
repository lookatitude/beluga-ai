// Package internal provides internal implementation details for the multimodal package.
package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

// Normalizer handles format conversion between base64, URLs, and file paths.
type Normalizer struct{}

// NewNormalizer creates a new normalizer.
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// Normalize converts a content block to the target format.
func (n *Normalizer) Normalize(ctx context.Context, block *types.ContentBlock, targetFormat string) (*types.ContentBlock, error) {
	if block == nil {
		return nil, fmt.Errorf("Normalize: content block cannot be nil")
	}

	validFormats := []string{"base64", "url", "file_path"}
	isValid := false
	for _, f := range validFormats {
		if targetFormat == f {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, fmt.Errorf("Normalize: invalid target format: %s", targetFormat)
	}

	// If already in target format, return as-is
	if n.getCurrentFormat(block) == targetFormat {
		return block, nil
	}

	// Convert to target format
	switch targetFormat {
	case "base64":
		return n.toBase64(ctx, block)
	case "url":
		return n.toURL(ctx, block)
	case "file_path":
		return n.toFilePath(ctx, block)
	default:
		return nil, fmt.Errorf("Normalize: unsupported target format: %s", targetFormat)
	}
}

// getCurrentFormat determines the current format of a content block.
func (n *Normalizer) getCurrentFormat(block *types.ContentBlock) string {
	if len(block.Data) > 0 && block.URL == "" && block.FilePath == "" {
		return "base64"
	}
	if block.URL != "" {
		return "url"
	}
	if block.FilePath != "" {
		return "file_path"
	}
	return "base64" // default
}

// toBase64 converts a content block to base64 format.
func (n *Normalizer) toBase64(ctx context.Context, block *types.ContentBlock) (*types.ContentBlock, error) {
	// If already has data, use it
	if len(block.Data) > 0 {
		return block, nil
	}

	// Fetch from URL
	if block.URL != "" {
		// Use context-aware HTTP request
		req, err := http.NewRequestWithContext(ctx, "GET", block.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("toBase64: failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("toBase64: failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("toBase64: failed to fetch URL: status %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("toBase64: failed to read response: %w", err)
		}

		// Preserve MIME type from response if not already set
		mimeType := block.MIMEType
		if mimeType == "" {
			mimeType = resp.Header.Get("Content-Type")
		}

		return &types.ContentBlock{
			Type:     block.Type,
			Data:     data,
			Size:     int64(len(data)),
			MIMEType: mimeType,
			Format:   block.Format,
			Metadata: block.Metadata,
		}, nil
	}

	// Read from file
	if block.FilePath != "" {
		data, err := os.ReadFile(block.FilePath)
		if err != nil {
			return nil, fmt.Errorf("toBase64: %w", err)
		}

		return &types.ContentBlock{
			Type:     block.Type,
			Data:     data,
			Size:     int64(len(data)),
			MIMEType: block.MIMEType,
			Format:   block.Format,
			Metadata: block.Metadata,
		}, nil
	}

	return nil, fmt.Errorf("toBase64: content block has no data source")
}

// toURL converts a content block to URL format (not implemented - would require upload service).
func (n *Normalizer) toURL(ctx context.Context, block *types.ContentBlock) (*types.ContentBlock, error) {
	// URL conversion requires an upload service, which is not implemented
	// For now, return error if not already a URL
	if block.URL != "" {
		return block, nil
	}

	return nil, fmt.Errorf("toURL: URL conversion requires an upload service (not implemented)")
}

// toFilePath converts a content block to file path format.
func (n *Normalizer) toFilePath(ctx context.Context, block *types.ContentBlock) (*types.ContentBlock, error) {
	// If already a file path, return as-is
	if block.FilePath != "" {
		return block, nil
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var data []byte
	var err error

	// Get data from URL if needed
	if block.URL != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", block.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("toFilePath: failed to create request: %w", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("toFilePath: failed to fetch URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("toFilePath: failed to fetch URL: status %d", resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("toFilePath: failed to read response: %w", err)
		}
	} else if len(block.Data) > 0 {
		data = block.Data
	} else {
		return nil, fmt.Errorf("toFilePath: content block has no data source")
	}

	// Determine file extension from MIME type or format
	ext := block.Format
	if ext == "" && block.MIMEType != "" {
		// Try to extract extension from MIME type
		parts := strings.Split(block.MIMEType, "/")
		if len(parts) == 2 {
			ext = parts[1]
			// Normalize common extensions
			switch ext {
			case "jpeg":
				ext = "jpg"
			case "svg+xml":
				ext = "svg"
			}
		}
	}
	if ext == "" {
		// Default extension based on content type
		switch block.Type {
		case "image":
			ext = "png"
		case "audio":
			ext = "mp3"
		case "video":
			ext = "mp4"
		default:
			ext = "bin"
		}
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("multimodal_*.%s", ext))
	if err != nil {
		return nil, fmt.Errorf("toFilePath: failed to create temp file: %w", err)
	}

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("toFilePath: failed to write data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("toFilePath: failed to close temp file: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(tmpFile.Name())
	if err != nil {
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("toFilePath: failed to get absolute path: %w", err)
	}

	// Create new content block with file path
	result := &types.ContentBlock{
		Type:     block.Type,
		FilePath: absPath,
		Data:     data, // Keep data for compatibility
		Size:     int64(len(data)),
		MIMEType: block.MIMEType,
		Format:   ext,
		Metadata: make(map[string]any),
	}

	// Copy metadata and add temp file indicator
	if block.Metadata != nil {
		for k, v := range block.Metadata {
			result.Metadata[k] = v
		}
	}
	result.Metadata["temp_file"] = true
	result.Metadata["auto_created"] = true

	return result, nil
}
