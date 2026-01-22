# Multimodal Embeddings with Google

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this guide, we'll use **Google's Gemini** models via `pkg/embeddings` to generate embeddings for both text and images. We will see how to leverage the `MultimodalEmbedder` interface to build a true semantic search engine.

## Learning Objectives
By the end of this tutorial, you will:
1.  Initialize the Google Multimodal provider.
2.  Embed text and images into a shared vector space.
3.  Calculate cosine similarity to find matches across modalities.

## Introduction
Welcome, colleague! Traditional search is limited to text matching. But the world is multimodal—we think in images, sounds, and interconnected concepts. What if you could search for a "sunset near the mountains" using an image of the Andes, or find a podcast segment that matches a text query?

**Multimodal Embeddings** make this possible. They project text and images into the *same* high-dimensional vector space. If an image of a cat and the word "feline" are close together in this space, your AI understands the *concept*, not just the keyword.

## Why This Matters

*   **Cross-Modal Retrieval**: Search your product catalog (images) using user queries (text).
*   **Rich Context**: Give your RAG pipeline eyes. Retrieve charts and diagrams, not just paragraphs.
*   **Unified Architecture**: Use the same `pkg/schema` types (`Document`) for everything.

## Prerequisites

*   A Google Cloud Project with the **Generative Language API** enabled.
*   An API Key.
*   The `pkg/embeddings` package.

## Concepts

### The MultimodalEmbedder Interface
While standard embedders only take strings, Beluga AI's `MultimodalEmbedder` accepts `schema.Document` objects enriched with metadata.
```go
type MultimodalEmbedder interface {
    EmbedDocumentsMultimodal(ctx context.Context, documents []schema.Document) ([][]float32, error)
    // ...
}
```

The provider reads special metadata keys like `image_url` or `image_base64` to understand the content.

## Step-by-Step Implementation

### Step 1: Initialize the Provider

We'll start by creating an instance of the Google Multimodal provider.
```go
package main

import (
    "context"
    "log"
    "os"
    
    "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/google_multimodal"
    "go.opentelemetry.io/otel"
)

func main() {
    apiKey := os.Getenv("GOOGLE_API_KEY")
    
    // Create the configuration
    config := &google_multimodal.Config{
        APIKey: apiKey,
        Model:  "multimodalembedding", // or "text-embedding-004" for newer models
    }
    
    // Initialize the embedder
    // We pass a tracer for observability
    tracer := otel.Tracer("my-app")
    embedder, err := google_multimodal.NewGoogleMultimodalEmbedder(config, tracer)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Step 2: Preparing Multimodal Documents

The `pkg/schema.Document` struct is flexible. We put the text description in `PageContent` and the image data in `Metadata`.
```text
import "github.com/lookatitude/beluga-ai/pkg/schema"
go
// Create a text document
textDoc := schema.Document{
    PageContent: "A happy dog playing in the park",
    Metadata:    map[string]string{"type": "text"},
}

// Create an image document (by URL)
imageDoc := schema.Document{
    PageContent: "", // Can be empty for pure image embedding
    Metadata: map[string]string{
        "type":      "image",
        "image_url": "https://example.com/dog.jpg", // The provider will fetch this
    },
}
```

### Step 3: Generating Embeddings

Now we pass these documents to the `EmbedDocumentsMultimodal` method.
```go
ctx := context.Background()

docs := []schema.Document{textDoc, imageDoc}
// Generate embeddings
embeddings, err := embedder.EmbedDocumentsMultimodal(ctx, docs)
text
if err != nil \{
    log.Fatal(err)
}
go
// Result is a slice of vectors
textVector := embeddings[0]
imageVector := embeddings[1]


```
log.Printf("Text Vector Dimension: %d", len(textVector))
log.Printf("Image Vector Dimension: %d", len(imageVector))

### Step 4: Comparing Vectors (Cosine Similarity)

To see if they match, we calculate the cosine similarity. A value close to 1.0 means they are very similar (the image matches the text).
```text
import "github.com/lookatitude/beluga-ai/pkg/embeddings/internal/math"
go
func main() {
    // ... (previous code) ...
    
    similarity := math.CosineSimilarity(textVector, imageVector)

    

    log.Printf("Similarity Score: %.4f", similarity)
    
    if similarity > 0.7 \{
        log.Println("It's a match! The image likely depicts the text description.")
    }
}
```

## Pro-Tips

*   **Batching**: Google's API has limits. The `EmbedDocumentsMultimodal` method handles single requests, but for production, you should batch your documents (e.g., groups of 100) to avoid timeouts.
*   **Base64 vs URLs**: URLs are cleaner, but they require the image to be publicly accessible. For local images or private uploads, read the file bytes, base64 encode them, and use the `image_base64` metadata key instead.
*   **Dimensions**: Ensure your vector database (e.g., Chroma, PGVector) is configured with the correct dimension size. Google's multimodal models typically output 768 or 1408 dimensions depending on the version. Check `embedder.GetDimension(ctx)` at runtime!

## Troubleshooting

### "Image fetch failed"
If you use `image_url`, the server running the specific customized Beluga AI code (or Google's servers) needs to be able to reach that URL. If you are developing locally with `localhost` URLs, Google's API cannot see them. Use Base64 for local development.

### "Dimension Mismatch"
If you try to compare a text vector from OpenAI (`1536`) with an image vector from Google (`768`), the math will panic or yield garbage. **Always use the same model** for both the query and the targets you want to compare.

## Conclusion

You have unlocked the power of vision. By treating images as just another form of "document," you can build search systems that understand the world the way humans do. Combine this with `pkg/vectorstores` (next on our roadmap!) to scale this up to millions of images.
