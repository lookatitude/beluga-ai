package loader

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	Register("csv", func(cfg config.ProviderConfig) (DocumentLoader, error) {
		contentCols, _ := config.GetOption[string](cfg, "content_columns")
		return NewCSVLoader(
			WithContentColumns(contentCols),
		), nil
	})
}

// CSVLoaderOption configures a CSVLoader.
type CSVLoaderOption func(*CSVLoader)

// WithContentColumns sets a comma-separated list of column names to
// concatenate as document content. If empty, all columns are used.
func WithContentColumns(cols string) CSVLoaderOption {
	return func(l *CSVLoader) {
		if cols != "" {
			l.contentColumns = strings.Split(cols, ",")
		}
	}
}

// CSVLoader reads CSV files and creates one Document per row. Column headers
// are used as metadata keys.
type CSVLoader struct {
	contentColumns []string
}

// NewCSVLoader creates a new CSVLoader with the given options.
func NewCSVLoader(opts ...CSVLoaderOption) *CSVLoader {
	l := &CSVLoader{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Load reads a CSV file and returns one Document per row. The first row is
// treated as headers. Each row's values are stored in metadata, and the
// content is either all columns or only the configured content columns.
func (l *CSVLoader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("loader: csv parse error: %w", err)
	}

	if len(records) < 2 {
		return nil, nil // No data rows.
	}

	headers := records[0]
	baseName := filepath.Base(source)

	// Build column index for content extraction.
	contentIdxs := l.resolveContentColumns(headers)

	docs := make([]schema.Document, 0, len(records)-1)
	for i, row := range records[1:] {
		meta := map[string]any{
			"source": source,
			"format": "csv",
			"name":   baseName,
			"row":    i,
		}
		for j, header := range headers {
			if j < len(row) {
				meta[header] = row[j]
			}
		}

		content := l.buildContent(headers, row, contentIdxs)

		doc := schema.Document{
			ID:       fmt.Sprintf("%s#%d", source, i),
			Content:  content,
			Metadata: meta,
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

// resolveContentColumns returns indices of columns to use for content.
func (l *CSVLoader) resolveContentColumns(headers []string) []int {
	if len(l.contentColumns) == 0 {
		idxs := make([]int, len(headers))
		for i := range headers {
			idxs[i] = i
		}
		return idxs
	}

	headerMap := make(map[string]int, len(headers))
	for i, h := range headers {
		headerMap[h] = i
	}

	var idxs []int
	for _, col := range l.contentColumns {
		if idx, ok := headerMap[col]; ok {
			idxs = append(idxs, idx)
		}
	}
	return idxs
}

// buildContent concatenates the selected columns into document content.
func (l *CSVLoader) buildContent(headers []string, row []string, idxs []int) string {
	var parts []string
	for _, idx := range idxs {
		if idx < len(row) {
			parts = append(parts, headers[idx]+": "+row[idx])
		}
	}
	return strings.Join(parts, "\n")
}
