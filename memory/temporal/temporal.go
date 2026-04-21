package temporal

import (
	"context"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/memory"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

const (
	opResolveConflicts  = "temporal.resolve_conflicts"
	entityTypeMessage   = "message"
	propKeyRole         = "role"
	propKeyText         = "text"
)

// TemporalMemory wraps a TemporalGraphStore to provide a Memory-compatible interface
// with bi-temporal knowledge graph capabilities. It supports saving conversation turns
// as graph entities/relations, loading relevant context, and querying the graph as
// it existed at any point in time.
type TemporalMemory struct {
	store    memory.TemporalGraphStore
	resolver ConflictResolver
	hooks    Hooks
}

// New creates a new TemporalMemory backed by the given TemporalGraphStore.
// If no conflict resolver is provided via options, the default TemporalResolver is used.
func New(store memory.TemporalGraphStore, opts ...Option) *TemporalMemory {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return &TemporalMemory{
		store:    store,
		resolver: o.resolver,
		hooks:    o.hooks,
	}
}

// Save persists a conversation turn as entities and relations in the temporal graph.
// The input message is stored as a "message" entity and relations are created to
// represent the conversation flow.
func (tm *TemporalMemory) Save(ctx context.Context, input, output schema.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	now := time.Now()

	// Create entities for input and output messages.
	inputText := extractText(input)
	outputText := extractText(output)

	inputEntity := memory.Entity{
		ID:         fmt.Sprintf("msg-input-%d", now.UnixNano()),
		Type:       entityTypeMessage,
		Properties: map[string]any{propKeyRole: string(input.GetRole()), propKeyText: inputText},
		CreatedAt:  now,
		Summary:    truncate(inputText, 200),
	}

	outputEntity := memory.Entity{
		ID:         fmt.Sprintf("msg-output-%d", now.UnixNano()),
		Type:       entityTypeMessage,
		Properties: map[string]any{propKeyRole: string(output.GetRole()), propKeyText: outputText},
		CreatedAt:  now,
		Summary:    truncate(outputText, 200),
	}

	if err := tm.store.AddEntity(ctx, inputEntity); err != nil {
		return core.Errorf(core.ErrProviderDown, "temporal: save input entity: %w", err)
	}
	if err := tm.store.AddEntity(ctx, outputEntity); err != nil {
		return core.Errorf(core.ErrProviderDown, "temporal: save output entity: %w", err)
	}

	// Create a "responds_to" relation between output and input.
	props := map[string]any{
		"turn_timestamp": now.Format(time.RFC3339Nano),
	}
	if err := tm.store.AddRelation(ctx, outputEntity.ID, inputEntity.ID, "responds_to", props); err != nil {
		return core.Errorf(core.ErrProviderDown, "temporal: save relation: %w", err)
	}

	return nil
}

// Load retrieves messages from the graph by querying for message entities matching
// the given query string. Returns messages ordered by creation time.
func (tm *TemporalMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	results, err := tm.store.Query(ctx, query)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "temporal: load: %w", err)
	}

	var msgs []schema.Message
	for _, result := range results {
		for _, entity := range result.Entities {
			if entity.Type != entityTypeMessage {
				continue
			}
			msg := entityToMessage(entity)
			if msg != nil {
				msgs = append(msgs, msg)
			}
		}
	}

	return msgs, nil
}

// LoadAt retrieves messages that were valid at a specific point in time. This enables
// querying the knowledge graph as it existed at any historical moment.
func (tm *TemporalMemory) LoadAt(ctx context.Context, query string, validTime time.Time) ([]schema.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if validTime.IsZero() {
		return nil, core.NewError("temporal.load_at", core.ErrInvalidInput, "validTime must not be zero", nil)
	}

	entities, _, err := tm.store.QueryAsOf(ctx, query, validTime)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "temporal: load_at: %w", err)
	}

	var msgs []schema.Message
	for _, entity := range entities {
		if entity.Type != entityTypeMessage {
			continue
		}
		msg := entityToMessage(entity)
		if msg != nil {
			msgs = append(msgs, msg)
		}
	}

	return msgs, nil
}

// Search finds documents in the graph relevant to the given query. The k parameter
// limits the number of results returned.
func (tm *TemporalMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if k <= 0 {
		return nil, nil
	}

	results, err := tm.store.Query(ctx, query)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "temporal: search: %w", err)
	}

	var docs []schema.Document
	for _, result := range results {
		for _, entity := range result.Entities {
			if len(docs) >= k {
				break
			}
			docs = append(docs, schema.Document{
				ID:       entity.ID,
				Content:  entity.Summary,
				Metadata: entity.Properties,
			})
		}
	}

	return docs, nil
}

