package schema

import (
	"math"
	"testing"
)

func TestDocument_Fields(t *testing.T) {
	tests := []struct {
		name         string
		doc          Document
		wantID       string
		wantContent  string
		wantScore    float64
		wantMeta     bool
		wantEmbed    bool
	}{
		{
			name: "fully_populated",
			doc: Document{
				ID:        "doc-1",
				Content:   "Hello, world!",
				Metadata:  map[string]any{"source": "test"},
				Score:     0.95,
				Embedding: []float32{0.1, 0.2, 0.3},
			},
			wantID:      "doc-1",
			wantContent: "Hello, world!",
			wantScore:   0.95,
			wantMeta:    true,
			wantEmbed:   true,
		},
		{
			name: "minimal",
			doc: Document{
				ID:      "doc-2",
				Content: "Some text",
			},
			wantID:      "doc-2",
			wantContent: "Some text",
			wantScore:   0,
			wantMeta:    false,
			wantEmbed:   false,
		},
		{
			name: "empty_content",
			doc: Document{
				ID:      "doc-3",
				Content: "",
			},
			wantID:      "doc-3",
			wantContent: "",
			wantScore:   0,
			wantMeta:    false,
			wantEmbed:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doc.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", tt.doc.ID, tt.wantID)
			}
			if tt.doc.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", tt.doc.Content, tt.wantContent)
			}
			if tt.doc.Score != tt.wantScore {
				t.Errorf("Score = %f, want %f", tt.doc.Score, tt.wantScore)
			}
			hasMeta := tt.doc.Metadata != nil
			if hasMeta != tt.wantMeta {
				t.Errorf("has Metadata = %v, want %v", hasMeta, tt.wantMeta)
			}
			hasEmbed := tt.doc.Embedding != nil
			if hasEmbed != tt.wantEmbed {
				t.Errorf("has Embedding = %v, want %v", hasEmbed, tt.wantEmbed)
			}
		})
	}
}

func TestDocument_ZeroValue(t *testing.T) {
	var doc Document
	if doc.ID != "" {
		t.Errorf("zero ID = %q, want empty", doc.ID)
	}
	if doc.Content != "" {
		t.Errorf("zero Content = %q, want empty", doc.Content)
	}
	if doc.Metadata != nil {
		t.Errorf("zero Metadata = %v, want nil", doc.Metadata)
	}
	if doc.Score != 0 {
		t.Errorf("zero Score = %f, want 0", doc.Score)
	}
	if doc.Embedding != nil {
		t.Errorf("zero Embedding = %v, want nil", doc.Embedding)
	}
}

func TestDocument_Metadata(t *testing.T) {
	doc := Document{
		ID:      "doc-meta",
		Content: "test",
		Metadata: map[string]any{
			"source":   "wikipedia",
			"page":     42,
			"verified": true,
			"tags":     []string{"science", "math"},
		},
	}

	if v, ok := doc.Metadata["source"].(string); !ok || v != "wikipedia" {
		t.Errorf("Metadata[\"source\"] = %v, want %q", doc.Metadata["source"], "wikipedia")
	}
	if v, ok := doc.Metadata["page"].(int); !ok || v != 42 {
		t.Errorf("Metadata[\"page\"] = %v, want 42", doc.Metadata["page"])
	}
	if v, ok := doc.Metadata["verified"].(bool); !ok || !v {
		t.Errorf("Metadata[\"verified\"] = %v, want true", doc.Metadata["verified"])
	}
	if _, ok := doc.Metadata["nonexistent"]; ok {
		t.Error("Metadata[\"nonexistent\"] should not exist")
	}
}

func TestDocument_Embedding(t *testing.T) {
	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	doc := Document{
		ID:        "doc-embed",
		Content:   "test",
		Embedding: embedding,
	}

	if len(doc.Embedding) != 5 {
		t.Fatalf("len(Embedding) = %d, want 5", len(doc.Embedding))
	}

	for i, want := range embedding {
		if got := doc.Embedding[i]; got != want {
			t.Errorf("Embedding[%d] = %f, want %f", i, got, want)
		}
	}
}

func TestDocument_Score_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		score float64
	}{
		{"zero", 0.0},
		{"one", 1.0},
		{"negative", -0.5},
		{"very_small", 0.000001},
		{"large", 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := Document{ID: "test", Score: tt.score}
			if math.Abs(doc.Score-tt.score) > 1e-10 {
				t.Errorf("Score = %f, want %f", doc.Score, tt.score)
			}
		})
	}
}

func TestDocument_EmptyEmbedding(t *testing.T) {
	doc := Document{
		ID:        "doc-empty-embed",
		Content:   "test",
		Embedding: []float32{},
	}

	if doc.Embedding == nil {
		t.Error("Embedding should not be nil for empty slice")
	}
	if len(doc.Embedding) != 0 {
		t.Errorf("len(Embedding) = %d, want 0", len(doc.Embedding))
	}
}

func TestDocument_MetadataMutation(t *testing.T) {
	doc := Document{
		ID:       "doc-mut",
		Content:  "test",
		Metadata: map[string]any{"key": "original"},
	}

	doc.Metadata["key"] = "modified"
	doc.Metadata["new_key"] = "new_value"

	if doc.Metadata["key"] != "modified" {
		t.Errorf("Metadata[\"key\"] = %v, want %q", doc.Metadata["key"], "modified")
	}
	if doc.Metadata["new_key"] != "new_value" {
		t.Errorf("Metadata[\"new_key\"] = %v, want %q", doc.Metadata["new_key"], "new_value")
	}
}
