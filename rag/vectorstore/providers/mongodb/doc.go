// Package mongodb provides a VectorStore backed by MongoDB Atlas Vector Search.
// It uses the official MongoDB Go driver (go.mongodb.org/mongo-driver/v2) and
// the $vectorSearch aggregation stage. The similarity function (cosine,
// dot product or Euclidean) is set by the Atlas vector index definition.
//
// Earlier releases spoke the Atlas Data API, which MongoDB retired on
// 30 September 2025. A Data API URL is now rejected with a
// core.ErrInvalidInput error; pass a MongoDB connection string instead.
// WithAPIKey and WithHTTPClient are deprecated no-ops.
//
// # Registration
//
// The provider registers as "mongodb" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/vectorstore/providers/mongodb"
//
//	store, err := vectorstore.New("mongodb", config.ProviderConfig{
//	    BaseURL: "mongodb+srv://user:pass@cluster0.example.mongodb.net",
//	    Options: map[string]any{
//	        "database":   "my_db",
//	        "collection": "documents",
//	        "index":      "vector_index",
//	    },
//	})
//
// To reuse an existing client, wrap a collection handle:
//
//	store := mongodb.NewWithCollection(client.Database("my_db").Collection("documents"))
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — MongoDB connection string, mongodb:// or mongodb+srv:// (required)
//   - Options["database"] — database name (default: "beluga")
//   - Options["collection"] — collection name (default: "documents")
//   - Options["index"] — Atlas vector search index name (default: "vector_index")
//
// Documents are stored as {_id, content, embedding, metadata}. The vector
// index must index "embedding" and declare every "metadata.<key>" used in
// search filters as a filter field. Filter keys must be plain identifiers and
// values scalars. Stores created by New or NewFromConfig own their client;
// call Close to disconnect it.
package mongodb
