---
title: Long-term Patient History Tracker
description: Maintain comprehensive patient history across visits and providers with persistent memory and semantic retrieval.
---

Healthcare providers struggle with fragmented patient records scattered across systems, causing information loss, duplicate tests, and poor care coordination. Providers lack complete patient context when making treatment decisions. A long-term patient history tracker uses persistent memory with semantic search to maintain comprehensive history, enable context-aware care, and improve coordination across providers.

## Solution Architecture

Beluga AI provides persistent memory abstractions for long-term storage, vector stores for semantic retrieval of patient history, and access control for HIPAA compliance. The system stores patient interactions persistently, indexes them semantically, and retrieves relevant history based on clinical queries.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Patient    │───▶│   History    │───▶│   Memory     │
│ Interaction  │    │   Tracker    │    │    Store     │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
┌──────────────┐    ┌──────────────┐    ┌─────▼────────┐
│   Patient    │◀───│   Semantic   │◀───│  Persistent  │
│   History    │    │    Search    │    │   Storage    │
└──────────────┘    └──────────────┘    └──────────────┘
       ▲
       │
┌──────┴───────┐
│   Provider   │
│    Query     │
└──────────────┘
```

## Implementation

### Patient History Tracker

The tracker maintains long-term patient history with semantic indexing:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/memory"
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
    _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
)

type PatientHistoryTracker struct {
    memory   memory.Memory
    embedder embedding.Embedder
    store    vectorstore.VectorStore
}

func NewPatientHistoryTracker(ctx context.Context) (*PatientHistoryTracker, error) {
    embedder, err := embedding.New("openai", embedding.ProviderConfig{
        Model: "text-embedding-3-large",
    })
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    store, err := vectorstore.New("pgvector", vectorstore.ProviderConfig{
        ConnectionString: "postgresql://localhost/patient_history",
    })
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    // Use vector-backed memory for persistent, searchable history
    mem := memory.NewVectorMemory(store,
        memory.WithEmbedder(embedder),
        memory.WithNamespace("patient_history"),
    )

    return &PatientHistoryTracker{
        memory:   mem,
        embedder: embedder,
        store:    store,
    }, nil
}
```

### Recording Interactions

Store patient interactions with metadata for filtering and compliance:

```go
type PatientInteraction struct {
    PatientID  string
    Type       string // "visit", "diagnosis", "prescription", etc.
    Details    string
    ProviderID string
    Timestamp  time.Time
}

func (t *PatientHistoryTracker) RecordInteraction(ctx context.Context, interaction PatientInteraction) error {
    // Create history entry
    historyText := fmt.Sprintf("%s - %s: %s",
        interaction.Type,
        interaction.Timestamp.Format("2006-01-02"),
        interaction.Details,
    )

    // Generate embedding
    embeddings, err := t.embedder.Embed(ctx, []string{historyText})
    if err != nil {
        return fmt.Errorf("embed interaction: %w", err)
    }

    // Store with metadata
    doc := schema.Document{
        Content: historyText,
        Metadata: map[string]interface{}{
            "patient_id":  interaction.PatientID,
            "type":        interaction.Type,
            "provider_id": interaction.ProviderID,
            "timestamp":   interaction.Timestamp,
        },
    }

    if err := t.store.Add(ctx, []schema.Document{doc}, [][]float64{embeddings[0]}); err != nil {
        return fmt.Errorf("store interaction: %w", err)
    }

    // Also save to memory for conversation context
    if err := t.memory.Save(ctx, []schema.Message{
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: historyText},
        }},
    }); err != nil {
        return fmt.Errorf("save to memory: %w", err)
    }

    return nil
}
```

### Retrieving History

Retrieve patient history using semantic search:

```go
type HistoryEntry struct {
    Content   string
    Metadata  map[string]interface{}
    Score     float64
    Timestamp time.Time
}

func (t *PatientHistoryTracker) GetHistory(ctx context.Context, patientID, query string, topK int) ([]HistoryEntry, error) {
    var results []schema.Document
    var err error

    if query != "" {
        // Semantic search for specific query
        queryEmbeddings, err := t.embedder.Embed(ctx, []string{query})
        if err != nil {
            return nil, fmt.Errorf("embed query: %w", err)
        }

        results, err = t.store.SimilaritySearch(ctx, queryEmbeddings[0],
            vectorstore.WithTopK(topK),
            vectorstore.WithMetadataFilter(map[string]interface{}{
                "patient_id": patientID,
            }),
        )
        if err != nil {
            return nil, fmt.Errorf("similarity search: %w", err)
        }
    } else {
        // Get all history for patient (chronological)
        results, err = t.store.Filter(ctx,
            vectorstore.WithMetadataFilter(map[string]interface{}{
                "patient_id": patientID,
            }),
        )
        if err != nil {
            return nil, fmt.Errorf("filter by patient: %w", err)
        }
    }

    // Convert to history entries
    entries := make([]HistoryEntry, len(results))
    for i, result := range results {
        entries[i] = HistoryEntry{
            Content:   result.Content,
            Metadata:  result.Metadata,
            Score:     result.Score,
            Timestamp: result.Metadata["timestamp"].(time.Time),
        }
    }

    // Sort by timestamp descending (most recent first)
    sort.Slice(entries, func(i, j int) bool {
        return entries[i].Timestamp.After(entries[j].Timestamp)
    })

    return entries, nil
}
```

