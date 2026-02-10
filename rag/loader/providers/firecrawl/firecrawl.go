package firecrawl

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
	fc "github.com/mendableai/firecrawl-go"
)

func init() {
	loader.Register("firecrawl", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
		return New(cfg)
	})
}

// Loader uses Firecrawl to crawl web pages and return markdown documents.
type Loader struct {
	client *fc.FirecrawlApp
}

// Compile-time interface check.
var _ loader.DocumentLoader = (*Loader)(nil)

// New creates a new Firecrawl document loader.
func New(cfg config.ProviderConfig) (*Loader, error) {
	apiURL := cfg.BaseURL
	if apiURL == "" {
		apiURL = "https://api.firecrawl.dev"
	}
	app, err := fc.NewFirecrawlApp(cfg.APIKey, apiURL)
	if err != nil {
		return nil, fmt.Errorf("firecrawl: init client: %w", err)
	}
	return &Loader{client: app}, nil
}

// Load crawls the given URL and returns the content as markdown documents.
func (l *Loader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	if source == "" {
		return nil, fmt.Errorf("firecrawl: source URL is required")
	}

	result, err := l.client.ScrapeURL(source, nil)
	if err != nil {
		return nil, fmt.Errorf("firecrawl: scrape %q: %w", source, err)
	}

	content := result.Markdown
	if content == "" {
		return nil, nil
	}

	meta := map[string]any{
		"source": source,
		"format": "markdown",
		"loader": "firecrawl",
	}
	if result.Metadata.Title != nil && *result.Metadata.Title != "" {
		meta["title"] = *result.Metadata.Title
	}
	if result.Metadata.Description != nil && *result.Metadata.Description != "" {
		meta["description"] = *result.Metadata.Description
	}
	if result.Metadata.SourceURL != nil && *result.Metadata.SourceURL != "" {
		meta["source_url"] = *result.Metadata.SourceURL
	}

	return []schema.Document{{
		ID:       source,
		Content:  content,
		Metadata: meta,
	}}, nil
}
