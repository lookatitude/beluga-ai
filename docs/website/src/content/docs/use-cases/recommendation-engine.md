---
title: Intelligent Recommendation Engine
description: Build semantic product recommendations with vector similarity search to solve cold-start problems using Beluga AI's vector store package.
---

E-commerce platforms relying on traditional collaborative filtering face cold-start problems with new users and products, deliver limited personalization, and cannot understand semantic product relationships. A semantic recommendation engine uses vector embeddings to represent products and user preferences, enabling content-based recommendations that work from day one and understand why products are similar beyond purchase history.

## Solution Architecture

Beluga AI's vector store package provides similarity search at scale. The recommendation engine embeds product descriptions and user preferences into vector space, uses similarity search to find relevant products, and applies hybrid ranking that combines content-based and collaborative signals for optimal accuracy.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Product    │───▶│   Product    │───▶│ Vector Store │
│   Catalog    │    │   Embedder   │    │              │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
┌──────────────┐    ┌──────────────┐           │
│     User     │───▶│  User Pref   │           │
│   Behavior   │    │   Engine     │           │
└──────────────┘    └──────┬───────┘           │
                           │                   │
                           ▼                   ▼
                    ┌──────────────────────────┐
                    │   Similarity Search      │
                    │   + Hybrid Ranker        │
                    └──────┬───────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Personalized │
                    │Recommendations│
                    └──────────────┘
```

## Product Embedding

Generate semantic vectors for product catalog:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

type RecommendationEngine struct {
    embedder    embedding.Embedder
    vectorStore vectorstore.VectorStore
}

func NewRecommendationEngine(ctx context.Context) (*RecommendationEngine, error) {
    embedder, err := embedding.New("openai", &embedding.Config{
        Model:  "text-embedding-3-large",
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    store, err := vectorstore.New("pgvector", &vectorstore.Config{
        ConnectionString: os.Getenv("DATABASE_URL"),
        CollectionName:   "products",
        Dimensions:       3072,
    })
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    return &RecommendationEngine{
        embedder:    embedder,
        vectorStore: store,
    }, nil
}

type ProductMetadata struct {
    ProductID   string
    Name        string
    Description string
    Category    string
    Price       float64
    Brand       string
    Tags        []string
}

func (r *RecommendationEngine) IndexProduct(ctx context.Context, product ProductMetadata) error {
    // Create rich product description for embedding
    description := fmt.Sprintf(
        "Product: %s\nBrand: %s\nCategory: %s\nDescription: %s\nTags: %s",
        product.Name,
        product.Brand,
        product.Category,
        product.Description,
        strings.Join(product.Tags, ", "),
    )

    // Generate embedding
    embeddings, err := r.embedder.Embed(ctx, []string{description})
    if err != nil {
        return fmt.Errorf("embed product: %w", err)
    }

    // Store in vector database
    doc := schema.Document{
        Content:   description,
        Embedding: embeddings[0],
        Metadata: map[string]interface{}{
            "product_id":  product.ProductID,
            "name":        product.Name,
            "category":    product.Category,
            "price":       product.Price,
            "brand":       product.Brand,
            "tags":        product.Tags,
        },
    }

    if err := r.vectorStore.Add(ctx, []schema.Document{doc}); err != nil {
        return fmt.Errorf("store product: %w", err)
    }

    return nil
}
```

## User Preference Learning

Build user preference vectors from behavior:

