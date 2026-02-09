---
title: "Metadata-Aware Embedding Clusters"
description: "Cluster embeddings while preserving metadata constraints to create semantically and domain-coherent groups."
---

## Problem

You need to cluster embeddings while preserving and utilizing metadata (categories, tags, timestamps) to create more meaningful clusters that respect domain boundaries and improve retrieval quality.

## Solution

Implement clustering algorithms that incorporate metadata as constraints or features, allowing you to create clusters that are both semantically similar and metadata-coherent. Combining embedding similarity with metadata constraints produces clusters that are more useful for real-world applications.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "math"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.embeddings.clustering")

// MetadataAwareClusterer clusters embeddings with metadata constraints.
type MetadataAwareClusterer struct {
    embedder       embedding.Embedder
    metadataWeight float64 // Weight for metadata similarity (0-1)
    minClusterSize int
}

// DocumentWithEmbedding represents a document with its embedding.
type DocumentWithEmbedding struct {
    Document  schema.Document
    Embedding []float32
}

// Cluster represents a group of similar documents.
type Cluster struct {
    ID        string
    Documents []DocumentWithEmbedding
    Centroid  []float32
    Metadata  map[string]interface{}
}

// NewMetadataAwareClusterer creates a new clusterer.
func NewMetadataAwareClusterer(embedder embedding.Embedder, metadataWeight float64, minClusterSize int) *MetadataAwareClusterer {
    return &MetadataAwareClusterer{
        embedder:       embedder,
        metadataWeight: metadataWeight,
        minClusterSize: minClusterSize,
    }
}

// ClusterDocuments clusters documents using embeddings and metadata.
func (mac *MetadataAwareClusterer) ClusterDocuments(ctx context.Context, documents []schema.Document, k int) ([]Cluster, error) {
    ctx, span := tracer.Start(ctx, "clusterer.cluster_documents")
    defer span.End()

    span.SetAttributes(
        attribute.Int("document_count", len(documents)),
        attribute.Int("k_clusters", k),
        attribute.Float64("metadata_weight", mac.metadataWeight),
    )

    // Generate embeddings
    texts := make([]string, len(documents))
    for i, doc := range documents {
        texts[i] = doc.GetContent()
    }

    embeddings, err := mac.embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    // Create document-embedding pairs
    docsWithEmbeddings := make([]DocumentWithEmbedding, len(documents))
    for i := range documents {
        docsWithEmbeddings[i] = DocumentWithEmbedding{
            Document:  documents[i],
            Embedding: embeddings[i],
        }
    }

    // Perform metadata-aware clustering
    clusters, err := mac.performClustering(ctx, docsWithEmbeddings, k)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }

    span.SetAttributes(attribute.Int("cluster_count", len(clusters)))
    span.SetStatus(trace.StatusOK, "clustering completed")

    return clusters, nil
}

// performClustering performs k-means clustering with metadata constraints.
func (mac *MetadataAwareClusterer) performClustering(ctx context.Context, docs []DocumentWithEmbedding, k int) ([]Cluster, error) {
    clusters := make([]Cluster, k)
    for i := 0; i < k; i++ {
        clusters[i] = Cluster{
            ID:        fmt.Sprintf("cluster-%d", i),
            Documents: []DocumentWithEmbedding{},
            Metadata:  make(map[string]interface{}),
        }
    }

    // Initialize centroids
    for i := range clusters {
        clusters[i].Centroid = docs[i%len(docs)].Embedding
    }

    // K-means iteration
    maxIterations := 100
    for iteration := 0; iteration < maxIterations; iteration++ {
        for i := range clusters {
            clusters[i].Documents = []DocumentWithEmbedding{}
        }

        // Assign documents to clusters
        for _, doc := range docs {
            bestCluster := mac.findBestCluster(doc, clusters)
            clusters[bestCluster].Documents = append(clusters[bestCluster].Documents, doc)
        }

        // Update centroids
        changed := false
        for i := range clusters {
            if len(clusters[i].Documents) > 0 {
                newCentroid := mac.calculateCentroid(clusters[i].Documents)
                if !mac.vectorsEqual(clusters[i].Centroid, newCentroid) {
                    clusters[i].Centroid = newCentroid
                    changed = true
                }
            }
        }

        if !changed {
            break
        }
    }

    // Filter small clusters and aggregate metadata
    filteredClusters := []Cluster{}
    for _, cluster := range clusters {
        if len(cluster.Documents) >= mac.minClusterSize {
            cluster.Metadata = mac.aggregateMetadata(cluster.Documents)
            filteredClusters = append(filteredClusters, cluster)
        }
    }

    return filteredClusters, nil
}