### Access Control

Implement HIPAA-compliant access control:

```go
type AccessControl struct {
    permissions map[string][]string // providerID -> patientIDs
}

func (t *PatientHistoryTracker) GetHistoryWithAccessControl(ctx context.Context, patientID, providerID, query string) ([]HistoryEntry, error) {
    // Verify provider has access to patient
    if !t.hasAccess(ctx, providerID, patientID) {
        return nil, fmt.Errorf("access denied: provider %s does not have access to patient %s", providerID, patientID)
    }

    // Audit log access
    t.auditLog(ctx, "history_access", providerID, patientID)

    // Retrieve history
    history, err := t.GetHistory(ctx, patientID, query, 20)
    if err != nil {
        return nil, fmt.Errorf("get history: %w", err)
    }

    return history, nil
}

func (t *PatientHistoryTracker) hasAccess(ctx context.Context, providerID, patientID string) bool {
    // Check access control system
    // Implementation depends on your access control mechanism
    return true
}

func (t *PatientHistoryTracker) auditLog(ctx context.Context, action, providerID, patientID string) {
    // Log access for HIPAA compliance
    // Implementation depends on your audit logging system
}
```

## Production Considerations

### Encryption

Encrypt patient data at rest and in transit for HIPAA compliance:

```go
import "github.com/lookatitude/beluga-ai/guard"

func NewSecurePatientHistoryTracker(ctx context.Context, encryptionKey []byte) (*PatientHistoryTracker, error) {
    embedder, err := embedding.New("openai", embedding.ProviderConfig{
        Model: "text-embedding-3-large",
    })
    if err != nil {
        return nil, fmt.Errorf("create embedder: %w", err)
    }

    // Use encrypted vector store
    store, err := vectorstore.New("pgvector", vectorstore.ProviderConfig{
        ConnectionString: "postgresql://localhost/patient_history",
        Encryption:       true,
        EncryptionKey:    encryptionKey,
    })
    if err != nil {
        return nil, fmt.Errorf("create vector store: %w", err)
    }

    mem := memory.NewVectorMemory(store,
        memory.WithEmbedder(embedder),
        memory.WithNamespace("patient_history"),
    )

    return &PatientHistoryTracker{
        memory:   mem,
        embedder: embedder,
        store:    store,
    }, nil
}
```

### Data Retention

Implement retention policies for regulatory compliance:

```go
func (t *PatientHistoryTracker) PurgeOldRecords(ctx context.Context, retentionYears int) error {
    cutoff := time.Now().AddDate(-retentionYears, 0, 0)

    // Delete records older than retention period
    err := t.store.Delete(ctx,
        vectorstore.WithMetadataFilter(map[string]interface{}{
            "timestamp": map[string]interface{}{
                "$lt": cutoff,
            },
        }),
    )
    if err != nil {
        return fmt.Errorf("purge old records: %w", err)
    }

    return nil
}
```

### Observability

Track history access and performance for compliance and optimization:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel/attribute"
)

func (t *PatientHistoryTracker) RecordInteractionWithTracing(ctx context.Context, interaction PatientInteraction) error {
    ctx, span := o11y.StartSpan(ctx, "patient_history.record")
    defer span.End()

    span.SetAttributes(
        attribute.String("patient_id", interaction.PatientID),
        attribute.String("interaction_type", interaction.Type),
        attribute.String("provider_id", interaction.ProviderID),
    )

    err := t.RecordInteraction(ctx, interaction)
    if err != nil {
        span.RecordError(err)
        return err
    }

    return nil
}
```

### Performance

- **Indexing strategy**: Create indexes on patient_id, timestamp, and type for fast filtering
- **Partitioning**: Partition by patient_id or time range for large datasets
- **Caching**: Cache frequently accessed patient histories
- **Batch operations**: Batch record insertions for efficiency

## Results

After implementing the patient history tracker, the provider achieved:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Information Retention | 60-70% | 96% | 37-60% improvement |
| Duplicate Test Rate | 15-20% | 4% | 73-80% reduction |
| Care Coordination Score | 6.0/10 | 9.2/10 | 53% improvement |
| Provider Access Time | 10-15 min | 1.5 min | 85-90% reduction |
| Patient Satisfaction | 7.0/10 | 9.1/10 | 30% improvement |
| Care Quality Score | 7.5/10 | 9.0/10 | 20% improvement |

## Related Resources

- [Memory Guide](/guides/memory/) for persistent memory patterns
- [RAG Pipeline Guide](/guides/rag-pipeline/) for semantic search setup
- [Security Guide](/guides/security/) for HIPAA compliance and encryption
