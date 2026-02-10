// Package unstructured provides a DocumentLoader that uses the Unstructured.io
// API to extract structured content from files (PDFs, DOCX, images, etc.).
//
// The loader uploads files to the Unstructured.io partition API and returns
// the extracted elements as documents.
//
// # Registration
//
// The provider registers as "unstructured" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/unstructured"
//
//	l, err := loader.New("unstructured", config.ProviderConfig{
//	    APIKey:  "key-...",
//	    BaseURL: "https://api.unstructured.io",
//	})
//	docs, err := l.Load(ctx, "/path/to/document.pdf")
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Unstructured.io API key (required)
//   - BaseURL — API base URL (default: "https://api.unstructured.io")
package unstructured
