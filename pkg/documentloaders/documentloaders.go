package documentloaders

import (
	"context"
	"io/fs"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/documentloaders/iface"
	"github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/directory"
	"github.com/lookatitude/beluga-ai/pkg/documentloaders/providers/text"
)

// NewDirectoryLoader creates a loader that recursively reads files from a directory.
// The loader uses the provided file system abstraction (fs.FS) for testability and flexibility.
//
// Parameters:
//   - fsys: The file system to read from (e.g., os.DirFS("/path/to/documents"))
//   - opts: Optional configuration functions (e.g., WithMaxDepth, WithExtensions)
//
// Returns:
//   - DocumentLoader: A configured directory loader instance
//   - error: Returns LoaderError if configuration is invalid
//
// Example:
//
//	fsys := os.DirFS("/path/to/documents")
//	loader, err := NewDirectoryLoader(fsys,
//	    WithMaxDepth(5),
//	    WithExtensions(".txt", ".md"),
//	)
func NewDirectoryLoader(fsys fs.FS, opts ...DirectoryOption) (iface.DocumentLoader, error) {
	cfg := DefaultDirectoryConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Convert to directory package's config type
	dirCfg := &directory.DirectoryConfig{
		MaxDepth:       cfg.MaxDepth,
		Extensions:     cfg.Extensions,
		Concurrency:    cfg.Concurrency,
		MaxFileSize:    cfg.MaxFileSize,
		FollowSymlinks: cfg.FollowSymlinks,
	}

	return directory.NewRecursiveDirectoryLoader(fsys, dirCfg)
}

// NewTextLoader creates a loader for a single text file.
// This is useful when you need to load a specific file without directory traversal overhead.
//
// Parameters:
//   - path: Absolute or relative path to the text file
//   - opts: Optional configuration functions (e.g., WithMaxFileSize)
//
// Returns:
//   - DocumentLoader: A configured text loader instance
//   - error: Returns LoaderError if the file path is invalid or inaccessible
//
// Example:
//
//	loader, err := NewTextLoader("/path/to/file.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	docs, err := loader.Load(ctx)
func NewTextLoader(path string, opts ...Option) (iface.DocumentLoader, error) {
	cfg := DefaultLoaderConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Convert to text package's config type
	textCfg := &text.LoaderConfig{
		MaxFileSize: cfg.MaxFileSize,
	}

	return text.NewTextLoader(path, textCfg)
}

// logWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}
