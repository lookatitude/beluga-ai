// Package internal provides internal implementation details for the multimodal package.
package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	chatmodelsiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// BaseMultimodalModel provides a base implementation of MultimodalModel.
type BaseMultimodalModel struct {
	llmProvider        chatmodelsiface.ChatModel
	embedder           embeddingsiface.Embedder
	multimodalEmbedder embeddingsiface.MultimodalEmbedder
	vectorStore        vectorstores.VectorStore
	config             map[string]any
	capabilities       *types.ModalityCapabilities
	router             *Router
	normalizer         *Normalizer
	streamingState     *StreamingState
	providerName       string
	modelName          string
	streamingStateMu   sync.Mutex
}

// StreamingState manages state for streaming operations.
type StreamingState struct {
	activeStreams map[string]context.CancelFunc    // Map of input ID to cancel function
	chunkBuffers  map[string][]*types.ContentBlock // Map of input ID to chunk buffers
	mu            sync.RWMutex
}

// NewStreamingState creates a new streaming state.
func NewStreamingState() *StreamingState {
	return &StreamingState{
		activeStreams: make(map[string]context.CancelFunc),
		chunkBuffers:  make(map[string][]*types.ContentBlock),
	}
}

// NewBaseMultimodalModel creates a new base multimodal model.
func NewBaseMultimodalModel(providerName, modelName string, config map[string]any, capabilities *types.ModalityCapabilities) *BaseMultimodalModel {
	router := NewRouter(registry.GetRegistry())
	normalizer := NewNormalizer()

	return &BaseMultimodalModel{
		providerName:   providerName,
		modelName:      modelName,
		config:         config,
		capabilities:   capabilities,
		router:         router,
		normalizer:     normalizer,
		streamingState: NewStreamingState(),
	}
}

// Process processes a multimodal input and returns a multimodal output.
func (m *BaseMultimodalModel) Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.Process",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
		))
	defer span.End()

	// Metrics recording would go here if metrics were available
	// For now, we avoid importing multimodal package to prevent import cycles

	// Validate input - basic validation
	if len(input.ContentBlocks) == 0 {
		err := errors.New("input must have at least one content block")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Input validation failed", "error", err)
		return nil, err
	}

	// Route content blocks to appropriate providers
	routing, err := m.router.Route(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Content routing failed", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("routed_providers_count", len(routing)))
	logWithOTELContext(ctx, slog.LevelInfo, "Content routed successfully",
		"routed_providers_count", len(routing))

	// Normalize content blocks to provider-preferred format
	// Pre-allocate slice with exact capacity to avoid reallocations
	targetFormat := input.Format
	if targetFormat == "" {
		targetFormat = "base64"
	}
	normalizedBlocks := make([]*types.ContentBlock, len(input.ContentBlocks))
	for i, block := range input.ContentBlocks {
		normalized, err := m.normalizer.Normalize(ctx, block, targetFormat)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			logWithOTELContext(ctx, slog.LevelError, "Content normalization failed",
				"error", err, "block_index", i)
			return nil, err
		}
		normalizedBlocks[i] = normalized
	}

	// Generate output using reasoning or generation pipeline
	output, err := m.generateOutput(ctx, input, normalizedBlocks, routing)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Output generation failed", "error", err)
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Multimodal processing completed successfully",
		"output_id", output.ID,
		"content_blocks_count", len(output.ContentBlocks))

	return output, nil
}

