// Package unstructured provides a DocumentLoader that uses the Unstructured.io
// API to extract structured content from files (PDFs, DOCX, images, etc.).
//
// The loader uploads files to the Unstructured.io partition API and returns
// the extracted elements as documents.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/unstructured"
//
//	l, err := loader.New("unstructured", config.ProviderConfig{
//	    APIKey:  "key-...",
//	    BaseURL: "https://api.unstructured.io",
//	})
//	docs, err := l.Load(ctx, "/path/to/document.pdf")
package unstructured

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
)

const defaultBaseURL = "https://api.unstructured.io"

func init() {
	loader.Register("unstructured", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// element represents a single extracted element from Unstructured API.
type element struct {
	Type     string `json:"type"`
	ElementID string `json:"element_id"`
	Text     string `json:"text"`
	Metadata struct {
		Filename    string `json:"filename"`
		PageNumber  int    `json:"page_number"`
		Languages   []string `json:"languages"`
	} `json:"metadata"`
}

// Loader uploads files to Unstructured.io API and extracts content.
type Loader struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// New creates a new Unstructured document loader.
func New(cfg config.ProviderConfig) (*Loader, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	client := &http.Client{}
	if cfg.Timeout > 0 {
		client.Timeout = cfg.Timeout
	}

	return &Loader{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		client:  client,
	}, nil
}

// Load reads a file from disk and uploads it to the Unstructured API.
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("unstructured: source file path is required")
	}

	f, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("unstructured: open file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("files", filepath.Base(source))
	if err != nil {
		return nil, fmt.Errorf("unstructured: create form: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("unstructured: copy file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("unstructured: close writer: %w", err)
	}

	url := l.baseURL + "/general/v0/general"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("unstructured: create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if l.apiKey != "" {
		req.Header.Set("unstructured-api-key", l.apiKey)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unstructured: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unstructured: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var elements []element
	if err := json.NewDecoder(resp.Body).Decode(&elements); err != nil {
		return nil, fmt.Errorf("unstructured: decode response: %w", err)
	}

	if len(elements) == 0 {
		return nil, nil
	}

	// Combine all text elements into a single document.
	var content strings.Builder
	for i, el := range elements {
		if el.Text == "" {
			continue
		}
		if i > 0 && content.Len() > 0 {
			content.WriteString("\n\n")
		}
		content.WriteString(el.Text)
	}

	if content.Len() == 0 {
		return nil, nil
	}

	meta := map[string]any{
		"source":   source,
		"format":   "unstructured",
		"loader":   "unstructured",
		"filename": filepath.Base(source),
		"elements": len(elements),
	}

	return []schema.Document{{
		ID:       source,
		Content:  content.String(),
		Metadata: meta,
	}}, nil
}
