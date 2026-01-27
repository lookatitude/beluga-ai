package rag

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
)

// mu provides synchronization for ragPipeline.
var mu sync.RWMutex

// Query executes a RAG query and returns the generated answer.
func (p *ragPipeline) Query(ctx context.Context, query string) (string, error) {
	answer, _, err := p.QueryWithSources(ctx, query)
	return answer, err
}

// QueryWithSources executes a RAG query and returns the answer along with source documents.
func (p *ragPipeline) QueryWithSources(ctx context.Context, query string) (string, []schema.Document, error) {
	const op = "QueryWithSources"

	start := time.Now()
	ctx, span := p.metrics.StartQuerySpan(ctx, "query")
	if span != nil {
		defer span.End()
	}

	// Check if LLM is configured
	if p.llm == nil {
		p.metrics.RecordQuery(ctx, time.Since(start), 0, false)
		return "", nil, NewError(op, ErrCodeNoLLM, ErrNoLLM)
	}

	// Retrieve relevant documents
	docs, _, err := p.Search(ctx, query, p.topK)
	if err != nil {
		p.metrics.RecordQuery(ctx, time.Since(start), 0, false)
		return "", nil, NewError(op, ErrCodeRetrievalFailed, err)
	}

	// Build context from retrieved documents
	contextText := p.buildContext(docs)

	// Build messages for LLM
	messages := p.buildQueryMessages(query, contextText)

	// Generate response
	response, err := p.llm.Generate(ctx, messages)
	if err != nil {
		p.metrics.RecordQuery(ctx, time.Since(start), len(docs), false)
		return "", nil, NewError(op, ErrCodeGenerationFailed, err)
	}

	p.metrics.RecordQuery(ctx, time.Since(start), len(docs), true)
	return response.GetContent(), docs, nil
}

// IngestDocuments loads and processes documents from the configured paths.
func (p *ragPipeline) IngestDocuments(ctx context.Context) error {
	if len(p.docPaths) == 0 {
		return nil
	}
	return p.IngestFromPaths(ctx, p.docPaths)
}

// IngestFromPaths loads and processes documents from the specified paths.
func (p *ragPipeline) IngestFromPaths(ctx context.Context, paths []string) error {
	const op = "IngestFromPaths"

	start := time.Now()
	ctx, span := p.metrics.StartQuerySpan(ctx, "ingest")
	if span != nil {
		defer span.End()
	}

	// For now, we don't have document loaders integrated
	// Users should use AddDocuments directly with pre-loaded documents
	// This is a placeholder for future implementation with document loaders

	p.metrics.RecordIngestion(ctx, time.Since(start), 0, false)
	return NewErrorWithMessage(op, ErrCodeDocumentLoad,
		"document loading from paths not yet implemented, use AddDocuments directly", nil)
}

// AddDocuments directly adds pre-loaded documents to the vector store.
func (p *ragPipeline) AddDocuments(ctx context.Context, docs []schema.Document) error {
	const op = "AddDocuments"

	start := time.Now()
	ctx, span := p.metrics.StartQuerySpan(ctx, "add_documents")
	if span != nil {
		defer span.End()
	}

	if len(docs) == 0 {
		return nil
	}

	// Split documents into chunks
	chunks, err := p.splitter.SplitDocuments(ctx, docs)
	if err != nil {
		p.metrics.RecordIngestion(ctx, time.Since(start), 0, false)
		return NewError(op, ErrCodeIngestionFailed, fmt.Errorf("failed to split documents: %w", err))
	}

	// Add chunks to vector store
	_, err = p.vectorStore.AddDocuments(ctx, chunks)
	if err != nil {
		p.metrics.RecordIngestion(ctx, time.Since(start), 0, false)
		return NewError(op, ErrCodeIngestionFailed, fmt.Errorf("failed to add documents to vector store: %w", err))
	}

	mu.Lock()
	p.documentCount += len(chunks)
	mu.Unlock()

	p.metrics.RecordIngestion(ctx, time.Since(start), len(chunks), true)
	return nil
}

