package procedural

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// ProceduralMemory is the 4th memory tier in the MemGPT model, providing
// storage and semantic retrieval of procedural knowledge (how-to skills).
// It uses vector embeddings for similarity search over skill descriptions
// and triggers.
type ProceduralMemory struct {
	emb  embedding.Embedder
	vs   vectorstore.VectorStore
	opts options
}

// New creates a new ProceduralMemory with the given embedder, vector store,
// and functional options. Both embedder and vector store must be non-nil.
func New(emb embedding.Embedder, vs vectorstore.VectorStore, opts ...Option) (*ProceduralMemory, error) {
	if emb == nil {
		return nil, fmt.Errorf("procedural: Embedder is required")
	}
	if vs == nil {
		return nil, fmt.Errorf("procedural: VectorStore is required")
	}
	o := defaults()
	for _, opt := range opts {
		opt(&o)
	}
	return &ProceduralMemory{
		emb:  emb,
		vs:   vs,
		opts: o,
	}, nil
}

// SaveSkill persists a skill in the vector store. The skill's description
// and triggers are embedded for later semantic retrieval. The skill is
// serialized as JSON in the document content, and its metadata includes
// the skill ID, agent ID, confidence, and version for filtering.
//
// SaveSkill is intended for first-time persistence of a skill. Calling
// SaveSkill repeatedly with the same skill.ID may produce duplicate
// entries depending on the backing vector store's document-ID semantics;
// callers MUST use UpdateSkill when updating an existing skill.
func (p *ProceduralMemory) SaveSkill(ctx context.Context, skill *schema.Skill) error {
	if skill == nil {
		return fmt.Errorf("procedural: skill must not be nil")
	}
	if skill.ID == "" {
		return fmt.Errorf("procedural: skill ID is required")
	}
	if skill.Name == "" {
		return fmt.Errorf("procedural: skill name is required")
	}

	now := time.Now()
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	skill.UpdatedAt = now
	if skill.Version == 0 {
		skill.Version = 1
	}

	doc, err := p.skillToDocument(skill)
	if err != nil {
		return fmt.Errorf("procedural: serialize skill: %w", err)
	}

	text := p.skillSearchText(skill)
	vec, err := p.emb.EmbedSingle(ctx, text)
	if err != nil {
		return fmt.Errorf("procedural: embed skill: %w", err)
	}

	if err := p.vs.Add(ctx, []schema.Document{doc}, [][]float32{vec}); err != nil {
		return fmt.Errorf("procedural: store skill: %w", err)
	}

	if p.opts.hooks.OnSkillSaved != nil {
		p.opts.hooks.OnSkillSaved(ctx, skill)
	}
	return nil
}

// SearchSkills finds skills semantically similar to the query, returning at
// most k results. Skills with confidence below the configured minimum
// threshold are filtered out.
func (p *ProceduralMemory) SearchSkills(ctx context.Context, query string, k int) ([]*schema.Skill, error) {
	if k <= 0 {
		k = 10
	}

	vec, err := p.emb.EmbedSingle(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("procedural: embed query: %w", err)
	}

	// Request more results than k to allow for confidence filtering.
	fetchK := k * 2
	if fetchK < 20 {
		fetchK = 20
	}

	docs, err := p.vs.Search(ctx, vec, fetchK)
	if err != nil {
		return nil, fmt.Errorf("procedural: search skills: %w", err)
	}

	var skills []*schema.Skill
	for _, doc := range docs {
		sk, err := p.documentToSkill(doc)
		if err != nil {
			continue // skip malformed entries
		}
		if sk.Confidence < p.opts.minConfidence {
			continue
		}
		skills = append(skills, sk)
		if len(skills) >= k {
			break
		}
	}

	if p.opts.hooks.OnSkillRetrieved != nil {
		p.opts.hooks.OnSkillRetrieved(ctx, query, skills)
	}
	return skills, nil
}

