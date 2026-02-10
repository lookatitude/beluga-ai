package docling

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

const defaultBaseURL = "http://localhost:5001"

func init() {
	loader.Register("docling", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// convertRequest is the request body for Docling's URL-based conversion.
type convertRequest struct {
	Source string `json:"source"`
}

// convertResponse is the response from Docling's convert endpoint.
type convertResponse struct {
	Document struct {
		Markdown string `json:"md_content"`
		Text     string `json:"text_content"`
	} `json:"document"`
	Status string `json:"status"`
}

// Loader converts documents using the IBM Docling API.
type Loader struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// Compile-time interface check.
var _ loader.DocumentLoader = (*Loader)(nil)

// New creates a new Docling document loader.
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

// Load converts a document from a file path or URL using the Docling API.
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("docling: source is required")
	}

	// If source looks like a URL, use JSON body. Otherwise, upload file.
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return l.loadFromURL(ctx, source)
	}
	return l.loadFromFile(ctx, source)
}

func (l *Loader) loadFromURL(ctx context.Context, source string) ([]schema.Document, error) {
	reqBody := convertRequest{Source: source}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("docling: marshal request: %w", err)
	}

	url := l.baseURL + "/v1/convert"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("docling: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if l.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.apiKey)
	}

	return l.doRequest(ctx, req, source)
}

func (l *Loader) loadFromFile(ctx context.Context, source string) ([]schema.Document, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("docling: open file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(source))
	if err != nil {
		return nil, fmt.Errorf("docling: create form: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("docling: copy file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("docling: close writer: %w", err)
	}

	url := l.baseURL + "/v1/convert"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, fmt.Errorf("docling: create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if l.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.apiKey)
	}

	return l.doRequest(ctx, req, source)
}

func (l *Loader) doRequest(_ context.Context, req *http.Request, source string) ([]schema.Document, error) {
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docling: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("docling: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result convertResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("docling: decode response: %w", err)
	}

	content := result.Document.Markdown
	if content == "" {
		content = result.Document.Text
	}
	if content == "" {
		return nil, nil
	}

	meta := map[string]any{
		"source": source,
		"format": "docling",
		"loader": "docling",
	}

	return []schema.Document{{
		ID:       source,
		Content:  content,
		Metadata: meta,
	}}, nil
}