// AddDocumentsRaw adds documents without splitting.
func (p *ragPipeline) AddDocumentsRaw(ctx context.Context, docs []schema.Document) error {
	const op = "AddDocumentsRaw"

	start := time.Now()
	ctx, span := p.metrics.StartQuerySpan(ctx, "add_documents_raw")
	if span != nil {
		defer span.End()
	}

	if len(docs) == 0 {
		return nil
	}

	// Add directly to vector store without splitting
	_, err := p.vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		p.metrics.RecordIngestion(ctx, time.Since(start), 0, false)
		return NewError(op, ErrCodeIngestionFailed, fmt.Errorf("failed to add documents to vector store: %w", err))
	}

	mu.Lock()
	p.documentCount += len(docs)
	mu.Unlock()

	p.metrics.RecordIngestion(ctx, time.Since(start), len(docs), true)
	return nil
}

// Search performs similarity search and returns matching documents with scores.
func (p *ragPipeline) Search(ctx context.Context, query string, k int) ([]schema.Document, []float32, error) {
	const op = "Search"

	start := time.Now()
	ctx, span := p.metrics.StartQuerySpan(ctx, "search")
	if span != nil {
		defer span.End()
	}

	docs, scores, err := p.vectorStore.SimilaritySearchByQuery(ctx, query, k, p.embedder)
	if err != nil {
		p.metrics.RecordSearch(ctx, time.Since(start), false)
		return nil, nil, NewError(op, ErrCodeRetrievalFailed, err)
	}

	// Filter by score threshold if set
	if p.scoreThreshold > 0 {
		var filteredDocs []schema.Document
		var filteredScores []float32
		for i, score := range scores {
			if score >= p.scoreThreshold {
				filteredDocs = append(filteredDocs, docs[i])
				filteredScores = append(filteredScores, score)
			}
		}
		docs = filteredDocs
		scores = filteredScores
	}

	p.metrics.RecordSearch(ctx, time.Since(start), true)
	return docs, scores, nil
}

// GetDocumentCount returns the number of documents in the vector store.
func (p *ragPipeline) GetDocumentCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return p.documentCount
}

// Clear removes all documents from the vector store.
func (p *ragPipeline) Clear(ctx context.Context) error {
	// Create a new empty vector store
	p.vectorStore = newInMemoryVectorStore(p.embedder)

	mu.Lock()
	p.documentCount = 0
	mu.Unlock()

	return nil
}

// buildContext creates a context string from retrieved documents.
func (p *ragPipeline) buildContext(docs []schema.Document) string {
	if len(docs) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, doc := range docs {
		builder.WriteString(fmt.Sprintf("Document %d:\n%s\n\n", i+1, doc.GetContent()))
	}
	return builder.String()
}

// buildQueryMessages creates the messages for the LLM call.
func (p *ragPipeline) buildQueryMessages(query, context string) []schema.Message {
	var messages []schema.Message

	// Add system prompt
	systemPrompt := p.systemPrompt
	if systemPrompt == "" {
		systemPrompt = defaultRAGSystemPrompt
	}
	messages = append(messages, schema.NewSystemMessage(systemPrompt))

	// Add user message with context and query
	userContent := fmt.Sprintf("Context information:\n%s\nQuestion: %s", context, query)
	messages = append(messages, schema.NewHumanMessage(userContent))

	return messages
}

// Default RAG system prompt.
const defaultRAGSystemPrompt = `You are a helpful assistant that answers questions based on the provided context.
Use only the information from the context to answer the question.
If the context doesn't contain enough information to answer the question, say so.
Be concise and accurate in your responses.`

// Helper function to create vector store.
func newInMemoryVectorStore(embedder embeddingsiface.Embedder) *inmemory.InMemoryVectorStore {
	return inmemory.NewInMemoryVectorStore(embedder)
}

// Ensure ragPipeline implements Pipeline interface.
var _ Pipeline = (*ragPipeline)(nil)