```go
package main

import (
    "context"
)

type UserPreferenceEngine struct {
    embedder embedding.Embedder
}

type UserBehavior struct {
    ViewedProducts    []string
    PurchasedProducts []string
    SearchQueries     []string
    ClickedCategories []string
}

func (u *UserPreferenceEngine) GetUserVector(ctx context.Context, userID string) ([]float64, error) {
    // Fetch user behavior
    behavior, err := u.fetchUserBehavior(ctx, userID)
    if err != nil {
        return nil, err
    }

    if len(behavior.PurchasedProducts) == 0 && len(behavior.ViewedProducts) == 0 {
        return nil, fmt.Errorf("insufficient user data for %s", userID)
    }

    // Build user preference description from behavior
    var preferences strings.Builder

    if len(behavior.SearchQueries) > 0 {
        preferences.WriteString("User searches: ")
        preferences.WriteString(strings.Join(behavior.SearchQueries, ", "))
        preferences.WriteString(". ")
    }

    if len(behavior.ClickedCategories) > 0 {
        preferences.WriteString("Interested in categories: ")
        preferences.WriteString(strings.Join(behavior.ClickedCategories, ", "))
        preferences.WriteString(". ")
    }

    if len(behavior.PurchasedProducts) > 0 {
        preferences.WriteString("Previously purchased: ")
        for _, productID := range behavior.PurchasedProducts {
            product := u.getProductInfo(ctx, productID)
            preferences.WriteString(fmt.Sprintf("%s, ", product.Name))
        }
    }

    // Generate preference embedding
    embeddings, err := u.embedder.Embed(ctx, []string{preferences.String()})
    if err != nil {
        return nil, fmt.Errorf("embed preferences: %w", err)
    }

    return embeddings[0], nil
}

func (u *UserPreferenceEngine) fetchUserBehavior(ctx context.Context, userID string) (*UserBehavior, error) {
    // Fetch from database or analytics system
    // Implementation depends on your data storage
    return &UserBehavior{}, nil
}

func (u *UserPreferenceEngine) getProductInfo(ctx context.Context, productID string) *ProductMetadata {
    // Fetch product details
    return &ProductMetadata{}
}
```

## Personalized Recommendations

Generate recommendations using vector similarity:

```go
package main

import (
    "context"
)

type Recommendation struct {
    ProductID   string
    Name        string
    Category    string
    Price       float64
    Score       float64
    Reason      string
}

func (r *RecommendationEngine) GetRecommendations(ctx context.Context, userID string, limit int) ([]Recommendation, error) {
    // Get user preference vector
    userPrefs := NewUserPreferenceEngine(r.embedder)
    userVector, err := userPrefs.GetUserVector(ctx, userID)
    if err != nil {
        // Cold-start: use popular products or category-based recommendations
        return r.getColdStartRecommendations(ctx, limit)
    }

    // Search for similar products
    results, err := r.vectorStore.SimilaritySearch(ctx, userVector,
        vectorstore.WithTopK(limit*2), // Get more for ranking
        vectorstore.WithThreshold(0.6),
    )
    if err != nil {
        return nil, fmt.Errorf("similarity search: %w", err)
    }

    // Rank and filter
    recommendations := r.rankRecommendations(ctx, results, userID, limit)

    return recommendations, nil
}

func (r *RecommendationEngine) rankRecommendations(ctx context.Context, docs []schema.Document, userID string, limit int) []Recommendation {
    recommendations := make([]Recommendation, 0, len(docs))

    // Apply business logic and filters
    for _, doc := range docs {
        // Skip already purchased products
        if r.hasUserPurchased(ctx, userID, doc.Metadata["product_id"].(string)) {
            continue
        }

        // Apply diversity filter
        // Apply price range filter
        // Apply inventory filter

        recommendations = append(recommendations, Recommendation{
            ProductID: doc.Metadata["product_id"].(string),
            Name:      doc.Metadata["name"].(string),
            Category:  doc.Metadata["category"].(string),
            Price:     doc.Metadata["price"].(float64),
            Score:     doc.Metadata["score"].(float64),
            Reason:    r.generateReason(doc),
        })

        if len(recommendations) >= limit {
            break
        }
    }

    return recommendations
}

func (r *RecommendationEngine) generateReason(doc schema.Document) string {
    // Generate human-readable reason for recommendation
    return fmt.Sprintf("Based on your interest in %s", doc.Metadata["category"])
}

func (r *RecommendationEngine) hasUserPurchased(ctx context.Context, userID, productID string) bool {
    // Check purchase history
    return false
}
```

## Cold-Start Handling

Recommend to new users using content similarity:

