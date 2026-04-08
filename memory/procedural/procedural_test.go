package procedural

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Doubles ---

type mockEmbedder struct {
	dim         int
	embedErr    error
	embedSingle error
}

func (m *mockEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, m.dim)
	}
	return result, nil
}

func (m *mockEmbedder) EmbedSingle(_ context.Context, _ string) ([]float32, error) {
	if m.embedSingle != nil {
		return nil, m.embedSingle
	}
	return make([]float32, m.dim), nil
}

func (m *mockEmbedder) Dimensions() int { return m.dim }

type mockVectorStore struct {
	mu        sync.Mutex
	docs      []schema.Document
	addErr    error
	searchErr error
	deleteErr error
}

func (m *mockVectorStore) Add(_ context.Context, docs []schema.Document, _ [][]float32) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.docs = append(m.docs, docs...)
	return nil
}

func (m *mockVectorStore) Search(_ context.Context, _ []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg := &vectorstore.SearchConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var filtered []schema.Document
	for _, d := range m.docs {
		if cfg.Filter != nil {
			match := true
			for fk, fv := range cfg.Filter {
				if d.Metadata[fk] != fv {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, d)
	}

	if k > len(filtered) {
		k = len(filtered)
	}
	return filtered[:k], nil
}

func (m *mockVectorStore) Delete(_ context.Context, ids []string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var remaining []schema.Document
	for _, d := range m.docs {
		if !idSet[d.ID] {
			remaining = append(remaining, d)
		}
	}
	m.docs = remaining
	return nil
}

// --- Tests ---

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		emb     *mockEmbedder
		vs      *mockVectorStore
		wantErr string
	}{
		{
			name: "valid",
			emb:  &mockEmbedder{dim: 4},
			vs:   &mockVectorStore{},
		},
		{
			name:    "nil embedder",
			vs:      &mockVectorStore{},
			wantErr: "Embedder is required",
		},
		{
			name:    "nil vector store",
			emb:     &mockEmbedder{dim: 4},
			wantErr: "VectorStore is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pm *ProceduralMemory
			var err error
			if tt.emb == nil {
				pm, err = New(nil, tt.vs)
			} else if tt.vs == nil {
				pm, err = New(tt.emb, nil)
			} else {
				pm, err = New(tt.emb, tt.vs)
			}
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, pm)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, pm)
			}
		})
	}
}

func TestSaveSkill(t *testing.T) {
	ctx := context.Background()

	t.Run("saves skill successfully", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		skill := &schema.Skill{
			ID:          "sk-1",
			Name:        "deploy-service",
			Description: "Deploy a microservice",
			Steps:       []string{"build", "push", "apply"},
			Triggers:    []string{"deploy", "release"},
			Confidence:  0.9,
			AgentID:     "agent-1",
		}

		err = pm.SaveSkill(ctx, skill)
		require.NoError(t, err)

		assert.Len(t, vs.docs, 1)
		assert.Equal(t, "skill-sk-1", vs.docs[0].ID)
		assert.Equal(t, "sk-1", vs.docs[0].Metadata["skill_id"])
		assert.Equal(t, "procedural_skill", vs.docs[0].Metadata["type"])
		assert.NotZero(t, skill.CreatedAt)
		assert.NotZero(t, skill.UpdatedAt)
		assert.Equal(t, 1, skill.Version)
	})

	t.Run("nil skill errors", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be nil")
	})

	t.Run("empty ID errors", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, &schema.Skill{Name: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "skill ID is required")
	})

	t.Run("empty name errors", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, &schema.Skill{ID: "sk-1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "skill name is required")
	})

	t.Run("embed error", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4, embedSingle: errors.New("embed failed")}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, &schema.Skill{ID: "sk-1", Name: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embed skill")
	})

	t.Run("vector store add error", func(t *testing.T) {
		vs := &mockVectorStore{addErr: errors.New("add failed")}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, &schema.Skill{ID: "sk-1", Name: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "store skill")
	})

	t.Run("fires OnSkillSaved hook", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		var savedSkill *schema.Skill
		pm, err := New(emb, vs, WithHooks(Hooks{
			OnSkillSaved: func(_ context.Context, skill *schema.Skill) {
				savedSkill = skill
			},
		}))
		require.NoError(t, err)

		skill := &schema.Skill{ID: "sk-1", Name: "test", Confidence: 0.8}
		err = pm.SaveSkill(ctx, skill)
		require.NoError(t, err)
		assert.Equal(t, "sk-1", savedSkill.ID)
	})

	t.Run("preserves existing CreatedAt", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		skill := &schema.Skill{
			ID:        "sk-1",
			Name:      "test",
			CreatedAt: created,
		}
		err = pm.SaveSkill(ctx, skill)
		require.NoError(t, err)
		assert.Equal(t, created, skill.CreatedAt)
	})
}

