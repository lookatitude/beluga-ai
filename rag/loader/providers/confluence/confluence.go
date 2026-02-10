package confluence

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/internal/httpclient"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	loader.Register("confluence", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// Loader loads pages from Confluence REST API.
type Loader struct {
	client *httpclient.Client
}

// Compile-time interface check.
var _ loader.DocumentLoader = (*Loader)(nil)

// New creates a new Confluence document loader.
func New(cfg config.ProviderConfig) (*Loader, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		return nil, fmt.Errorf("confluence: base URL is required (e.g., https://your-domain.atlassian.net/wiki)")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("confluence: API key/token is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	user, _ := config.GetOption[string](cfg, "user")

	clientOpts := []httpclient.Option{
		httpclient.WithBaseURL(baseURL),
		httpclient.WithTimeout(timeout),
	}

	if user != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(user + ":" + cfg.APIKey))
		clientOpts = append(clientOpts, httpclient.WithHeader("Authorization", "Basic "+auth))
	} else {
		clientOpts = append(clientOpts, httpclient.WithBearerToken(cfg.APIKey))
	}

	return &Loader{
		client: httpclient.New(clientOpts...),
	}, nil
}

// pageResponse is the Confluence content API response for a single page.
type pageResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Body  struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	Space struct {
		Key string `json:"key"`
	} `json:"space"`
}

// Load fetches a Confluence page by its ID. The source should be the page ID
// (numeric string) or "SPACE_KEY/page-id".
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("confluence: source (page ID) is required")
	}

	pageID := source
	if parts := strings.SplitN(source, "/", 2); len(parts) == 2 {
		pageID = parts[1]
	}

	path := fmt.Sprintf("/rest/api/content/%s?expand=body.storage,space", pageID)
	resp, err := httpclient.DoJSON[pageResponse](ctx, l.client, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("confluence: fetch page %q: %w", pageID, err)
	}

	content := resp.Body.Storage.Value
	if content == "" {
		return nil, nil
	}

	content = stripHTML(content)

	meta := map[string]any{
		"source":  source,
		"loader":  "confluence",
		"page_id": resp.ID,
		"title":   resp.Title,
		"space":   resp.Space.Key,
	}

	return []schema.Document{{
		ID:       resp.ID,
		Content:  content,
		Metadata: meta,
	}}, nil
}

// stripHTML removes HTML tags from content for plain text extraction.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