```go
package main

import (
    "context"
)

func (r *RecommendationEngine) getColdStartRecommendations(ctx context.Context, limit int) ([]Recommendation, error) {
    // Strategy 1: Popular products
    // Strategy 2: New arrivals
    // Strategy 3: Trending in category

    // For this example, use popular products
    popularProducts, err := r.getPopularProducts(ctx, limit)
    if err != nil {
        return nil, err
    }

    recommendations := make([]Recommendation, len(popularProducts))
    for i, product := range popularProducts {
        recommendations[i] = Recommendation{
            ProductID: product.ProductID,
            Name:      product.Name,
            Category:  product.Category,
            Price:     product.Price,
            Score:     1.0,
            Reason:    "Popular choice",
        }
    }

    return recommendations, nil
}

func (r *RecommendationEngine) GetSimilarProducts(ctx context.Context, productID string, limit int) ([]Recommendation, error) {
    // Get product embedding
    doc, err := r.vectorStore.Get(ctx, productID)
    if err != nil {
        return nil, fmt.Errorf("get product: %w", err)
    }

    // Find similar products
    results, err := r.vectorStore.SimilaritySearch(ctx, doc.Embedding,
        vectorstore.WithTopK(limit+1), // +1 to exclude self
        vectorstore.WithThreshold(0.7),
    )
    if err != nil {
        return nil, fmt.Errorf("similarity search: %w", err)
    }

    recommendations := make([]Recommendation, 0)
    for _, result := range results {
        // Skip the product itself
        if result.Metadata["product_id"].(string) == productID {
            continue
        }

        recommendations = append(recommendations, Recommendation{
            ProductID: result.Metadata["product_id"].(string),
            Name:      result.Metadata["name"].(string),
            Category:  result.Metadata["category"].(string),
            Price:     result.Metadata["price"].(float64),
            Score:     result.Metadata["score"].(float64),
            Reason:    "Similar product",
        })

        if len(recommendations) >= limit {
            break
        }
    }

    return recommendations, nil
}
```

## Hybrid Ranking

Combine multiple signals for better recommendations:

```go
package main

import (
    "context"
)

type HybridRanker struct {
    weights map[string]float64
}

func NewHybridRanker() *HybridRanker {
    return &HybridRanker{
        weights: map[string]float64{
            "semantic_similarity": 0.4,
            "popularity":          0.2,
            "user_affinity":       0.2,
            "freshness":           0.1,
            "price_match":         0.1,
        },
    }
}

func (h *HybridRanker) Rank(ctx context.Context, docs []schema.Document, userID string) []schema.Document {
    scored := make([]scoredDoc, len(docs))

    for i, doc := range docs {
        score := 0.0

        // Semantic similarity (from vector search)
        score += doc.Metadata["score"].(float64) * h.weights["semantic_similarity"]

        // Popularity (view count, purchase count)
        popularity := h.calculatePopularity(doc)
        score += popularity * h.weights["popularity"]

        // User affinity (category match, brand preference)
        affinity := h.calculateUserAffinity(ctx, userID, doc)
        score += affinity * h.weights["user_affinity"]

        // Freshness (newer products ranked higher)
        freshness := h.calculateFreshness(doc)
        score += freshness * h.weights["freshness"]

        // Price match (user's typical price range)
        priceMatch := h.calculatePriceMatch(ctx, userID, doc)
        score += priceMatch * h.weights["price_match"]

        scored[i] = scoredDoc{
            doc:   doc,
            score: score,
        }
    }

    // Sort by combined score
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })

    result := make([]schema.Document, len(scored))
    for i, s := range scored {
        result[i] = s.doc
        result[i].Metadata["final_score"] = s.score
    }

    return result
}

type scoredDoc struct {
    doc   schema.Document
    score float64
}

func (h *HybridRanker) calculatePopularity(doc schema.Document) float64 {
    // Implementation: combine view count, purchase count, rating
    return 0.5
}

func (h *HybridRanker) calculateUserAffinity(ctx context.Context, userID string, doc schema.Document) float64 {
    // Implementation: check category preferences, brand loyalty
    return 0.5
}

func (h *HybridRanker) calculateFreshness(doc schema.Document) float64 {
    // Implementation: time since product creation
    return 0.5
}

func (h *HybridRanker) calculatePriceMatch(ctx context.Context, userID string, doc schema.Document) float64 {
    // Implementation: compare to user's typical price range
    return 0.5
}
```

## Production Considerations

### Observability

Track recommendation quality and user engagement:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel/attribute"
)

