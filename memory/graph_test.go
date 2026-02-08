package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntity(t *testing.T) {
	entity := Entity{
		ID:   "person-1",
		Type: "person",
		Properties: map[string]any{
			"name": "Alice",
			"age":  30,
		},
	}

	assert.Equal(t, "person-1", entity.ID)
	assert.Equal(t, "person", entity.Type)
	assert.Equal(t, "Alice", entity.Properties["name"])
	assert.Equal(t, 30, entity.Properties["age"])
}

func TestRelation(t *testing.T) {
	relation := Relation{
		From: "person-1",
		To:   "company-1",
		Type: "works_at",
		Properties: map[string]any{
			"since": "2020",
			"role":  "engineer",
		},
	}

	assert.Equal(t, "person-1", relation.From)
	assert.Equal(t, "company-1", relation.To)
	assert.Equal(t, "works_at", relation.Type)
	assert.Equal(t, "2020", relation.Properties["since"])
	assert.Equal(t, "engineer", relation.Properties["role"])
}

func TestGraphResult(t *testing.T) {
	entities := []Entity{
		{ID: "1", Type: "person"},
		{ID: "2", Type: "company"},
	}

	relations := []Relation{
		{From: "1", To: "2", Type: "works_at"},
	}

	result := GraphResult{
		Entities:  entities,
		Relations: relations,
	}

	assert.Len(t, result.Entities, 2)
	assert.Len(t, result.Relations, 1)
	assert.Equal(t, "person", result.Entities[0].Type)
	assert.Equal(t, "works_at", result.Relations[0].Type)
}

func TestGraphStoreInterface(t *testing.T) {
	// Verify that mockGraphStore implements GraphStore.
	var _ GraphStore = (*mockGraphStore)(nil)
}

func TestEntityWithNilProperties(t *testing.T) {
	entity := Entity{
		ID:         "test-1",
		Type:       "test",
		Properties: nil,
	}

	assert.Equal(t, "test-1", entity.ID)
	assert.Nil(t, entity.Properties)
}

func TestRelationWithNilProperties(t *testing.T) {
	relation := Relation{
		From:       "a",
		To:         "b",
		Type:       "relates_to",
		Properties: nil,
	}

	assert.Equal(t, "a", relation.From)
	assert.Nil(t, relation.Properties)
}

func TestGraphResultEmpty(t *testing.T) {
	result := GraphResult{}

	assert.Nil(t, result.Entities)
	assert.Nil(t, result.Relations)
}

func TestEntityComplexProperties(t *testing.T) {
	entity := Entity{
		ID:   "org-1",
		Type: "organization",
		Properties: map[string]any{
			"name":      "ACME Corp",
			"founded":   1990,
			"active":    true,
			"revenue":   1.5e9,
			"locations": []string{"NYC", "SF", "LON"},
			"metadata": map[string]string{
				"industry": "tech",
				"size":     "large",
			},
		},
	}

	assert.Equal(t, "ACME Corp", entity.Properties["name"])
	assert.Equal(t, 1990, entity.Properties["founded"])
	assert.Equal(t, true, entity.Properties["active"])
	assert.Equal(t, 1.5e9, entity.Properties["revenue"])

	locations := entity.Properties["locations"].([]string)
	assert.Len(t, locations, 3)
	assert.Contains(t, locations, "NYC")

	metadata := entity.Properties["metadata"].(map[string]string)
	assert.Equal(t, "tech", metadata["industry"])
}

func TestRelationComplexProperties(t *testing.T) {
	relation := Relation{
		From: "person-1",
		To:   "person-2",
		Type: "knows",
		Properties: map[string]any{
			"since":      "2020-01-15",
			"strength":   0.85,
			"mutual":     true,
			"context":    []string{"work", "hobby"},
			"last_contact": map[string]any{
				"date":   "2024-01-10",
				"method": "email",
			},
		},
	}

	assert.Equal(t, "2020-01-15", relation.Properties["since"])
	assert.Equal(t, 0.85, relation.Properties["strength"])
	assert.Equal(t, true, relation.Properties["mutual"])

	context := relation.Properties["context"].([]string)
	assert.Contains(t, context, "work")

	lastContact := relation.Properties["last_contact"].(map[string]any)
	assert.Equal(t, "email", lastContact["method"])
}