// generateOutput generates a multimodal output from normalized blocks.
func (m *BaseMultimodalModel) generateOutput(ctx context.Context, input *types.MultimodalInput, normalizedBlocks []*types.ContentBlock, routing map[string]string) (*types.MultimodalOutput, error) {
	// Determine if this is a reasoning task (input has multimodal content) or generation task (text-only input)
	// Early exit optimization: check first non-text block
	hasMultimodalContent := false
	for i := range normalizedBlocks {
		if normalizedBlocks[i].Type != "text" {
			hasMultimodalContent = true
			break
		}
	}

	var output *types.MultimodalOutput
	var err error

	if hasMultimodalContent {
		// Reasoning pipeline: process multimodal inputs and generate reasoning outputs
		output, err = m.reasoningPipeline(ctx, input, normalizedBlocks, routing)
	} else {
		// Generation pipeline: generate multimodal outputs from text instructions
		output, err = m.generationPipeline(ctx, input, normalizedBlocks)
	}

	if err != nil {
		return nil, err
	}

	// Format output for subsequent operations
	return m.formatOutput(ctx, output), nil
}

// reasoningPipeline processes multimodal inputs and generates reasoning outputs.
func (m *BaseMultimodalModel) reasoningPipeline(ctx context.Context, input *types.MultimodalInput, blocks []*types.ContentBlock, routing map[string]string) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.reasoningPipeline",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.Int("blocks_count", len(blocks)),
		))
	defer span.End()

	// Metrics not available to avoid import cycles

	// Convert content blocks to schema messages for LLM processing
	messages, err := m.contentBlocksToMessages(ctx, blocks)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("reasoningPipeline: %w", err)
	}

	// Use LLM for reasoning if available
	if m.llmProvider != nil {
		response, err := m.llmProvider.GenerateMessages(ctx, messages)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("reasoningPipeline: %w", err)
		}

		// Convert response to output
		// Pre-allocate with exact capacity
		outputBlocks := make([]*types.ContentBlock, len(response))
		for i, msg := range response {
			block, err := types.NewContentBlock("text", []byte(msg.GetContent()))
			if err != nil {
				return nil, err
			}
			outputBlocks[i] = block
		}

		return &types.MultimodalOutput{
			ID:            uuid.New().String(),
			InputID:       input.ID,
			ContentBlocks: outputBlocks,
			Metadata:      make(map[string]any),
			Confidence:    0.95,
			Provider:      m.providerName,
			Model:         m.modelName,
			CreatedAt:     time.Now(),
		}, nil
	}

	// Fallback: return input blocks as output (no processing)
	return &types.MultimodalOutput{
		ID:            uuid.New().String(),
		InputID:       input.ID,
		ContentBlocks: blocks,
		Metadata:      make(map[string]any),
		Confidence:    0.90,
		Provider:      m.providerName,
		Model:         m.modelName,
		CreatedAt:     time.Now(),
	}, nil
}

// generationPipeline generates multimodal outputs from text instructions.
func (m *BaseMultimodalModel) generationPipeline(ctx context.Context, input *types.MultimodalInput, blocks []*types.ContentBlock) (*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.generationPipeline",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
		))
	defer span.End()

	// Metrics not available to avoid import cycles

	// Extract text instruction
	var textInstruction string
	for _, block := range blocks {
		if block.Type == "text" {
			textInstruction = string(block.Data)
			break
		}
	}

	if textInstruction == "" {
		return nil, errors.New("generationPipeline: text instruction required for generation")
	}

	// For now, return a simple text response
	// In a full implementation, this would generate images, audio, or video based on the instruction
	outputBlock, err := types.NewContentBlock("text", []byte("Generated response for: "+textInstruction))
	if err != nil {
		return nil, err
	}

	return &types.MultimodalOutput{
		ID:            uuid.New().String(),
		InputID:       input.ID,
		ContentBlocks: []*types.ContentBlock{outputBlock},
		Metadata:      make(map[string]any),
		Confidence:    0.90,
		Provider:      m.providerName,
		Model:         m.modelName,
		CreatedAt:     time.Now(),
	}, nil
}

