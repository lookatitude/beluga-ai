package associative

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/schema"
)

// Compile-time check that AssociativeMemory implements memory.Memory.
var _ memory.Memory = (*AssociativeMemory)(nil)

// options holds the configuration for AssociativeMemory.
type options struct {
	model                 llm.ChatModel
	store                 NoteStore
	linkCandidates        int
	retroactiveRefinement bool
	retroactiveMaxWorkers int
	maxTags               int
	hooks                 Hooks
}

func defaults() options {
	return options{
		linkCandidates:        10,
		retroactiveRefinement: false,
		retroactiveMaxWorkers: 4,
		maxTags:               8,
	}
}

// Option configures an AssociativeMemory.
type Option func(*options)

// WithLLM sets the language model used for note enrichment and retroactive
// refinement. If not set, notes are stored without enrichment.
func WithLLM(model llm.ChatModel) Option {
	return func(o *options) { o.model = model }
}

// WithStore sets a custom NoteStore. If not set, an InMemoryNoteStore is used.
func WithStore(store NoteStore) Option {
	return func(o *options) { o.store = store }
}

// WithLinkCandidates sets the maximum number of similar notes to link to
// each new note. Default is 10.
func WithLinkCandidates(k int) Option {
	return func(o *options) {
		if k > 0 {
			o.linkCandidates = k
		}
	}
}

// WithRetroactiveRefinement enables or disables retroactive updating of
// neighbor notes when a new note is added. Default is false.
func WithRetroactiveRefinement(enabled bool) Option {
	return func(o *options) { o.retroactiveRefinement = enabled }
}

// WithRetroactiveMaxWorkers sets the maximum number of concurrent workers
// for retroactive refinement. Default is 4.
func WithRetroactiveMaxWorkers(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.retroactiveMaxWorkers = n
		}
	}
}

// WithMaxTags sets the maximum number of tags the enricher will produce
// per note. Default is 8.
func WithMaxTags(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.maxTags = n
		}
	}
}

// WithHooks sets lifecycle hooks for the associative memory.
func WithHooks(hooks Hooks) Option {
	return func(o *options) { o.hooks = hooks }
}

// AssociativeMemory implements a Zettelkasten-style associative memory system.
// It orchestrates enrichment, embedding, linking, and optional retroactive
// refinement when notes are added. It implements the memory.Memory interface
// for integration with the composite memory system.
type AssociativeMemory struct {
	embedder embedding.Embedder
	store    NoteStore
	enricher *NoteEnricher
	linker   *LinkManager
	updater  *RetroactiveUpdater
	opts     options
}

// NewAssociativeMemory creates a new AssociativeMemory with the given embedder
// and options. The embedder is required; all other dependencies are optional.
func NewAssociativeMemory(embedder embedding.Embedder, opts ...Option) (*AssociativeMemory, error) {
	if embedder == nil {
		return nil, core.NewError("associative.new", core.ErrInvalidInput, "embedder is required", nil)
	}

	o := defaults()
	for _, opt := range opts {
		opt(&o)
	}

	store := o.store
	if store == nil {
		store = NewInMemoryNoteStore()
	}

	enricher := NewNoteEnricher(o.model, o.maxTags)
	linker := NewLinkManager(store, o.linkCandidates)

	var updater *RetroactiveUpdater
	if o.retroactiveRefinement {
		updater = NewRetroactiveUpdater(store, enricher, embedder, o.retroactiveMaxWorkers)
	}

	return &AssociativeMemory{
		embedder: embedder,
		store:    store,
		enricher: enricher,
		linker:   linker,
		updater:  updater,
		opts:     o,
	}, nil
}

