// Package main demonstrates multimodal RAG (Retrieval-Augmented Generation) in Beluga AI.
// This example shows how to build systems that can answer questions about images,
// text, and mixed content using vector similarity search and vision-capable LLMs.
//
// Key patterns demonstrated:
//   - Multimodal embedding creation
//   - Vector store integration for mixed content
//   - Multimodal retrieval and ranking
//   - Generation with image and text context
//   - OTEL instrumentation for observability
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// We define a tracer for observability
var tracer = otel.Tracer("beluga.rag.multimodal.example")

// DocumentType represents the type of content in a document
type DocumentType string

const (
	TypeText      DocumentType = "text"
	TypeImage     DocumentType = "image"
	TypeImageText DocumentType = "image_text"
)

// Document represents a multimodal document for indexing
type Document struct {
	ID       string
	Type     DocumentType
	Content  string  // Text content or caption
	ImageURL string  // URL for images
	Metadata map[string]any
}

// EmbeddedDocument is a document with its embedding vector
type EmbeddedDocument struct {
	Document
	Embedding []float32
}

// SearchResult represents a retrieval result with similarity score
type SearchResult struct {
	Document
	Score float32
}

// MultimodalRAGExample demonstrates multimodal RAG functionality
type MultimodalRAGExample struct {
	textEmbedder embeddings.Embedder
	llm          llmsiface.ChatModel
	documents    []EmbeddedDocument // In-memory store for demo
	topK         int
}

// NewMultimodalRAGExample creates a new multimodal RAG example
func NewMultimodalRAGExample(textEmbedder embeddings.Embedder, llm llmsiface.ChatModel, topK int) *MultimodalRAGExample {
	return &MultimodalRAGExample{
		textEmbedder: textEmbedder,
		llm:          llm,
		documents:    make([]EmbeddedDocument, 0),
		topK:         topK,
	}
}

// IndexDocuments adds documents to the RAG system
func (r *MultimodalRAGExample) IndexDocuments(ctx context.Context, docs []Document) error {
	ctx, span := tracer.Start(ctx, "multimodal_rag.index",
		trace.WithAttributes(
			attribute.Int("document_count", len(docs)),
		))
	defer span.End()

	start := time.Now()

	for _, doc := range docs {
		// Create embedding based on content
		// For text, embed the content directly
		// For images, embed the caption/description
		// For image+text, embed the combined text
		textToEmbed := doc.Content
		if doc.Type == TypeImage && textToEmbed == "" {
			// Use a placeholder description for images without captions
			textToEmbed = fmt.Sprintf("Image: %s", doc.ID)
		}

		embeddings, err := r.textEmbedder.EmbedDocuments(ctx, []string{textToEmbed})
		if err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to embed document %s: %w", doc.ID, err)
		}

		if len(embeddings) == 0 {
			continue
		}

		r.documents = append(r.documents, EmbeddedDocument{
			Document:  doc,
			Embedding: embeddings[0],
		})
	}

	span.SetAttributes(
		attribute.Int("indexed_count", len(r.documents)),
		attribute.Float64("duration_ms", float64(time.Since(start).Milliseconds())),
	)
	span.SetStatus(codes.Ok, "")

	return nil
}

// Query performs multimodal RAG to answer a question
func (r *MultimodalRAGExample) Query(ctx context.Context, question string) (string, []SearchResult, error) {
	ctx, span := tracer.Start(ctx, "multimodal_rag.query",
		trace.WithAttributes(
			attribute.String("question", question),
		))
	defer span.End()

	start := time.Now()

	// Step 1: Embed the query
	queryEmbedding, err := r.textEmbedder.EmbedQuery(ctx, question)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Step 2: Retrieve similar documents
	results := r.similaritySearch(queryEmbedding, r.topK)
	span.AddEvent("documents_retrieved", trace.WithAttributes(
		attribute.Int("count", len(results)),
	))

	// Step 3: Build context and generate response
	answer, err := r.generateAnswer(ctx, question, results)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", results, fmt.Errorf("failed to generate answer: %w", err)
	}

	span.SetAttributes(
		attribute.Int("answer_length", len(answer)),
		attribute.Float64("duration_ms", float64(time.Since(start).Milliseconds())),
	)
	span.SetStatus(codes.Ok, "")

	return answer, results, nil
}

