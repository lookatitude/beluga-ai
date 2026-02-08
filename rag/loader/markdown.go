package loader

import (
	"context"
	"os"
	"path/filepath"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	Register("markdown", func(cfg config.ProviderConfig) (DocumentLoader, error) {
		return NewMarkdownLoader(), nil
	})
}

// MarkdownLoader reads Markdown files and creates one Document per file.
// The full Markdown source is preserved as content, with format metadata.
type MarkdownLoader struct{}

// NewMarkdownLoader creates a new MarkdownLoader.
func NewMarkdownLoader() *MarkdownLoader {
	return &MarkdownLoader{}
}

// Load reads the Markdown file at the given path and returns a single-element
// Document slice.
func (l *MarkdownLoader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	doc := schema.Document{
		ID:      source,
		Content: string(data),
		Metadata: map[string]any{
			"source": source,
			"format": "markdown",
			"name":   filepath.Base(source),
		},
	}
	return []schema.Document{doc}, nil
}
