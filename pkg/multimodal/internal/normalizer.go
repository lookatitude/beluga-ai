// Package internal provides internal implementation details for the multimodal package.
package internal

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

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
		resp, err := http.Get(block.URL)
		if err != nil {
			return nil, fmt.Errorf("toBase64: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("toBase64: failed to fetch URL: status %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
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

	// For base64 or URL, we would need to save to a temporary file
	// This is a simplified implementation - in production, you'd want proper temp file handling
	if len(block.Data) > 0 {
		// In a real implementation, save to temp file
		// For now, return error indicating this requires file system access
		return nil, fmt.Errorf("toFilePath: file path conversion requires file system access (not fully implemented)")
	}

	return nil, fmt.Errorf("toFilePath: content block has no data source")
}