// contentBlocksToMessages converts content blocks to schema messages.
func (m *BaseMultimodalModel) contentBlocksToMessages(ctx context.Context, blocks []*types.ContentBlock) ([]schema.Message, error) {
	messages := make([]schema.Message, 0, len(blocks))

	for _, block := range blocks {
		switch block.Type {
		case "text":
			msg := schema.NewHumanMessage(string(block.Data))
			messages = append(messages, msg)
		case "image":
			// Create ImageMessage if available, otherwise use text with metadata
			msg := schema.NewHumanMessage("")
			// In a full implementation, would use schema.ImageMessage
			messages = append(messages, msg)
		case "audio", "video":
			// Similar handling for audio/video
			msg := schema.NewHumanMessage("")
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// formatOutput formats the output for subsequent operations.
func (m *BaseMultimodalModel) formatOutput(ctx context.Context, output *types.MultimodalOutput) *types.MultimodalOutput {
	// Ensure all content blocks are properly formatted
	for _, block := range output.ContentBlocks {
		// Validate and normalize each block
		if err := block.Validate(); err != nil {
			// Log but don't fail - best effort formatting
			logWithOTELContext(ctx, slog.LevelWarn, "Content block validation failed during formatting",
				"error", err, "block_type", block.Type)
		}
	}

	return output
}

// Multimodal RAG Integration Methods

// EmbedMultimodalDocuments generates embeddings for multimodal documents (text+images).
func (m *BaseMultimodalModel) EmbedMultimodalDocuments(ctx context.Context, documents []schema.Document) ([][]float32, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.EmbedMultimodalDocuments",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.Int("documents_count", len(documents)),
		))
	defer span.End()

	if m.multimodalEmbedder != nil && m.multimodalEmbedder.SupportsMultimodal() {
		embeddings, err := m.multimodalEmbedder.EmbedDocumentsMultimodal(ctx, documents)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("EmbedMultimodalDocuments: %w", err)
		}
		span.SetStatus(codes.Ok, "")
		return embeddings, nil
	}

	// Fallback to text-only embedding
	if m.embedder != nil {
		texts := make([]string, len(documents))
		for i, doc := range documents {
			texts[i] = doc.GetContent()
		}
		embeddings, err := m.embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("EmbedMultimodalDocuments: %w", err)
		}
		span.SetStatus(codes.Ok, "")
		return embeddings, nil
	}

	return nil, errors.New("EmbedMultimodalDocuments: no embedder available")
}

// EmbedMultimodalQuery generates an embedding for a multimodal query (text+image).
func (m *BaseMultimodalModel) EmbedMultimodalQuery(ctx context.Context, document schema.Document) ([]float32, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.EmbedMultimodalQuery",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
		))
	defer span.End()

	if m.multimodalEmbedder != nil && m.multimodalEmbedder.SupportsMultimodal() {
		embedding, err := m.multimodalEmbedder.EmbedQueryMultimodal(ctx, document)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("EmbedMultimodalQuery: %w", err)
		}
		span.SetStatus(codes.Ok, "")
		return embedding, nil
	}

	// Fallback to text-only embedding
	if m.embedder != nil {
		embedding, err := m.embedder.EmbedQuery(ctx, document.GetContent())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("EmbedMultimodalQuery: %w", err)
		}
		span.SetStatus(codes.Ok, "")
		return embedding, nil
	}

	return nil, errors.New("EmbedMultimodalQuery: no embedder available")
}

// StoreMultimodalDocuments stores multimodal documents in a vector store.
func (m *BaseMultimodalModel) StoreMultimodalDocuments(ctx context.Context, documents []schema.Document) ([]string, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.StoreMultimodalDocuments",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.Int("documents_count", len(documents)),
		))
	defer span.End()

	if m.vectorStore == nil {
		return nil, errors.New("StoreMultimodalDocuments: vector store not configured")
	}

	// Use multimodal embedder if available, otherwise use regular embedder
	var embedder vectorstores.Embedder
	if m.multimodalEmbedder != nil && m.multimodalEmbedder.SupportsMultimodal() {
		// Create a wrapper to convert MultimodalEmbedder to vectorstores.Embedder
		embedder = &multimodalEmbedderWrapper{embedder: m.multimodalEmbedder, documents: documents}
	} else if m.embedder != nil {
		embedder = &embedderWrapper{embedder: m.embedder}
	}

	// Use vectorstores options
	opts := []vectorstores.Option{}
	if embedder != nil {
		opts = append(opts, vectorstores.WithEmbedder(embedder))
	}
	ids, err := m.vectorStore.AddDocuments(ctx, documents, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("StoreMultimodalDocuments: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Multimodal documents stored successfully",
		"documents_count", len(documents),
		"ids_count", len(ids))

	return ids, nil
}