// findBestCluster finds the best cluster for a document using combined similarity.
func (mac *MetadataAwareClusterer) findBestCluster(doc DocumentWithEmbedding, clusters []Cluster) int {
    bestIdx := 0
    bestScore := math.MaxFloat32

    for i, cluster := range clusters {
        embeddingSim := mac.cosineSimilarity(doc.Embedding, cluster.Centroid)
        metadataSim := mac.metadataSimilarity(doc.Document, cluster.Documents)

        score := (1-mac.metadataWeight)*(1-embeddingSim) + mac.metadataWeight*(1-metadataSim)

        if score < bestScore {
            bestScore = score
            bestIdx = i
        }
    }

    return bestIdx
}

// cosineSimilarity calculates cosine similarity between two vectors.
func (mac *MetadataAwareClusterer) cosineSimilarity(a, b []float32) float64 {
    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += float64(a[i] * b[i])
        normA += float64(a[i] * a[i])
        normB += float64(b[i] * b[i])
    }
    if normA == 0 || normB == 0 {
        return 0
    }
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// metadataSimilarity calculates metadata overlap between a document and a cluster.
func (mac *MetadataAwareClusterer) metadataSimilarity(doc schema.Document, clusterDocs []DocumentWithEmbedding) float64 {
    if len(clusterDocs) == 0 {
        return 0
    }

    docMeta := doc.GetMetadata()
    matches := 0

    for _, clusterDoc := range clusterDocs {
        clusterMeta := clusterDoc.Document.GetMetadata()
        for key, value := range docMeta {
            if clusterValue, exists := clusterMeta[key]; exists && clusterValue == value {
                matches++
            }
        }
    }

    totalComparisons := len(clusterDocs) * len(docMeta)
    if totalComparisons == 0 {
        return 0
    }

    return float64(matches) / float64(totalComparisons)
}

// calculateCentroid calculates the centroid of a cluster.
func (mac *MetadataAwareClusterer) calculateCentroid(docs []DocumentWithEmbedding) []float32 {
    if len(docs) == 0 {
        return nil
    }

    dim := len(docs[0].Embedding)
    centroid := make([]float32, dim)

    for _, doc := range docs {
        for i := range doc.Embedding {
            centroid[i] += doc.Embedding[i]
        }
    }

    for i := range centroid {
        centroid[i] /= float32(len(docs))
    }

    return centroid
}

// aggregateMetadata aggregates metadata from cluster documents.
func (mac *MetadataAwareClusterer) aggregateMetadata(docs []DocumentWithEmbedding) map[string]interface{} {
    aggregated := make(map[string]interface{})
    valueCounts := make(map[string]map[interface{}]int)

    for _, doc := range docs {
        meta := doc.Document.GetMetadata()
        for key, value := range meta {
            if valueCounts[key] == nil {
                valueCounts[key] = make(map[interface{}]int)
            }
            valueCounts[key][value]++
        }
    }

    for key, counts := range valueCounts {
        maxCount := 0
        var maxValue interface{}
        for value, count := range counts {
            if count > maxCount {
                maxCount = count
                maxValue = value
            }
        }
        aggregated[key] = maxValue
    }

    return aggregated
}

// vectorsEqual checks if two vectors are approximately equal.
func (mac *MetadataAwareClusterer) vectorsEqual(a, b []float32) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if math.Abs(float64(a[i]-b[i])) > 0.0001 {
            return false
        }
    }
    return true
}

func main() {
    ctx := context.Background()

    // embedder := your embedding.Embedder instance
    clusterer := NewMetadataAwareClusterer(embedder, 0.3, 5)

    documents := []schema.Document{
        schema.NewDocument("Document 1", map[string]string{"category": "tech"}),
        // ... more documents
    }

    clusters, err := clusterer.ClusterDocuments(ctx, documents, 10)
    if err != nil {
        log.Fatalf("Failed to cluster: %v", err)
    }
    fmt.Printf("Created %d clusters\n", len(clusters))
}
```

## Explanation

1. **Combined similarity metric** — Embedding similarity is combined with metadata similarity using a configurable weight parameter. This balances semantic similarity with domain constraints (e.g., keep documents from the same category together).

2. **Metadata aggregation** — Metadata from cluster members is aggregated to create cluster-level metadata. This helps identify what each cluster represents (e.g., "tech documents from Q4").

3. **Minimum cluster size** — Clusters below a minimum size are filtered out. This prevents creating too many tiny clusters that are not useful for retrieval.

4. **Domain boundary respect** — A document about "Python" in the "programming" category should not cluster with "Python" the snake in the "animals" category, even if embeddings are similar. The metadata weight prevents this.

## Variations

### Hierarchical Clustering

Use hierarchical clustering for nested categories:

```go
type HierarchicalClusterer struct {
    // Build tree of clusters
}
```

### Dynamic K Selection

Automatically determine optimal k:

```go
func (mac *MetadataAwareClusterer) FindOptimalK(ctx context.Context, docs []DocumentWithEmbedding) int {
    // Use elbow method or silhouette score
}
```

## Related Recipes

- [Batch Embedding Optimization](/cookbook/batch-embeddings) — Optimize batch embedding operations
- [Advanced Metadata Filtering](/cookbook/meta-filtering) — Filter vector store results with metadata
