package gdrive

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
)

const defaultBaseURL = "https://www.googleapis.com"

func init() {
	loader.Register("gdrive", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// Loader loads files from Google Drive via the REST API.
type Loader struct {
	client *httpclient.Client
}

// Compile-time interface check.
var _ loader.DocumentLoader = (*Loader)(nil)

// New creates a new Google Drive document loader.
func New(cfg config.ProviderConfig) (*Loader, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gdrive: API key or OAuth token is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	client := httpclient.New(
		httpclient.WithBaseURL(baseURL),
		httpclient.WithBearerToken(cfg.APIKey),
		httpclient.WithTimeout(timeout),
	)

	return &Loader{client: client}, nil
}

// fileMetadata is the Google Drive file metadata response.
type fileMetadata struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	Size     string `json:"size"`
}

// Load fetches a file from Google Drive by its file ID. For Google Docs/Sheets,
// it exports as plain text. For other files, it downloads the content directly.
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("gdrive: file ID is required")
	}

	// Get file metadata.
	path := fmt.Sprintf("/drive/v3/files/%s?fields=id,name,mimeType,size", source)
	meta, err := httpclient.DoJSON[fileMetadata](ctx, l.client, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("gdrive: get metadata %q: %w", source, err)
	}

	// Download or export content.
	var content string
	if isGoogleDoc(meta.MimeType) {
		content, err = l.exportFile(ctx, source, meta.MimeType)
	} else {
		content, err = l.downloadFile(ctx, source)
	}
	if err != nil {
		return nil, fmt.Errorf("gdrive: get content %q: %w", source, err)
	}

	if content == "" {
		return nil, nil
	}

	docMeta := map[string]any{
		"source":    source,
		"loader":    "gdrive",
		"file_id":   meta.ID,
		"file_name": meta.Name,
		"mime_type": meta.MimeType,
	}

	return []schema.Document{{
		ID:       meta.ID,
		Content:  content,
		Metadata: docMeta,
	}}, nil
}

// isGoogleDoc returns true if the MIME type is a Google Workspace format.
func isGoogleDoc(mimeType string) bool {
	return strings.HasPrefix(mimeType, "application/vnd.google-apps.")
}

// exportFile exports a Google Workspace document as plain text.
func (l *Loader) exportFile(ctx context.Context, fileID, mimeType string) (string, error) {
	exportMime := "text/plain"
	if strings.Contains(mimeType, "spreadsheet") {
		exportMime = "text/csv"
	}

	path := fmt.Sprintf("/drive/v3/files/%s/export?mimeType=%s", fileID, exportMime)
	resp, err := l.client.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("export failed (status %d): %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read export: %w", err)
	}
	return string(data), nil
}

// downloadFile downloads a non-Google-Workspace file's content.
func (l *Loader) downloadFile(ctx context.Context, fileID string) (string, error) {
	path := fmt.Sprintf("/drive/v3/files/%s?alt=media", fileID)
	resp, err := l.client.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed (status %d): %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read download: %w", err)
	}
	return string(data), nil
}