// Clear removes all data from the underlying store by creating a fresh store.
// Note: For the in-memory store, this is handled by replacing the internal state.
// For persistent stores, this delegates to the store's Clear method if available.
func (tm *TemporalMemory) Clear(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// If the store supports clearing, use it.
	type clearer interface {
		Clear(ctx context.Context) error
	}
	if c, ok := tm.store.(clearer); ok {
		return c.Clear(ctx)
	}

	return nil
}

// ResolveConflicts runs the conflict resolution algorithm for a new relation against
// existing relations between the same entities. This should be called when adding
// knowledge that may contradict existing facts.
func (tm *TemporalMemory) ResolveConflicts(ctx context.Context, newRelation *memory.Relation) ([]memory.Relation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if newRelation == nil {
		return nil, core.NewError(opResolveConflicts, core.ErrInvalidInput, "newRelation must not be nil", nil)
	}

	sameType, err := tm.samTypeHistory(ctx, newRelation)
	if err != nil {
		return nil, err
	}

	invalidated, err := tm.resolver.Resolve(ctx, newRelation, sameType)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "temporal: resolve_conflicts: %w", err)
	}

	if err := tm.applyInvalidations(ctx, invalidated); err != nil {
		return nil, err
	}

	if tm.hooks.OnConflictResolved != nil && len(invalidated) > 0 {
		tm.hooks.OnConflictResolved(ctx, invalidated, *newRelation)
	}

	return invalidated, nil
}

// samTypeHistory retrieves the history of relations between the new relation's
// entities and filters to the same relation type.
func (tm *TemporalMemory) samTypeHistory(ctx context.Context, newRelation *memory.Relation) ([]memory.Relation, error) {
	candidates, err := tm.store.History(ctx, newRelation.From, newRelation.To)
	if err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "temporal: resolve_conflicts history: %w", err)
	}
	var sameType []memory.Relation
	for _, c := range candidates {
		if c.Type == newRelation.Type {
			sameType = append(sameType, c)
		}
	}
	return sameType, nil
}

// applyInvalidations applies each invalidated relation to the store.
// Missing or malformed identifiers surface as errors to prevent store desync.
func (tm *TemporalMemory) applyInvalidations(ctx context.Context, invalidated []memory.Relation) error {
	for _, inv := range invalidated {
		relID, err := extractRelationID(inv)
		if err != nil {
			return err
		}
		if inv.InvalidAt == nil {
			continue
		}
		if err := tm.store.InvalidateRelation(ctx, relID, *inv.InvalidAt); err != nil {
			return core.Errorf(core.ErrProviderDown, "temporal: apply invalidation: %w", err)
		}
	}
	return nil
}

// extractRelationID returns the relation ID string from Properties["id"].
func extractRelationID(rel memory.Relation) (string, error) {
	id, ok := rel.Properties["id"]
	if !ok {
		return "", core.NewError(opResolveConflicts, core.ErrInvalidInput, "invalidated relation missing Properties[\"id\"]", nil)
	}
	relID, ok := id.(string)
	if !ok {
		return "", core.NewError(opResolveConflicts, core.ErrInvalidInput, "invalidated relation Properties[\"id\"] is not a string", nil)
	}
	return relID, nil
}

// Store returns the underlying TemporalGraphStore for direct access to graph operations.
func (tm *TemporalMemory) Store() memory.TemporalGraphStore {
	return tm.store
}

// entityToMessage converts a message entity back to a schema.Message.
func entityToMessage(entity memory.Entity) schema.Message {
	text, _ := entity.Properties[propKeyText].(string)
	role, _ := entity.Properties[propKeyRole].(string)

	switch schema.Role(role) {
	case schema.RoleHuman:
		return schema.NewHumanMessage(text)
	case schema.RoleAI:
		return schema.NewAIMessage(text)
	case schema.RoleSystem:
		return schema.NewSystemMessage(text)
	default:
		if text != "" {
			return schema.NewHumanMessage(text)
		}
		return nil
	}
}

// extractText extracts the text content from a message.
func extractText(msg schema.Message) string {
	parts := msg.GetContent()
	for _, p := range parts {
		if tp, ok := p.(schema.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}

// truncate shortens a string to at most maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Compile-time check that TemporalMemory implements memory.Memory.
var _ memory.Memory = (*TemporalMemory)(nil)
