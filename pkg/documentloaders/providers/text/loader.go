package text

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// LoaderConfig is duplicated here to avoid import cycle.
type LoaderConfig struct {
	MaxFileSize int64
}

// TextLoader loads a single text file.
type TextLoader struct {
	path   string
	config *LoaderConfig
	tracer trace.Tracer
}

// NewTextLoader creates a new TextLoader.
func NewTextLoader(path string, config *LoaderConfig) (*TextLoader, error) {
	// Basic validation
	if path == "" {
		return nil, newLoaderError("NewTextLoader", ErrCodeInvalidConfig, "", "path cannot be empty", nil)
	}
	if config.MaxFileSize < 1 {
		config.MaxFileSize = 100 * 1024 * 1024 // Default 100MB
	}

	return &TextLoader{
		path:   path,
		config: config,
		tracer: otel.Tracer("github.com/lookatitude/beluga-ai/pkg/documentloaders/text"),
	}, nil
}

// Load implements the DocumentLoader interface.
func (l *TextLoader) Load(ctx context.Context) ([]schema.Document, error) {
	ctx, span := l.tracer.Start(ctx, "documentloaders.text.Load",
		trace.WithAttributes(
			attribute.String("loader.type", "text"),
			attribute.String("loader.path", l.path),
		))
	defer span.End()

	start := time.Now()

	// Open file
	file, err := os.Open(l.path)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, newLoaderError("Load", ErrCodeNotFound, l.path, "file not found", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log but don't fail - file is already read
		}
	}()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, newLoaderError("Load", ErrCodeIOError, l.path, fmt.Sprintf("failed to stat file: %v", err), err)
	}

	// Check MaxFileSize
	if info.Size() > l.config.MaxFileSize {
		span.RecordError(fmt.Errorf("file too large"))
		span.SetStatus(codes.Error, "file too large")
		return nil, newLoaderError("Load", ErrCodeFileTooLarge, l.path, fmt.Sprintf("file size %d exceeds max %d", info.Size(), l.config.MaxFileSize), nil)
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, newLoaderError("Load", ErrCodeIOError, l.path, fmt.Sprintf("failed to read file: %v", err), err)
	}

	// Create document
	doc := schema.Document{
		PageContent: string(content),
		Metadata: map[string]string{
			"source":      l.path,
			"file_size":   fmt.Sprintf("%d", info.Size()),
			"modified_at": info.ModTime().Format(time.RFC3339),
			"loader_type": "text",
		},
	}

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int("loader.documents_count", 1),
		attribute.Int64("loader.duration_ms", duration.Milliseconds()),
	)
	span.SetStatus(codes.Ok, "")

	return []schema.Document{doc}, nil
}

// LazyLoad implements the DocumentLoader interface.
func (l *TextLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
	ctx, span := l.tracer.Start(ctx, "documentloaders.text.LazyLoad",
		trace.WithAttributes(
			attribute.String("loader.type", "text"),
			attribute.String("loader.path", l.path),
		))
	defer span.End()

	ch := make(chan any, 1)

	go func() {
		defer close(ch)

		doc, err := l.Load(ctx)
		if err != nil {
			ch <- err
			return
		}

		if len(doc) > 0 {
			ch <- doc[0]
		}
	}()

	return ch, nil
}

// Ensure TextLoader implements iface.DocumentLoader
var _ iface.DocumentLoader = (*TextLoader)(nil)
