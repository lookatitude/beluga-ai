package temporal

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemporalResolver_Resolve(t *testing.T) {
	resolver := NewTemporalResolver()
	ctx := context.Background()
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		newRelation     *memory.Relation
		candidates      []memory.Relation
		wantInvalidated int
		wantNilResult   bool
	}{
		{
			name:          "nil new relation returns nil",
			newRelation:   nil,
			candidates:    nil,
			wantNilResult: true,
		},
		{
			name: "no candidates returns empty",
			newRelation: &memory.Relation{
				From:    "a",
				To:      "b",
				Type:    "works_at",
				ValidAt: baseTime.Add(24 * time.Hour),
			},
			candidates:      nil,
			wantInvalidated: 0,
		},
		{
			name: "candidate already superseded coexists",
			newRelation: &memory.Relation{
				From:    "a",
				To:      "b",
				Type:    "works_at",
				ValidAt: baseTime.Add(48 * time.Hour),
			},
			candidates: []memory.Relation{
				{
					From:      "a",
					To:        "b",
					Type:      "works_at",
					ValidAt:   baseTime,
					InvalidAt: timePtr(baseTime.Add(24 * time.Hour)), // already invalidated before new relation
				},
			},
			wantInvalidated: 0,
		},
		{
			name: "non-overlapping time ranges coexist",
			newRelation: &memory.Relation{
				From:      "a",
				To:        "b",
				Type:      "works_at",
				ValidAt:   baseTime,
				InvalidAt: timePtr(baseTime.Add(24 * time.Hour)),
			},
			candidates: []memory.Relation{
				{
					From:       "a",
					To:         "b",
					Type:       "works_at",
					ValidAt:    baseTime.Add(48 * time.Hour),
					Properties: map[string]any{"id": "rel-1"},
				},
			},
			wantInvalidated: 0,
		},
		{
			name: "older candidate is invalidated",
			newRelation: &memory.Relation{
				From:    "a",
				To:      "b",
				Type:    "works_at",
				ValidAt: baseTime.Add(48 * time.Hour),
			},
			candidates: []memory.Relation{
				{
					From:       "a",
					To:         "b",
					Type:       "works_at",
					ValidAt:    baseTime,
					Properties: map[string]any{"id": "rel-1"},
				},
			},
			wantInvalidated: 1,
		},
		{
			name: "newer candidate is not invalidated",
			newRelation: &memory.Relation{
				From:    "a",
				To:      "b",
				Type:    "works_at",
				ValidAt: baseTime,
			},
			candidates: []memory.Relation{
				{
					From:       "a",
					To:         "b",
					Type:       "works_at",
					ValidAt:    baseTime.Add(48 * time.Hour),
					Properties: map[string]any{"id": "rel-1"},
				},
			},
			wantInvalidated: 0,
		},
		{
			name: "multiple candidates mixed resolution",
			newRelation: &memory.Relation{
				From:    "a",
				To:      "b",
				Type:    "works_at",
				ValidAt: baseTime.Add(72 * time.Hour),
			},
			candidates: []memory.Relation{
				{
					From:       "a",
					To:         "b",
					Type:       "works_at",
					ValidAt:    baseTime,
					Properties: map[string]any{"id": "rel-1"},
				},
				{
					From:      "a",
					To:        "b",
					Type:      "works_at",
					ValidAt:   baseTime.Add(24 * time.Hour),
					InvalidAt: timePtr(baseTime.Add(48 * time.Hour)), // already superseded
				},
				{
					From:       "a",
					To:         "b",
					Type:       "works_at",
					ValidAt:    baseTime.Add(48 * time.Hour),
					Properties: map[string]any{"id": "rel-3"},
				},
			},
			wantInvalidated: 2, // rel-1 and rel-3 invalidated, rel-2 already superseded
		},
		{
			name: "candidate with same ValidAt not invalidated",
			newRelation: &memory.Relation{
				From:    "a",
				To:      "b",
				Type:    "works_at",
				ValidAt: baseTime,
			},
			candidates: []memory.Relation{
				{
					From:       "a",
					To:         "b",
					Type:       "works_at",
					ValidAt:    baseTime, // same time -- not strictly before
					Properties: map[string]any{"id": "rel-1"},
				},
			},
			wantInvalidated: 0,
		},
		{
			name: "candidate InvalidAt equals new ValidAt coexists",
			newRelation: &memory.Relation{
				From:    "a",
				To:      "b",
				Type:    "works_at",
				ValidAt: baseTime.Add(24 * time.Hour),
			},
			candidates: []memory.Relation{
				{
					From:       "a",
					To:         "b",
					Type:       "works_at",
					ValidAt:    baseTime,
					InvalidAt:  timePtr(baseTime.Add(24 * time.Hour)), // exactly equals new ValidAt
					Properties: map[string]any{"id": "rel-1"},
				},
			},
			wantInvalidated: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.Resolve(ctx, tt.newRelation, tt.candidates)
			require.NoError(t, err)

			if tt.wantNilResult {
				assert.Nil(t, result)
				return
			}

			assert.Len(t, result, tt.wantInvalidated)

			// Verify invalidated relations have InvalidAt and ExpiredAt set.
			for _, r := range result {
				assert.NotNil(t, r.InvalidAt, "invalidated relation should have InvalidAt set")
				assert.NotNil(t, r.ExpiredAt, "invalidated relation should have ExpiredAt set")
			}
		})
	}
}

func TestTemporalResolver_CompileTimeCheck(t *testing.T) {
	var _ ConflictResolver = (*TemporalResolver)(nil)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
