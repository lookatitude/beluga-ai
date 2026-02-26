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

// Compile-time interface check.
var _ loader.DocumentLoader = (*Loader)(nil)

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

	elements, err := l.uploadAndParse(ctx, source)
	if err != nil {
		return nil, err
	}

	return elementsToDocuments(elements, source), nil
}

// uploadAndParse uploads a file to the Unstructured API and decodes the response.
func (l *Loader) uploadAndParse(ctx context.Context, source string) ([]element, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("unstructured: open file: %w", err)
	}
	defer f.Close()

	body, contentType, err := buildMultipartBody(f, filepath.Base(source))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		l.baseURL+"/general/v0/general", body)
	if err != nil {
		return nil, fmt.Errorf("unstructured: create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
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
	return elements, nil
}

// buildMultipartBody creates a multipart form body from a file reader.
func buildMultipartBody(r io.Reader, filename string) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		return nil, "", fmt.Errorf("unstructured: create form: %w", err)
	}
	if _, err := io.Copy(part, r); err != nil {
		return nil, "", fmt.Errorf("unstructured: copy file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("unstructured: close writer: %w", err)
	}
	return &buf, writer.FormDataContentType(), nil
}

// elementsToDocuments combines extracted elements into a single document.
func elementsToDocuments(elements []element, source string) []schema.Document {
	if len(elements) == 0 {
		return nil
	}

	var content strings.Builder
	for _, el := range elements {
		if el.Text == "" {
			continue
		}
		if content.Len() > 0 {
			content.WriteString("\n\n")
		}
		content.WriteString(el.Text)
	}

	if content.Len() == 0 {
		return nil
	}

	return []schema.Document{{
		ID:      source,
		Content: content.String(),
		Metadata: map[string]any{
			"source":   source,
			"format":   "unstructured",
			"loader":   "unstructured",
			"filename": filepath.Base(source),
			"elements": len(elements),
		},
	}}
}