func (r *RecommendationEngine) GetRecommendationsWithTracking(ctx context.Context, userID string, limit int) ([]Recommendation, error) {
    ctx, span := tracer.Start(ctx, "recommendation.get")
    defer span.End()

    span.SetAttributes(
        attribute.String("user.id", userID),
        attribute.Int("limit", limit),
    )

    start := time.Now()
    recommendations, err := r.GetRecommendations(ctx, userID, limit)
    duration := time.Since(start)

    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.Int("recommendations.count", len(recommendations)),
        attribute.Float64("duration.ms", float64(duration.Milliseconds())),
    )

    meter.RecordHistogram(ctx, "recommendation.duration", duration.Milliseconds())
    meter.IncrementCounter(ctx, "recommendations.generated")

    return recommendations, nil
}
```

### Caching

Cache recommendations for frequent users:

```go
import "github.com/lookatitude/beluga-ai/cache"

type CachedRecommendationEngine struct {
    RecommendationEngine
    cache cache.Cache
}

func (c *CachedRecommendationEngine) GetRecommendations(ctx context.Context, userID string, limit int) ([]Recommendation, error) {
    cacheKey := fmt.Sprintf("rec:%s:%d", userID, limit)

    // Check cache
    if cached, ok := c.cache.Get(ctx, cacheKey); ok {
        return cached.([]Recommendation), nil
    }

    // Generate recommendations
    recommendations, err := c.RecommendationEngine.GetRecommendations(ctx, userID, limit)
    if err != nil {
        return nil, err
    }

    // Cache for 5 minutes
    c.cache.Set(ctx, cacheKey, recommendations, 5*time.Minute)

    return recommendations, nil
}
```

### A/B Testing

Track recommendation strategy performance:

```go
type ABTestingEngine struct {
    RecommendationEngine
    experiments map[string]RecommendationStrategy
}

type RecommendationStrategy interface {
    GetRecommendations(ctx context.Context, userID string, limit int) ([]Recommendation, error)
}

func (a *ABTestingEngine) GetRecommendations(ctx context.Context, userID string, limit int) ([]Recommendation, error) {
    // Assign user to experiment group
    variant := a.getVariant(userID)

    strategy := a.experiments[variant]
    recommendations, err := strategy.GetRecommendations(ctx, userID, limit)
    if err != nil {
        return nil, err
    }

    // Track which variant was served
    a.trackVariant(ctx, userID, variant)

    return recommendations, nil
}

func (a *ABTestingEngine) getVariant(userID string) string {
    // Hash-based assignment for consistent experience
    hash := hashUserID(userID)
    if hash%2 == 0 {
        return "semantic"
    }
    return "hybrid"
}
```

### Batch Updates

Update product embeddings efficiently:

```go
func (r *RecommendationEngine) IndexBatch(ctx context.Context, products []ProductMetadata) error {
    batchSize := 100

    for i := 0; i < len(products); i += batchSize {
        end := min(i+batchSize, len(products))
        batch := products[i:end]

        // Generate descriptions
        descriptions := make([]string, len(batch))
        for j, product := range batch {
            descriptions[j] = fmt.Sprintf(
                "Product: %s\nBrand: %s\nCategory: %s\nDescription: %s",
                product.Name, product.Brand, product.Category, product.Description,
            )
        }

        // Batch embed
        embeddings, err := r.embedder.EmbedBatch(ctx, descriptions)
        if err != nil {
            return err
        }

        // Create documents
        docs := make([]schema.Document, len(batch))
        for j, product := range batch {
            docs[j] = schema.Document{
                Content:   descriptions[j],
                Embedding: embeddings[j],
                Metadata: map[string]interface{}{
                    "product_id": product.ProductID,
                    "name":       product.Name,
                    "category":   product.Category,
                    "price":      product.Price,
                },
            }
        }

        // Batch store
        if err := r.vectorStore.Add(ctx, docs); err != nil {
            return err
        }
    }

    return nil
}
```

## Related Resources

- [Vector Store Guide](/guides/vector-stores/) for scaling strategies
- [Embedding Guide](/guides/embeddings/) for embedding optimization
- [Knowledge QA System](/use-cases/knowledge-qa/) for similarity patterns
- [Enterprise RAG](/use-cases/enterprise-rag/) for RAG pipeline patterns
