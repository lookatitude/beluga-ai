// Package docling provides a DocumentLoader that uses the IBM Docling API
// to convert documents (PDFs, DOCX, images, etc.) into structured content.
//
// Docling (https://github.com/DS4SD/docling) is IBM's document understanding
// service that extracts text, tables, and layout from documents.
//
// # Registration
//
// The provider registers as "docling" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/docling"
//
//	l, err := loader.New("docling", config.ProviderConfig{
//	    BaseURL: "http://localhost:5001",
//	})
//	docs, err := l.Load(ctx, "/path/to/document.pdf")
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — Docling API server URL (required)
package docling
