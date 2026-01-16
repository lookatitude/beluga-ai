package twilio

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	multimodaliface "github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
	twiliov2010 "github.com/twilio/twilio-go/rest/api/v2010"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TranscriptionManager manages transcription storage and RAG integration.
type TranscriptionManager struct {
	backend     *TwilioBackend
	vectorStore vectorstoresiface.VectorStore
	embedder    embeddingsiface.Embedder
	retriever   core.Retriever
	multimodal  multimodaliface.MultimodalModel
	metrics     *Metrics
}

// NewTranscriptionManager creates a new transcription manager.
func NewTranscriptionManager(backend *TwilioBackend) *TranscriptionManager {
	return &TranscriptionManager{
		backend: backend,
		// Get vector store, embedder, retriever from config
		vectorStore: backend.config.VectorStore,
		embedder:    backend.config.Embedder,
		multimodal:  backend.config.MultimodalModel,
		metrics:     backend.metrics,
	}
}

// RetrieveTranscription retrieves a transcription from Twilio API.
func (tm *TranscriptionManager) RetrieveTranscription(ctx context.Context, transcriptionSID string) (*Transcription, error) {
	ctx, span := tm.startSpan(ctx, "TranscriptionManager.RetrieveTranscription")
	defer span.End()

	span.SetAttributes(attribute.String("transcription_sid", transcriptionSID))

	// Fetch transcription from Twilio API
	transcription, err := tm.backend.client.Api.FetchTranscription(transcriptionSID, &twiliov2010.FetchTranscriptionParams{})
	if err != nil {
		err = MapTwilioError("RetrieveTranscription", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	result := &Transcription{
		TranscriptionSID: getStringValue(transcription.Sid),
		CallSID:          "", // CallSID not directly available - would need to fetch from Recording
		AccountSID:       getStringValue(transcription.AccountSid),
		Status:           getStringValue(transcription.Status),
		Text:             getStringValue(transcription.TranscriptionText),
		Language:         "",         // Language not directly available in Transcription struct
		DateCreated:      time.Now(), // Twilio returns date as string, would need parsing
		DateUpdated:      time.Now(),
	}

	// Store RecordingSID in metadata for later retrieval of CallSID
	if recordingSID := getStringValue(transcription.RecordingSid); recordingSID != "" {
		result.Metadata = map[string]any{
			"recording_sid": recordingSID,
		}
	}

	span.SetStatus(codes.Ok, "transcription retrieved")
	return result, nil
}

// StoreTranscription stores a transcription in the vector store for RAG.
func (tm *TranscriptionManager) StoreTranscription(ctx context.Context, transcription *Transcription) error {
	ctx, span := tm.startSpan(ctx, "TranscriptionManager.StoreTranscription")
	defer span.End()

	span.SetAttributes(
		attribute.String("transcription_sid", transcription.TranscriptionSID),
		attribute.String("call_sid", transcription.CallSID),
	)

	if tm.vectorStore == nil {
		err := fmt.Errorf("vector store not configured")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return NewTwilioError("StoreTranscription", ErrCodeTwilioTranscriptionError, err)
	}

	// Create document from transcription
	metadata := map[string]string{
		"transcription_sid": transcription.TranscriptionSID,
		"call_sid":          transcription.CallSID,
		"account_sid":       transcription.AccountSID,
		"status":            transcription.Status,
		"language":          transcription.Language,
		"date_created":      transcription.DateCreated.Format(time.RFC3339),
		"date_updated":      transcription.DateUpdated.Format(time.RFC3339),
	}
	doc := schema.NewDocument(transcription.Text, metadata)

	// Store in vector store with embedding
	_, err := tm.vectorStore.AddDocuments(ctx, []schema.Document{doc},
		vectorstoresiface.WithEmbedder(tm.embedder),
	)
	if err != nil {
		err = MapTwilioError("StoreTranscription", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if tm.metrics != nil {
		tm.metrics.RecordTranscription(ctx, transcription.TranscriptionSID, true)
	}

	span.SetStatus(codes.Ok, "transcription stored")
	return nil
}

// SearchTranscriptions searches for relevant transcriptions using semantic search.
func (tm *TranscriptionManager) SearchTranscriptions(ctx context.Context, query string, limit int) ([]*Transcription, error) {
	ctx, span := tm.startSpan(ctx, "TranscriptionManager.SearchTranscriptions")
	defer span.End()

	span.SetAttributes(
		attribute.String("query", query),
		attribute.Int("limit", limit),
	)

	if tm.retriever == nil {
		// Create retriever from vector store if not set
		if tm.vectorStore == nil {
			err := fmt.Errorf("vector store not configured")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, NewTwilioError("SearchTranscriptions", ErrCodeTwilioTranscriptionError, err)
		}

		// In a full implementation, would create retriever from vector store
		err := fmt.Errorf("retriever not configured")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, NewTwilioError("SearchTranscriptions", ErrCodeTwilioTranscriptionError, err)
	}

	// Search using retriever
	documents, err := tm.retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		err = MapTwilioError("SearchTranscriptions", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	// Convert documents to transcriptions
	transcriptions := make([]*Transcription, 0, len(documents))
	for _, doc := range documents {
		transcription := &Transcription{
			Text: doc.GetContent(),
		}

		// Extract metadata
		if metadata := doc.Metadata; metadata != nil {
			if sid, ok := metadata["transcription_sid"]; ok {
				transcription.TranscriptionSID = sid
			}
			if callSID, ok := metadata["call_sid"]; ok {
				transcription.CallSID = callSID
			}
		}

		transcriptions = append(transcriptions, transcription)
	}

	span.SetAttributes(attribute.Int("results_count", len(transcriptions)))
	span.SetStatus(codes.Ok, "transcriptions searched")
	return transcriptions, nil
}

// RetrieveRelevantTranscriptions retrieves relevant transcriptions for agent context (RAG).
func (tm *TranscriptionManager) RetrieveRelevantTranscriptions(ctx context.Context, query string, limit int) ([]*Transcription, error) {
	return tm.SearchTranscriptions(ctx, query, limit)
}

// MultimodalRAGSearch performs multimodal RAG search combining transcriptions with other data sources.
func (tm *TranscriptionManager) MultimodalRAGSearch(ctx context.Context, query string, dataSources []string, limit int) ([]*Transcription, error) {
	ctx, span := tm.startSpan(ctx, "TranscriptionManager.MultimodalRAGSearch")
	defer span.End()

	span.SetAttributes(
		attribute.String("query", query),
		attribute.StringSlice("data_sources", dataSources),
		attribute.Int("limit", limit),
	)

	if tm.multimodal == nil {
		// Fallback to regular search if multimodal not available
		return tm.SearchTranscriptions(ctx, query, limit)
	}

	// In a full implementation, this would:
	// 1. Search transcriptions
	// 2. Search other data sources (documents, images, etc.)
	// 3. Combine results using multimodal model
	// 4. Return ranked results

	// For now, use regular search
	return tm.SearchTranscriptions(ctx, query, limit)
}

// Transcription represents a Twilio transcription.
type Transcription struct {
	TranscriptionSID string
	CallSID          string
	AccountSID       string
	Status           string
	Text             string
	Language         string
	Confidence       float64
	DateCreated      time.Time
	DateUpdated      time.Time
	Duration         int
	Metadata         map[string]any
}

// startSpan starts an OTEL span for tracing.
func (tm *TranscriptionManager) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	if tm.metrics != nil && tm.metrics.Tracer() != nil {
		return tm.metrics.Tracer().Start(ctx, operation)
	}
	return ctx, trace.SpanFromContext(ctx)
}