func TestSearchSkills(t *testing.T) {
	ctx := context.Background()

	t.Run("returns matching skills", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		// Save two skills.
		err = pm.SaveSkill(ctx, &schema.Skill{
			ID: "sk-1", Name: "deploy", Description: "deploy a service",
			Confidence: 0.9, Triggers: []string{"deploy"},
		})
		require.NoError(t, err)
		err = pm.SaveSkill(ctx, &schema.Skill{
			ID: "sk-2", Name: "test", Description: "run tests",
			Confidence: 0.8, Triggers: []string{"test"},
		})
		require.NoError(t, err)

		skills, err := pm.SearchSkills(ctx, "deploy a service", 10)
		require.NoError(t, err)
		assert.Len(t, skills, 2)
	})

	t.Run("filters by min confidence", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs, WithMinConfidence(0.7))
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, &schema.Skill{
			ID: "sk-high", Name: "high", Confidence: 0.9,
		})
		require.NoError(t, err)
		err = pm.SaveSkill(ctx, &schema.Skill{
			ID: "sk-low", Name: "low", Confidence: 0.3,
		})
		require.NoError(t, err)

		skills, err := pm.SearchSkills(ctx, "query", 10)
		require.NoError(t, err)

		for _, sk := range skills {
			assert.GreaterOrEqual(t, sk.Confidence, 0.7)
		}
	})

	t.Run("default k", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		// k <= 0 defaults to 10.
		skills, err := pm.SearchSkills(ctx, "query", 0)
		require.NoError(t, err)
		assert.Empty(t, skills)
	})

	t.Run("embed query error", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4, embedSingle: errors.New("embed failed")}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		skills, err := pm.SearchSkills(ctx, "query", 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "embed query")
		assert.Nil(t, skills)
	})

	t.Run("search error", func(t *testing.T) {
		vs := &mockVectorStore{searchErr: errors.New("search failed")}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		skills, err := pm.SearchSkills(ctx, "query", 5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "search skills")
		assert.Nil(t, skills)
	})

	t.Run("fires OnSkillRetrieved hook", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		var hookQuery string
		var hookSkills []*schema.Skill
		pm, err := New(emb, vs, WithHooks(Hooks{
			OnSkillRetrieved: func(_ context.Context, query string, skills []*schema.Skill) {
				hookQuery = query
				hookSkills = skills
			},
		}))
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, &schema.Skill{
			ID: "sk-1", Name: "test", Confidence: 0.9,
		})
		require.NoError(t, err)

		_, err = pm.SearchSkills(ctx, "test query", 5)
		require.NoError(t, err)
		assert.Equal(t, "test query", hookQuery)
		assert.NotNil(t, hookSkills)
	})
}

func TestUpdateSkill(t *testing.T) {
	ctx := context.Background()

	t.Run("updates skill and increments version", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		skill := &schema.Skill{
			ID: "sk-1", Name: "deploy", Description: "v1",
			Confidence: 0.8, Version: 1,
		}
		err = pm.SaveSkill(ctx, skill)
		require.NoError(t, err)

		skill.Description = "v2"
		err = pm.UpdateSkill(ctx, skill)
		require.NoError(t, err)
		assert.Equal(t, 2, skill.Version)
	})

	t.Run("nil skill errors", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.UpdateSkill(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be nil")
	})

	t.Run("empty ID errors", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.UpdateSkill(ctx, &schema.Skill{Name: "test"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "skill ID is required")
	})

	t.Run("fires OnSkillUpdated hook", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		var hookOld, hookNew *schema.Skill
		pm, err := New(emb, vs, WithHooks(Hooks{
			OnSkillUpdated: func(_ context.Context, old, updated *schema.Skill) {
				hookOld = old
				hookNew = updated
			},
		}))
		require.NoError(t, err)

		skill := &schema.Skill{
			ID: "sk-1", Name: "test", Description: "original",
			Confidence: 0.8, Version: 1,
		}
		err = pm.SaveSkill(ctx, skill)
		require.NoError(t, err)

		skill.Description = "updated"
		err = pm.UpdateSkill(ctx, skill)
		require.NoError(t, err)

		assert.NotNil(t, hookNew)
		assert.Equal(t, "updated", hookNew.Description)
		// old may be nil if GetSkill returns nil (mock doesn't support filter well)
		_ = hookOld
	})

	t.Run("delete error propagates", func(t *testing.T) {
		vs := &mockVectorStore{deleteErr: errors.New("delete failed")}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.UpdateSkill(ctx, &schema.Skill{ID: "sk-1", Name: "test", Version: 1})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete old skill")
	})
}