// UpdateSkill updates an existing skill. The version is incremented
// automatically. To avoid destroying the skill on transient failures,
// the new version is fully prepared (serialized and embedded) before
// any mutation of the vector store; only then does UpdateSkill delete
// the old document and add the new one. On any error the in-memory
// skill's mutated Version/UpdatedAt are rolled back.
func (p *ProceduralMemory) UpdateSkill(ctx context.Context, skill *schema.Skill) error {
	if skill == nil {
		return fmt.Errorf("procedural: skill must not be nil")
	}
	if skill.ID == "" {
		return fmt.Errorf("procedural: skill ID is required")
	}

	// Fetch old version for hook.
	var old *schema.Skill
	if p.opts.hooks.OnSkillUpdated != nil {
		var err error
		old, err = p.GetSkill(ctx, skill.ID)
		if err != nil {
			return fmt.Errorf("procedural: get old skill for update: %w", err)
		}
	}

	// 1. Prepare the new version without touching the store.
	origVersion := skill.Version
	origUpdatedAt := skill.UpdatedAt
	skill.Version++
	skill.UpdatedAt = time.Now()

	doc, err := p.skillToDocument(skill)
	if err != nil {
		skill.Version = origVersion
		skill.UpdatedAt = origUpdatedAt
		return fmt.Errorf("procedural: serialize skill: %w", err)
	}

	text := p.skillSearchText(skill)
	vec, err := p.emb.EmbedSingle(ctx, text)
	if err != nil {
		skill.Version = origVersion
		skill.UpdatedAt = origUpdatedAt
		return fmt.Errorf("procedural: embed updated skill: %w", err)
	}

	// 2. Now mutate the store: delete the old, then add the new.
	// The delete-add window is inherently non-transactional without
	// backend transaction support; this is the minimum-risk ordering.
	if err := p.vs.Delete(ctx, []string{skillDocID(skill.ID)}); err != nil {
		skill.Version = origVersion
		skill.UpdatedAt = origUpdatedAt
		return fmt.Errorf("procedural: delete old skill: %w", err)
	}

	if err := p.vs.Add(ctx, []schema.Document{doc}, [][]float32{vec}); err != nil {
		return fmt.Errorf("procedural: store updated skill: %w", err)
	}

	if p.opts.hooks.OnSkillUpdated != nil {
		p.opts.hooks.OnSkillUpdated(ctx, old, skill)
	}
	return nil
}

// DeleteSkill removes a skill from the vector store by its ID.
func (p *ProceduralMemory) DeleteSkill(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("procedural: skill ID is required")
	}
	return p.vs.Delete(ctx, []string{skillDocID(id)})
}

// GetSkill retrieves a single skill by ID. Returns nil and no error if the
// skill is not found.
//
// NOTE: The underlying VectorStore does not expose a get-by-ID primitive,
// so GetSkill uses a filtered Search with a zero vector. Two caveats apply:
//  1. vectorstore.WithFilter is a hint, not a guarantee — if the backend
//     ignores the filter and holds more than 100 total documents, the
//     target skill may be pushed beyond the result window and GetSkill
//     will return nil for an existing skill.
//  2. Some backends compute cosine similarity over normalised vectors;
//     searching with a zero vector has undefined ordering semantics.
//
// For production workloads that require reliable ID lookup, pair this
// package with a backend that honours metadata filters, or maintain an
// auxiliary key-value index.
func (p *ProceduralMemory) GetSkill(ctx context.Context, id string) (*schema.Skill, error) {
	if id == "" {
		return nil, fmt.Errorf("procedural: skill ID is required")
	}

	// Use a zero vector to search, then filter by ID.
	// This is a workaround since VectorStore does not support get-by-ID.
	vec := make([]float32, p.emb.Dimensions())
	docs, err := p.vs.Search(ctx, vec, 100, vectorstore.WithFilter(map[string]any{
		"skill_id": id,
	}))
	if err != nil {
		return nil, fmt.Errorf("procedural: search for skill %q: %w", id, err)
	}
	for _, doc := range docs {
		sk, err := p.documentToSkill(doc)
		if err != nil {
			continue
		}
		if sk.ID == id {
			return sk, nil
		}
	}
	return nil, nil
}

// skillDocID returns the document ID used to store a skill in the vector store.
func skillDocID(skillID string) string {
	return "skill-" + skillID
}

// skillSearchText builds the text used for embedding. It combines the skill's
// name, description, and triggers into a single searchable string.
func (p *ProceduralMemory) skillSearchText(skill *schema.Skill) string {
	var b strings.Builder
	b.WriteString(skill.Name)
	b.WriteString(": ")
	b.WriteString(skill.Description)
	if len(skill.Triggers) > 0 {
		b.WriteString(" [triggers: ")
		b.WriteString(strings.Join(skill.Triggers, ", "))
		b.WriteString("]")
	}
	if len(skill.Steps) > 0 {
		b.WriteString(" [steps: ")
		b.WriteString(strings.Join(skill.Steps, "; "))
		b.WriteString("]")
	}
	return b.String()
}

// skillToDocument serializes a skill into a schema.Document for vector storage.
func (p *ProceduralMemory) skillToDocument(skill *schema.Skill) (schema.Document, error) {
	content, err := json.Marshal(skill)
	if err != nil {
		return schema.Document{}, fmt.Errorf("marshal skill: %w", err)
	}
	return schema.Document{
		ID:      skillDocID(skill.ID),
		Content: string(content),
		Metadata: map[string]any{
			"skill_id":   skill.ID,
			"agent_id":   skill.AgentID,
			"confidence": skill.Confidence,
			"version":    skill.Version,
			"type":       "procedural_skill",
		},
	}, nil
}

// documentToSkill deserializes a schema.Document back into a Skill.
func (p *ProceduralMemory) documentToSkill(doc schema.Document) (*schema.Skill, error) {
	var skill schema.Skill
	if err := json.Unmarshal([]byte(doc.Content), &skill); err != nil {
		return nil, fmt.Errorf("unmarshal skill: %w", err)
	}
	return &skill, nil
}