// SearchMultimodal performs multimodal retrieval with a multimodal query.
func (m *BaseMultimodalModel) SearchMultimodal(ctx context.Context, queryDocument schema.Document, k int) ([]schema.Document, []float32, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.SearchMultimodal",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.Int("k", k),
		))
	defer span.End()

	if m.vectorStore == nil {
		return nil, nil, errors.New("SearchMultimodal: vector store not configured")
	}

	// Generate query embedding
	queryEmbedding, err := m.EmbedMultimodalQuery(ctx, queryDocument)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, err
	}

	// Perform similarity search
	documents, scores, err := m.vectorStore.SimilaritySearch(ctx, queryEmbedding, k)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, nil, fmt.Errorf("SearchMultimodal: %w", err)
	}

	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Multimodal search completed",
		"results_count", len(documents))

	return documents, scores, nil
}

// FuseMultimodalContent fuses multimodal and text content for agent reasoning.
func (m *BaseMultimodalModel) FuseMultimodalContent(ctx context.Context, textContent string, multimodalDocuments []schema.Document) (string, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.FuseMultimodalContent",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.Int("multimodal_docs_count", len(multimodalDocuments)),
		))
	defer span.End()

	// Build fused content by combining text with multimodal document references
	fusedContent := textContent
	if len(multimodalDocuments) > 0 {
		fusedContent += "\n\nRetrieved multimodal context:\n"
		var fusedContentSb499 strings.Builder
		for i, doc := range multimodalDocuments {
			fusedContentSb499.WriteString(fmt.Sprintf("[%d] %s\n", i+1, doc.GetContent()))
			// Include metadata about multimodal content
			if imageURL, ok := doc.Metadata["image_url"]; ok {
				fusedContentSb499.WriteString(fmt.Sprintf("  Image: %v\n", imageURL))
			}
			if _, ok := doc.Metadata["image_base64"]; ok {
				fusedContentSb499.WriteString("  Image: [base64 encoded]\n")
			}
		}
		fusedContent += fusedContentSb499.String()
	}

	span.SetStatus(codes.Ok, "")
	return fusedContent, nil
}

// PreserveContext maintains context across text and multimodal modalities in RAG workflows.
func (m *BaseMultimodalModel) PreserveContext(ctx context.Context, previousContext map[string]any, newInput *types.MultimodalInput) map[string]any {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.PreserveContext",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
		))
	defer span.End()

	// Merge previous context with new input metadata
	preservedContext := make(map[string]any)
	if previousContext != nil {
		for k, v := range previousContext {
			preservedContext[k] = v
		}
	}

	if newInput != nil && newInput.Metadata != nil {
		for k, v := range newInput.Metadata {
			preservedContext[k] = v
		}
		// Preserve content block types for context
		contentTypes := make([]string, len(newInput.ContentBlocks))
		for i, block := range newInput.ContentBlocks {
			contentTypes[i] = block.Type
		}
		preservedContext["content_types"] = contentTypes
		preservedContext["input_id"] = newInput.ID
	}

	span.SetStatus(codes.Ok, "")
	return preservedContext
}

// Helper types for embedding wrapper

// multimodalEmbedderWrapper wraps MultimodalEmbedder to work with vectorstores.Embedder interface.
type multimodalEmbedderWrapper struct {
	embedder  embeddingsiface.MultimodalEmbedder
	documents []schema.Document
}

