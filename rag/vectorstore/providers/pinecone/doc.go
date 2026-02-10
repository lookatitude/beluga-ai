// Package pinecone provides a VectorStore backed by the Pinecone vector
// database. It communicates with Pinecone via its REST API.
//
// # Registration
//
// The provider registers as "pinecone" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pinecone"
//
//	store, err := vectorstore.New("pinecone", config.ProviderConfig{
//	    APIKey:  "your-api-key",
//	    BaseURL: "https://index-name-project.svc.environment.pinecone.io",
//	    Options: map[string]any{
//	        "namespace": "my_namespace",
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Pinecone API key (required)
//   - BaseURL — index endpoint URL (required)
//   - Options["namespace"] — namespace for document isolation
package pinecone
