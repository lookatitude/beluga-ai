// Package loaders provides implementations of the rag.Loader interface.
package loaders

import (
	"bufio"
	"context"
	"fmt"
	"os"

	rag "github.com/lookatitude/beluga-ai/pkg/retrievers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// FileLoader loads documents from a single text file.
// Each line in the file is treated as a separate document by default,
// but this can be customized.
type FileLoader struct {
	FilePath string
	// TODO: Add options for encoding, splitting strategy (e.g., whole file as one doc)
}

// NewFileLoader creates a new FileLoader.
func NewFileLoader(filePath string) *FileLoader {
	return &FileLoader{
		FilePath: filePath,
	}
}

// Load reads the entire file and returns documents (one per line).
func (l *FileLoader) Load(ctx context.Context) ([]schema.Document, error) {
	file, err := os.Open(l.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", l.FilePath, err)
	}
	defer file.Close()

	docs := make([]schema.Document, 0)
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		lineText := scanner.Text()
		metadata := map[string]any{
			"source": l.FilePath,
			"line":   lineNumber,
		}
		docs = append(docs, schema.NewDocument(lineText, metadata))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %w", l.FilePath, err)
	}

	return docs, nil
}

// LazyLoad reads the file line by line and sends documents over a channel.
func (l *FileLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
	file, err := os.Open(l.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", l.FilePath, err)
	}

	docChan := make(chan any)

	go func() {
		defer close(docChan)
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			lineNumber++
			lineText := scanner.Text()
			metadata := map[string]any{
				"source": l.FilePath,
				"line":   lineNumber,
			}
			doc := schema.NewDocument(lineText, metadata)

			select {
			case docChan <- doc:
			case <-ctx.Done():
				fmt.Printf("LazyLoad cancelled for file %s\n", l.FilePath)
				return
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case docChan <- fmt.Errorf("error scanning file %s: %w", l.FilePath, err):
			case <-ctx.Done():
				// Context cancelled, ignore final error
			default:
				fmt.Printf("Error sending scan error on channel: %v\n", err)
			}
		}
	}()

	return docChan, nil
}

// Ensure FileLoader implements the interface.
var _ rag.Loader = (*FileLoader)(nil)
