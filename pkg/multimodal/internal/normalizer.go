// Package internal provides internal implementation details for the multimodal package.
package internal

import (
	"context"
	"errors"
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
		return nil, errors.New("Normalize: content block cannot be nil")
	}

	// Use map for O(1) lookup instead of O(n) slice iteration
	validFormats := map[string]bool{
		"base64":    true,
		"url":       true,
		"file_path": true,
	}
	if !validFormats[targetFormat] {
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
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, block.URL, nil)
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

		// Pre-allocate buffer if Content-Length header is available for better performance
		var data []byte
		if contentLength := resp.ContentLength; contentLength > 0 {
			data = make([]byte, 0, contentLength)
		}

		bodyData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("toBase64: failed to read response: %w", err)
		}
		data = append(data, bodyData...)

		// Preserve MIME type from response if not already set
		mimeType := block.MIMEType
		if mimeType == "" {
			mimeType = resp.Header.Get("Content-Type")
		}

		// Reuse metadata map if it exists to avoid allocation
		metadata := block.Metadata
		if metadata == nil {
			metadata = make(map[string]any)
		}

		return &types.ContentBlock{
			Type:     block.Type,
			Data:     data,
			Size:     int64(len(data)),
			MIMEType: mimeType,
			Format:   block.Format,
			Metadata: metadata,
		}, nil
	}

	// Read from file
	if block.FilePath != "" {
		data, err := os.ReadFile(block.FilePath)
		if err != nil {
			return nil, fmt.Errorf("toBase64: %w", err)
		}

		// Reuse metadata map if it exists to avoid allocation
		metadata := block.Metadata
		if metadata == nil {
			metadata = make(map[string]any)
		}

		return &types.ContentBlock{
			Type:     block.Type,
			Data:     data,
			Size:     int64(len(data)),
			MIMEType: block.MIMEType,
			Format:   block.Format,
			Metadata: metadata,
		}, nil
	}

	return nil, errors.New("toBase64: content block has no data source")
}

// toURL converts a content block to URL format (not implemented - would require upload service).
func (n *Normalizer) toURL(ctx context.Context, block *types.ContentBlock) (*types.ContentBlock, error) {
	// URL conversion requires an upload service, which is not implemented
	// For now, return error if not already a URL
	if block.URL != "" {
		return block, nil
	}

	return nil, errors.New("toURL: URL conversion requires an upload service (not implemented)")
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
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, block.URL, nil)
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

		// Pre-allocate buffer if Content-Length header is available
		var bodyData []byte
		if contentLength := resp.ContentLength; contentLength > 0 {
			bodyData = make([]byte, 0, contentLength)
		}

		readData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("toFilePath: failed to read response: %w", err)
		}
		if bodyData != nil {
			data = append(bodyData, readData...)
		} else {
			data = readData
		}
	} else if len(block.Data) > 0 {
		data = block.Data
	} else {
		return nil, errors.New("toFilePath: content block has no data source")
	}

	// Determine file extension from MIME type or format
	// Use map lookup for better performance
	ext := block.Format
	if ext == "" && block.MIMEType != "" {
		// Try to extract extension from MIME type
		parts := strings.SplitN(block.MIMEType, "/", 2)
		if len(parts) == 2 {
			ext = parts[1]
			// Normalize common extensions using map for O(1) lookup
			extMap := map[string]string{
				"jpeg":    "jpg",
				"svg+xml": "svg",
			}
			if normalized, ok := extMap[ext]; ok {
				ext = normalized
			}
		}
	}
	if ext == "" {
		// Default extension based on content type - use map for O(1) lookup
		typeExtMap := map[string]string{
			"image": "png",
			"audio": "mp3",
			"video": "mp4",
		}
		if defaultExt, ok := typeExtMap[block.Type]; ok {
			ext = defaultExt
		} else {
			ext = "bin"
		}
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "multimodal_*."+ext)
	if err != nil {
		return nil, fmt.Errorf("toFilePath: failed to create temp file: %w", err)
	}

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("toFilePath: failed to write data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("toFilePath: failed to close temp file: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(tmpFile.Name())
	if err != nil {
		_ = os.Remove(tmpFile.Name())
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
	// Pre-allocate metadata map with known size to avoid reallocations
	if block.Metadata != nil {
		result.Metadata = make(map[string]any, len(block.Metadata)+2)
		for k, v := range block.Metadata {
			result.Metadata[k] = v
		}
	} else {
		result.Metadata = make(map[string]any, 2)
	}
	result.Metadata["temp_file"] = true
	result.Metadata["auto_created"] = true

	return result, nil
}
