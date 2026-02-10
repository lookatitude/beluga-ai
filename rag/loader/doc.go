// Package loader provides document loading capabilities for the RAG pipeline.
// It defines the [DocumentLoader] interface for reading content from various
// sources (files, URLs, APIs) and converting them into [schema.Document] slices.
//
// # Interfaces
//
// The core interface is [DocumentLoader]:
//
//	type DocumentLoader interface {
//	    Load(ctx context.Context, source string) ([]schema.Document, error)
//	}
//
// The [Transformer] interface allows post-load enrichment:
//
//	type Transformer interface {
//	    Transform(ctx context.Context, doc schema.Document) (schema.Document, error)
//	}
//
// # Registry
//
// The package follows Beluga's registry pattern. Providers register via
// init() and are instantiated with [New]:
//
//	l, err := loader.New("text", config.ProviderConfig{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	docs, err := l.Load(ctx, "/path/to/file.txt")
//
// Use [List] to discover all registered loader names.
//
// # Built-in Loaders
//
//   - "text" — plain text files
//   - "json" — JSON files with configurable path extraction
//   - "csv" — CSV files (one document per row)
//   - "markdown" — Markdown files
//
// # External Loaders
//
// Available as provider imports:
//   - "cloudstorage" — S3, GCS, Azure Blob Storage
//   - "confluence" — Atlassian Confluence pages
//   - "docling" — IBM Docling document understanding (PDFs, DOCX, images)
//   - "firecrawl" — Firecrawl web scraping and crawling
//   - "gdrive" — Google Drive files
//   - "github" — GitHub repository files
//   - "notion" — Notion pages
//   - "unstructured" — Unstructured.io document extraction
//
// # Pipeline
//
// [LoaderPipeline] chains multiple loaders and transformers. Loaders are invoked
// in order and their results concatenated, then transformers are applied to each
// document:
//
//	p := loader.NewPipeline(
//	    loader.WithLoader(textLoader),
//	    loader.WithTransformer(loader.TransformerFunc(func(ctx context.Context, doc schema.Document) (schema.Document, error) {
//	        doc.Metadata["processed"] = true
//	        return doc, nil
//	    })),
//	)
//	docs, err := p.Load(ctx, "/path/to/files")
//
// # Custom Provider
//
// To add a custom document loader:
//
//	func init() {
//	    loader.Register("custom", func(cfg config.ProviderConfig) (loader.DocumentLoader, error) {
//	        return &myLoader{apiKey: cfg.APIKey}, nil
//	    })
//	}
package loader