// AddNote creates a new note from the given content. It runs the full pipeline:
// enrich -> embed -> store -> link -> optionally refine neighbors.
// Returns the fully populated note.
func (am *AssociativeMemory) AddNote(ctx context.Context, content string) (*schema.Note, error) {
	if content == "" {
		return nil, core.NewError("associative.add_note", core.ErrInvalidInput, "content is empty", nil)
	}

	now := time.Now().UTC()
	note := &schema.Note{
		ID:        uuid.New().String(),
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Step 1: Enrich via LLM.
	enrichment, err := am.enricher.Enrich(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("associative.add_note: enrich: %w", err)
	}
	note.Keywords = enrichment.Keywords
	note.Tags = enrichment.Tags
	note.Description = enrichment.Description

	// Step 2: Embed.
	vec, err := am.embedder.EmbedSingle(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("associative.add_note: embed: %w", err)
	}
	note.Embedding = vec

	// Step 3: Store.
	if err := am.store.Add(ctx, note); err != nil {
		return nil, fmt.Errorf("associative.add_note: store: %w", err)
	}

	// Fire OnNoteCreated hook.
	if am.opts.hooks.OnNoteCreated != nil {
		am.opts.hooks.OnNoteCreated(ctx, note)
	}

	// Step 4: Link to similar notes.
	linkedIDs, err := am.linker.Link(ctx, note)
	if err != nil {
		return note, fmt.Errorf("associative.add_note: link: %w", err)
	}

	if len(linkedIDs) > 0 && am.opts.hooks.OnNoteLinked != nil {
		am.opts.hooks.OnNoteLinked(ctx, note, linkedIDs)
	}

	// Step 5: Optional retroactive refinement.
	if am.updater != nil && len(linkedIDs) > 0 {
		refinedIDs, err := am.updater.Update(ctx, note, linkedIDs)
		if err != nil {
			// Refinement errors are non-fatal; the note was already stored and linked.
			// Log so operators can observe partial failures in retroactive refinement.
			slog.WarnContext(ctx, "associative: retroactive refinement failed",
				slog.String("note_id", note.ID),
				slog.Any("error", err),
			)
		}
		if len(refinedIDs) > 0 && am.opts.hooks.OnNoteRefined != nil {
			am.opts.hooks.OnNoteRefined(ctx, note, refinedIDs)
		}
	}

	return note, nil
}

// GetNote retrieves a note by ID.
func (am *AssociativeMemory) GetNote(ctx context.Context, id string) (*schema.Note, error) {
	return am.store.Get(ctx, id)
}

// SearchNotes finds notes semantically similar to the query, returning at most k results.
func (am *AssociativeMemory) SearchNotes(ctx context.Context, query string, k int) ([]*schema.Note, error) {
	if k <= 0 {
		k = 10
	}
	vec, err := am.embedder.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("associative.search_notes: embed query: %w", err)
	}
	return am.store.Search(ctx, vec, k)
}

// DeleteNote removes a note by ID and cleans up any links from other notes
// that reference it.
func (am *AssociativeMemory) DeleteNote(ctx context.Context, id string) error {
	// Get the note to find its links before deletion.
	note, err := am.store.Get(ctx, id)
	if err != nil {
		return err
	}

	// Remove backlinks from neighbors.
	for _, linkedID := range note.Links {
		neighbor, err := am.store.Get(ctx, linkedID)
		if err != nil {
			continue // Neighbor may already be deleted.
		}
		neighbor.Links = removeString(neighbor.Links, id)
		if err := am.store.Update(ctx, neighbor); err != nil {
			// Log but continue: cleanup is best-effort to avoid dangling backlinks.
			slog.WarnContext(ctx, "associative: backlink cleanup failed",
				slog.String("note_id", id),
				slog.String("neighbor_id", linkedID),
				slog.Any("error", err),
			)
		}
	}

	return am.store.Delete(ctx, id)
}

// Save implements memory.Memory. It creates a note from the output message
// content, using the input message content as additional context.
func (am *AssociativeMemory) Save(ctx context.Context, input, output schema.Message) error {
	text := extractMessageText(output)
	if text == "" {
		return nil
	}

	inputText := extractMessageText(input)
	content := text
	if inputText != "" {
		content = fmt.Sprintf("Q: %s\nA: %s", inputText, text)
	}

	_, err := am.AddNote(ctx, content)
	return err
}

// Load implements memory.Memory. It searches for notes relevant to the query
// and returns their content as HumanMessage/AIMessage pairs.
func (am *AssociativeMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	notes, err := am.SearchNotes(ctx, query, 10)
	if err != nil {
		return nil, err
	}

	var msgs []schema.Message
	for _, note := range notes {
		msgs = append(msgs, schema.NewAIMessage(note.Content))
	}
	return msgs, nil
}

// Search implements memory.Memory. It searches for notes relevant to the query
// and returns them as Documents.
func (am *AssociativeMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	notes, err := am.SearchNotes(ctx, query, k)
	if err != nil {
		return nil, err
	}

	docs := make([]schema.Document, len(notes))
	for i, note := range notes {
		docs[i] = schema.Document{
			ID:      note.ID,
			Content: note.Content,
			Metadata: map[string]any{
				"keywords":    note.Keywords,
				"tags":        note.Tags,
				"description": note.Description,
				"links":       note.Links,
				"created_at":  note.CreatedAt,
			},
			Embedding: note.Embedding,
		}
	}
	return docs, nil
}

// Clear implements memory.Memory. It removes all notes from the store.
func (am *AssociativeMemory) Clear(ctx context.Context) error {
	notes, err := am.store.List(ctx)
	if err != nil {
		return fmt.Errorf("associative.clear: list: %w", err)
	}
	for _, note := range notes {
		if err := am.store.Delete(ctx, note.ID); err != nil {
			return fmt.Errorf("associative.clear: delete %q: %w", note.ID, err)
		}
	}
	return nil
}

// extractMessageText extracts concatenated text from a message's content parts.
func extractMessageText(msg schema.Message) string {
	var b strings.Builder
	for i, p := range msg.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			if i > 0 && b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(tp.Text)
		}
	}
	return b.String()
}

// removeString returns a new slice with all occurrences of s removed.
func removeString(ss []string, s string) []string {
	out := make([]string, 0, len(ss))
	for _, v := range ss {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}
