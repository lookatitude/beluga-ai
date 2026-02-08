package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	Register("json", func(cfg config.ProviderConfig) (DocumentLoader, error) {
		contentKey, _ := config.GetOption[string](cfg, "content_key")
		jqPath, _ := config.GetOption[string](cfg, "jq_path")
		return NewJSONLoader(
			WithContentKey(contentKey),
			WithJQPath(jqPath),
		), nil
	})
}

// JSONLoaderOption configures a JSONLoader.
type JSONLoaderOption func(*JSONLoader)

// WithContentKey sets the JSON key used to extract document content.
// If empty, the entire JSON object is serialized as content.
func WithContentKey(key string) JSONLoaderOption {
	return func(l *JSONLoader) {
		if key != "" {
			l.contentKey = key
		}
	}
}

// WithJQPath sets a simple dot-separated path for extracting an array of
// items from the JSON (e.g., "data.items"). Each item becomes a document.
func WithJQPath(path string) JSONLoaderOption {
	return func(l *JSONLoader) {
		if path != "" {
			l.jqPath = path
		}
	}
}

// JSONLoader reads JSON files and creates documents. It supports extracting
// items from a nested path and using a specific key for document content.
type JSONLoader struct {
	contentKey string
	jqPath     string
}

// NewJSONLoader creates a new JSONLoader with the given options.
func NewJSONLoader(opts ...JSONLoaderOption) *JSONLoader {
	l := &JSONLoader{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Load reads a JSON file and returns documents. If the top-level value is an
// array, each element becomes a document. If it is an object and jqPath is
// set, the path is traversed to find the array. Otherwise a single document
// is returned.
func (l *JSONLoader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, err
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("loader: json parse error: %w", err)
	}

	// Navigate jqPath if set.
	if l.jqPath != "" {
		raw, err = navigatePath(raw, l.jqPath)
		if err != nil {
			return nil, fmt.Errorf("loader: json path %q: %w", l.jqPath, err)
		}
	}

	// Convert to array of items.
	var items []any
	switch v := raw.(type) {
	case []any:
		items = v
	default:
		items = []any{v}
	}

	baseName := filepath.Base(source)
	docs := make([]schema.Document, 0, len(items))
	for i, item := range items {
		content, err := l.extractContent(item)
		if err != nil {
			return nil, fmt.Errorf("loader: json item %d: %w", i, err)
		}
		doc := schema.Document{
			ID:      fmt.Sprintf("%s#%d", source, i),
			Content: content,
			Metadata: map[string]any{
				"source": source,
				"format": "json",
				"name":   baseName,
				"index":  i,
			},
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// extractContent gets the text content from a JSON item.
func (l *JSONLoader) extractContent(item any) (string, error) {
	if l.contentKey != "" {
		obj, ok := item.(map[string]any)
		if !ok {
			return "", fmt.Errorf("expected object for content_key extraction, got %T", item)
		}
		val, ok := obj[l.contentKey]
		if !ok {
			return "", fmt.Errorf("key %q not found in object", l.contentKey)
		}
		if s, ok := val.(string); ok {
			return s, nil
		}
		b, err := json.Marshal(val)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	// Serialize entire item.
	b, err := json.Marshal(item)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// navigatePath traverses a dot-separated path through nested JSON objects.
func navigatePath(data any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected object at %q, got %T", part, current)
		}
		val, ok := obj[part]
		if !ok {
			return nil, fmt.Errorf("key %q not found", part)
		}
		current = val
	}
	return current, nil
}