func (w *multimodalEmbedderWrapper) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	// Convert texts to documents (simplified - in production would preserve multimodal metadata)
	docs := make([]schema.Document, len(texts))
	for i, text := range texts {
		docs[i] = schema.NewDocument(text, nil)
	}
	return w.embedder.EmbedDocumentsMultimodal(ctx, docs)
}

func (w *multimodalEmbedderWrapper) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	doc := schema.NewDocument(text, nil)
	return w.embedder.EmbedQueryMultimodal(ctx, doc)
}

// embedderWrapper wraps regular Embedder to work with vectorstores.Embedder interface.
type embedderWrapper struct {
	embedder embeddingsiface.Embedder
}

func (w *embedderWrapper) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	return w.embedder.EmbedDocuments(ctx, texts)
}

func (w *embedderWrapper) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return w.embedder.EmbedQuery(ctx, text)
}

// ProcessChain processes a chain of multimodal operations (e.g., image → text → image).
func (m *BaseMultimodalModel) ProcessChain(ctx context.Context, inputs []*types.MultimodalInput) ([]*types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.ProcessChain",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.Int("chain_length", len(inputs)),
		))
	defer span.End()

	// Metrics not available to avoid import cycles
	startTime := time.Now()
	_ = startTime // Suppress unused variable warning

	if len(inputs) == 0 {
		return nil, errors.New("ProcessChain: chain must have at least one input")
	}

	outputs := make([]*types.MultimodalOutput, 0, len(inputs))
	var lastOutput *types.MultimodalOutput

	for i, input := range inputs {
		// If this is not the first input, use the previous output as context
		if i > 0 && lastOutput != nil {
			// Merge previous output into current input metadata
			if input.Metadata == nil {
				input.Metadata = make(map[string]any)
			}
			input.Metadata["previous_output"] = lastOutput
		}

		output, err := m.Process(ctx, input)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			// Metrics not available
			logWithOTELContext(ctx, slog.LevelError, "Chain processing failed",
				"error", err, "chain_index", i)
			return outputs, err
		}

		outputs = append(outputs, output)
		lastOutput = output
	}

	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Multimodal chain processing completed",
		"chain_length", len(inputs),
		"outputs_count", len(outputs))

	return outputs, nil
}

// ProcessStream processes a multimodal input and streams results incrementally.
// Supports chunking for video (1MB) and audio (64KB) with low latency for interactive workflows.
func (m *BaseMultimodalModel) ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.ProcessStream",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.Int("content_blocks_count", len(input.ContentBlocks)),
			attribute.Bool("streaming", true),
		))
	defer span.End()

	// Metrics not available to avoid import cycles

	// Validate input - basic validation
	if len(input.ContentBlocks) == 0 {
		err := errors.New("input must have at least one content block")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Check for interruption - cancel any existing stream for this input ID
	m.handleInterruption(ctx, input.ID)

	// Create cancellable context for this stream
	streamCtx, cancel := context.WithCancel(ctx)
	m.streamingState.mu.Lock()
	m.streamingState.activeStreams[input.ID] = cancel
	m.streamingState.chunkBuffers[input.ID] = make([]*types.ContentBlock, 0)
	m.streamingState.mu.Unlock()

	ch := make(chan *types.MultimodalOutput, 10) // Buffered channel for better throughput

	go func() {
		defer func() {
			close(ch)
			// Metrics not available

			// Clean up streaming state
			m.streamingState.mu.Lock()
			delete(m.streamingState.activeStreams, input.ID)
			delete(m.streamingState.chunkBuffers, input.ID)
			m.streamingState.mu.Unlock()

			// Latency metrics would be recorded here if metrics were available
		}()

		// Chunk content blocks for streaming
		chunkedBlocks, err := m.chunkContentBlocks(streamCtx, input.ContentBlocks)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			// Metrics not available
			logWithOTELContext(ctx, slog.LevelError, "Content chunking failed", "error", err)
			return
		}

		span.SetAttributes(attribute.Int("chunks_count", len(chunkedBlocks)))

		// Process chunks incrementally
		for i, chunkBlock := range chunkedBlocks {
			select {
			case <-streamCtx.Done():
				span.SetStatus(codes.Error, "stream canceled")
				logWithOTELContext(ctx, slog.LevelWarn, "Stream canceled", "error", streamCtx.Err())
				return
			default:
				// Process chunk
				chunkInput := &types.MultimodalInput{
					ID:            fmt.Sprintf("%s-chunk-%d", input.ID, i),
					ContentBlocks: []*types.ContentBlock{chunkBlock},
					Metadata:      input.Metadata,
					Format:        input.Format,
					Routing:       input.Routing,
					CreatedAt:     time.Now(),
				}

				chunkOutput, err := m.Process(streamCtx, chunkInput)
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					// Metrics not available
					logWithOTELContext(ctx, slog.LevelError, "Chunk processing failed",
						"error", err, "chunk_index", i)
					// Continue with next chunk instead of failing completely
					continue
				}

				// Send incremental result
				select {
				case ch <- chunkOutput:
					span.SetAttributes(attribute.Int("chunk_index", i))
					modality := chunkBlock.Type
					if modality == "" {
						modality = "unknown"
					}
					logWithOTELContext(ctx, slog.LevelInfo, "Streaming chunk sent",
						"chunk_index", i,
						"modality", modality)
				case <-streamCtx.Done():
					logWithOTELContext(ctx, slog.LevelWarn, "Stream canceled during send", "error", streamCtx.Err())
					return
				}
			}
		}

		span.SetStatus(codes.Ok, "")
		logWithOTELContext(ctx, slog.LevelInfo, "Streaming completed successfully",
			"input_id", input.ID,
			"chunks_processed", len(chunkedBlocks))
	}()

	return ch, nil
}