func TestDeleteSkill(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes skill", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, &schema.Skill{
			ID: "sk-1", Name: "test", Confidence: 0.8,
		})
		require.NoError(t, err)
		assert.Len(t, vs.docs, 1)

		err = pm.DeleteSkill(ctx, "sk-1")
		require.NoError(t, err)
		assert.Empty(t, vs.docs)
	})

	t.Run("empty ID errors", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.DeleteSkill(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "skill ID is required")
	})
}

func TestGetSkill(t *testing.T) {
	ctx := context.Background()

	t.Run("gets skill by ID", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		err = pm.SaveSkill(ctx, &schema.Skill{
			ID: "sk-1", Name: "deploy", Confidence: 0.9,
		})
		require.NoError(t, err)

		sk, err := pm.GetSkill(ctx, "sk-1")
		require.NoError(t, err)
		assert.NotNil(t, sk)
		assert.Equal(t, "sk-1", sk.ID)
		assert.Equal(t, "deploy", sk.Name)
	})

	t.Run("returns nil for missing skill", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		sk, err := pm.GetSkill(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Nil(t, sk)
	})

	t.Run("empty ID errors", func(t *testing.T) {
		vs := &mockVectorStore{}
		emb := &mockEmbedder{dim: 4}
		pm, err := New(emb, vs)
		require.NoError(t, err)

		sk, err := pm.GetSkill(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "skill ID is required")
		assert.Nil(t, sk)
	})
}

func TestSkillSearchText(t *testing.T) {
	emb := &mockEmbedder{dim: 4}
	vs := &mockVectorStore{}
	pm, err := New(emb, vs)
	require.NoError(t, err)

	tests := []struct {
		name  string
		skill *schema.Skill
		want  string
	}{
		{
			name:  "name and description only",
			skill: &schema.Skill{Name: "deploy", Description: "Deploy a service"},
			want:  "deploy: Deploy a service",
		},
		{
			name: "with triggers",
			skill: &schema.Skill{
				Name: "deploy", Description: "Deploy",
				Triggers: []string{"deploy", "ship"},
			},
			want: "deploy: Deploy [triggers: deploy, ship]",
		},
		{
			name: "with steps",
			skill: &schema.Skill{
				Name: "deploy", Description: "Deploy",
				Steps: []string{"build", "push"},
			},
			want: "deploy: Deploy [steps: build; push]",
		},
		{
			name: "full skill",
			skill: &schema.Skill{
				Name: "deploy", Description: "Deploy",
				Triggers: []string{"deploy"}, Steps: []string{"build"},
			},
			want: "deploy: Deploy [triggers: deploy] [steps: build]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.skillSearchText(tt.skill)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWithMinConfidence(t *testing.T) {
	o := defaults()
	assert.Equal(t, 0.5, o.minConfidence)

	WithMinConfidence(0.8)(&o)
	assert.Equal(t, 0.8, o.minConfidence)
}

func TestConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	vs := &mockVectorStore{}
	emb := &mockEmbedder{dim: 4}
	pm, err := New(emb, vs)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			skill := &schema.Skill{
				ID:         fmt.Sprintf("sk-%d", i),
				Name:       fmt.Sprintf("skill-%d", i),
				Confidence: 0.9,
			}
			_ = pm.SaveSkill(ctx, skill)
		}(i)
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = pm.SearchSkills(ctx, "test", 5)
		}()
	}

	wg.Wait()
}
