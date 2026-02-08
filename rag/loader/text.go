package loader

import (
	"context"
	"os"
	"path/filepath"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	Register("text", func(cfg config.ProviderConfig) (DocumentLoader, error) {
		return NewTextLoader(), nil
	})
}

// TextLoader reads plain text files and creates one Document per file.
// The document content is the full file text, with the source path stored
// in metadata.
type TextLoader struct{}

// NewTextLoader creates a new TextLoader.
func NewTextLoader() *TextLoader {
	return &TextLoader{}
}

// Load reads the text file at the given path and returns a single-element
// Document slice with the file content.
func (l *TextLoader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	doc := schema.Document{
		ID:      source,
		Content: string(data),
		Metadata: map[string]any{
			"source": source,
			"format": "text",
			"name":   filepath.Base(source),
		},
	}
	return []schema.Document{doc}, nil
}