// chunkContentBlocks chunks content blocks for streaming (video: 1MB, audio: 64KB).
func (m *BaseMultimodalModel) chunkContentBlocks(ctx context.Context, blocks []*types.ContentBlock) ([]*types.ContentBlock, error) {
	chunks := make([]*types.ContentBlock, 0)

	for _, block := range blocks {
		switch block.Type {
		case "video":
			videoChunks, err := m.chunkVideo(ctx, block)
			if err != nil {
				return nil, fmt.Errorf("chunkContentBlocks: %w", err)
			}
			chunks = append(chunks, videoChunks...)
		case "audio":
			audioChunks, err := m.chunkAudio(ctx, block)
			if err != nil {
				return nil, fmt.Errorf("chunkContentBlocks: %w", err)
			}
			chunks = append(chunks, audioChunks...)
		default:
			// Text and images don't need chunking
			chunks = append(chunks, block)
		}
	}

	return chunks, nil
}

// chunkVideo chunks video content into 1MB chunks.
func (m *BaseMultimodalModel) chunkVideo(ctx context.Context, block *types.ContentBlock) ([]*types.ContentBlock, error) {
	const videoChunkSize = 1024 * 1024 // 1MB

	if len(block.Data) == 0 {
		// If data is not loaded, return block as-is
		return []*types.ContentBlock{block}, nil
	}

	if int64(len(block.Data)) <= videoChunkSize {
		return []*types.ContentBlock{block}, nil
	}

	chunks := make([]*types.ContentBlock, 0)
	totalSize := int64(len(block.Data))

	for offset := int64(0); offset < totalSize; offset += videoChunkSize {
		end := offset + videoChunkSize
		if end > totalSize {
			end = totalSize
		}

		chunkData := block.Data[offset:end]
		chunk, err := types.NewContentBlock("video", chunkData)
		if err != nil {
			return nil, err
		}
		chunk.Metadata = map[string]any{
			"chunk_offset": offset,
			"chunk_size":   len(chunkData),
			"total_size":   totalSize,
		}
		if err != nil {
			return nil, err
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// chunkAudio chunks audio content into 64KB chunks.
func (m *BaseMultimodalModel) chunkAudio(ctx context.Context, block *types.ContentBlock) ([]*types.ContentBlock, error) {
	const audioChunkSize = 64 * 1024 // 64KB

	if len(block.Data) == 0 {
		// If data is not loaded, return block as-is
		return []*types.ContentBlock{block}, nil
	}

	if int64(len(block.Data)) <= audioChunkSize {
		return []*types.ContentBlock{block}, nil
	}

	chunks := make([]*types.ContentBlock, 0)
	totalSize := int64(len(block.Data))

	for offset := int64(0); offset < totalSize; offset += audioChunkSize {
		end := offset + audioChunkSize
		if end > totalSize {
			end = totalSize
		}

		chunkData := block.Data[offset:end]
		chunk, err := types.NewContentBlock("audio", chunkData)
		if err != nil {
			return nil, err
		}
		chunk.Metadata = map[string]any{
			"chunk_offset": offset,
			"chunk_size":   len(chunkData),
			"total_size":   totalSize,
		}
		if err != nil {
			return nil, err
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// handleInterruption cancels any existing stream for the given input ID.
func (m *BaseMultimodalModel) handleInterruption(ctx context.Context, inputID string) {
	m.streamingState.mu.Lock()
	defer m.streamingState.mu.Unlock()

	if cancel, exists := m.streamingState.activeStreams[inputID]; exists {
		cancel()
		delete(m.streamingState.activeStreams, inputID)
		delete(m.streamingState.chunkBuffers, inputID)
		logWithOTELContext(ctx, slog.LevelInfo, "Interrupted existing stream",
			"input_id", inputID)
	}
}

// GetCapabilities returns the capabilities of this model.
func (m *BaseMultimodalModel) GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.GetCapabilities",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
		))
	defer span.End()

	// Metrics not available to avoid import cycles
	// Metrics not available

	if m.capabilities == nil {
		// Return default capabilities
		return &types.ModalityCapabilities{
			Text:  true,
			Image: false,
			Audio: false,
			Video: false,
		}, nil
	}

	span.SetStatus(codes.Ok, "")
	return m.capabilities, nil
}

// SupportsModality checks if this model supports a specific modality.
func (m *BaseMultimodalModel) SupportsModality(ctx context.Context, modality string) (bool, error) {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.SupportsModality",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
			attribute.String("modality", modality),
		))
	defer span.End()

	capabilities, err := m.GetCapabilities(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}

	var supported bool
	switch modality {
	case "text":
		supported = capabilities.Text
	case "image":
		supported = capabilities.Image
	case "audio":
		supported = capabilities.Audio
	case "video":
		supported = capabilities.Video
	default:
		supported = false
	}

	// Metrics not available to avoid import cycles

	span.SetStatus(codes.Ok, "")
	return supported, nil
}

// CheckHealth performs a health check and returns an error if the model is unhealthy.
func (m *BaseMultimodalModel) CheckHealth(ctx context.Context) error {
	tracer := otel.Tracer("github.com/lookatitude/beluga-ai/pkg/multimodal/internal")
	ctx, span := tracer.Start(ctx, "multimodal.CheckHealth",
		trace.WithAttributes(
			attribute.String("provider", m.providerName),
			attribute.String("model", m.modelName),
		))
	defer span.End()

	// Basic health check: verify capabilities can be retrieved
	_, err := m.GetCapabilities(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Health check failed: cannot get capabilities", "error", err)
		return fmt.Errorf("health check failed: %w", err)
	}

	// Verify router is available
	if m.router == nil {
		err := errors.New("router not initialized")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Health check failed: router not initialized")
		return err
	}

	// Verify normalizer is available
	if m.normalizer == nil {
		err := errors.New("normalizer not initialized")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logWithOTELContext(ctx, slog.LevelError, "Health check failed: normalizer not initialized")
		return err
	}

	span.SetStatus(codes.Ok, "")
	logWithOTELContext(ctx, slog.LevelInfo, "Health check passed",
		"provider", m.providerName, "model", m.modelName)
	return nil
}

// logWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}