// similaritySearch finds the most similar documents using cosine similarity
func (r *MultimodalRAGExample) similaritySearch(query []float32, k int) []SearchResult {
	type scored struct {
		doc   EmbeddedDocument
		score float32
	}

	// Calculate similarity scores
	scores := make([]scored, 0, len(r.documents))
	for _, doc := range r.documents {
		score := cosineSimilarity(query, doc.Embedding)
		scores = append(scores, scored{doc: doc, score: score})
	}

	// Sort by score (descending)
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Take top-k
	if k > len(scores) {
		k = len(scores)
	}

	results := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		results[i] = SearchResult{
			Document: scores[i].doc.Document,
			Score:    scores[i].score,
		}
	}

	return results
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt calculates square root for float32
func sqrt(x float32) float32 {
	if x <= 0 {
		return 0
	}
	// Newton's method
	z := x / 2
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// generateAnswer creates a response using the LLM with retrieved context
func (r *MultimodalRAGExample) generateAnswer(ctx context.Context, question string, results []SearchResult) (string, error) {
	ctx, span := tracer.Start(ctx, "multimodal_rag.generate")
	defer span.End()

	// Build context from retrieved documents
	var contextBuilder string
	contextBuilder += "Context information:\n\n"

	for i, result := range results {
		contextBuilder += fmt.Sprintf("[Document %d - %s]\n", i+1, result.Type)
		
		switch result.Type {
		case TypeText:
			contextBuilder += fmt.Sprintf("Content: %s\n", result.Content)
		case TypeImage:
			contextBuilder += fmt.Sprintf("Image URL: %s\n", result.ImageURL)
			if result.Content != "" {
				contextBuilder += fmt.Sprintf("Caption: %s\n", result.Content)
			}
		case TypeImageText:
			contextBuilder += fmt.Sprintf("Image URL: %s\n", result.ImageURL)
			contextBuilder += fmt.Sprintf("Description: %s\n", result.Content)
		}
		contextBuilder += fmt.Sprintf("Relevance Score: %.3f\n\n", result.Score)
	}

	// Create messages for the LLM
	systemPrompt := `You are a helpful assistant that answers questions based on the provided context.
The context may include text documents, image descriptions, and image URLs.
Analyze all provided content carefully and synthesize a comprehensive answer.
If you reference an image, describe what information it would contain based on the caption or description.
Be concise but thorough in your response.`

	userPrompt := fmt.Sprintf("%s\nQuestion: %s\n\nPlease provide a comprehensive answer based on the context above.", contextBuilder, question)

	messages := []schema.Message{
		schema.NewSystemMessage(systemPrompt),
		schema.NewHumanMessage(userPrompt),
	}

	// Generate response
	response, err := r.llm.Generate(ctx, messages)
	if err != nil {
		span.RecordError(err)
		return "", fmt.Errorf("LLM generation failed: %w", err)
	}

	return response.GetContent(), nil
}

// GetDocumentCount returns the number of indexed documents
func (r *MultimodalRAGExample) GetDocumentCount() int {
	return len(r.documents)
}

// createSampleDocuments creates sample documents for demonstration
func createSampleDocuments() []Document {
	return []Document{
		{
			ID:      "doc1",
			Type:    TypeText,
			Content: "Beluga whales are medium-sized toothed whales known for their distinctive white coloration. They are highly social animals that live in groups called pods and are found in Arctic and sub-Arctic waters.",
		},
		{
			ID:       "doc2",
			Type:     TypeImage,
			ImageURL: "https://example.com/beluga-whale.jpg",
			Content:  "A beluga whale swimming gracefully in clear arctic waters, displaying its characteristic white skin and rounded forehead (melon)",
		},
		{
			ID:      "doc3",
			Type:    TypeText,
			Content: "Beluga whales communicate using a variety of clicks, whistles, and other vocalizations. They are sometimes called 'canaries of the sea' because of their diverse range of sounds.",
		},
		{
			ID:       "doc4",
			Type:     TypeImageText,
			ImageURL: "https://example.com/beluga-migration.png",
			Content:  "Map showing beluga whale migration patterns across the Arctic. Arrows indicate seasonal movement from summer feeding grounds in shallow coastal areas to deeper offshore waters in winter.",
		},
		{
			ID:      "doc5",
			Type:    TypeText,
			Content: "Unlike most whale species, beluga whales can turn their heads due to unfused cervical vertebrae. They can also change the shape of their melon (forehead) to focus their echolocation signals.",
		},
		{
			ID:       "doc6",
			Type:     TypeImage,
			ImageURL: "https://example.com/beluga-pod.jpg",
			Content:  "A pod of approximately 15 beluga whales swimming together, showcasing their social behavior. Adult whales are pure white while younger calves show a grayish coloration.",
		},
	}
}

func main() {
	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create embedding provider
	embeddingProvider, err := embeddings.NewOpenAIEmbeddings(
		embeddings.WithAPIKey(apiKey),
		embeddings.WithModel("text-embedding-3-small"),
	)
	if err != nil {
		log.Fatalf("Failed to create embedding provider: %v", err)
	}

	// Create LLM client (use a model that can understand multimodal context descriptions)
	llmClient, err := llms.NewOpenAIChat(
		llms.WithAPIKey(apiKey),
		llms.WithModel("gpt-4"),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	// Create the multimodal RAG example
	rag := NewMultimodalRAGExample(embeddingProvider, llmClient, 3)

	fmt.Println("=== Multimodal RAG Example ===")
	fmt.Println()

	// Index sample documents
	fmt.Println("Indexing documents...")
	documents := createSampleDocuments()
	if err := rag.IndexDocuments(ctx, documents); err != nil {
		log.Fatalf("Failed to index documents: %v", err)
	}
	fmt.Printf("Indexed %d documents\n\n", rag.GetDocumentCount())

	// Run some queries
	queries := []string{
		"What do beluga whales look like?",
		"How do beluga whales communicate?",
		"Where do beluga whales migrate?",
	}

	for _, query := range queries {
		fmt.Printf("Question: %s\n", query)
		fmt.Println("---")

		answer, results, err := rag.Query(ctx, query)
		if err != nil {
			log.Printf("Query failed: %v", err)
			continue
		}

		fmt.Println("Retrieved documents:")
		for i, result := range results {
			fmt.Printf("  %d. [%s] Score: %.3f - %s\n", i+1, result.Type, result.Score, truncate(result.Content, 60))
		}
		fmt.Println()
		fmt.Printf("Answer: %s\n", answer)
		fmt.Println()
		fmt.Println("=" + "=================================================")
		fmt.Println()
	}
}

// truncate shortens a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
