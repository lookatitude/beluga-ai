package rl

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

// searchableMemory is a mock that supports Search for extractor tests.
type searchableMemory struct {
	docs      []schema.Document
	searchErr error
}

func (m *searchableMemory) Save(_ context.Context, _, _ schema.Message) error { return nil }
func (m *searchableMemory) Load(_ context.Context, _ string) ([]schema.Message, error) {
	return nil, nil
}
func (m *searchableMemory) Search(_ context.Context, _ string, _ int) ([]schema.Document, error) {
	return m.docs, m.searchErr
}
func (m *searchableMemory) Clear(_ context.Context) error { return nil }

func TestDefaultFeatureExtractor_Extract(t *testing.T) {
	tests := []struct {
		name             string
		docs             []schema.Document
		searchErr        error
		output           schema.Message
		wantStoreSize    float64
		wantMaxSim       float64
		wantMeanSim      float64
		wantHasMatch     bool
		wantTokenCountGt int
	}{
		{
			name: "with similar docs",
			docs: []schema.Document{
				{ID: "1", Score: 0.9, Metadata: map[string]any{"entry_age": 0.3, "retrieval_frequency": 5}},
				{ID: "2", Score: 0.5},
			},
			output:           schema.NewAIMessage("some test content here"),
			wantStoreSize:    2,
			wantMaxSim:       0.9,
			wantMeanSim:      0.7,
			wantHasMatch:     true,
			wantTokenCountGt: 0,
		},
		{
			name:             "no docs found",
			docs:             nil,
			output:           schema.NewAIMessage("hello"),
			wantStoreSize:    0,
			wantMaxSim:       0,
			wantMeanSim:      0,
			wantHasMatch:     false,
			wantTokenCountGt: 0,
		},
		{
			name:             "search error returns minimal features",
			searchErr:        errors.New("search failed"),
			output:           schema.NewAIMessage("hello world"),
			wantStoreSize:    0,
			wantTokenCountGt: 0,
		},
		{
			name: "below similarity threshold",
			docs: []schema.Document{
				{ID: "1", Score: 0.5},
			},
			output:        schema.NewAIMessage("test"),
			wantStoreSize: 1,
			wantMaxSim:    0.5,
			wantHasMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := &searchableMemory{docs: tt.docs, searchErr: tt.searchErr}
			extractor := &DefaultFeatureExtractor{}

			features, err := extractor.Extract(context.Background(), mem, schema.NewHumanMessage("input"), tt.output)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if features.StoreSize != tt.wantStoreSize {
				t.Errorf("StoreSize = %v, want %v", features.StoreSize, tt.wantStoreSize)
			}
			if features.MaxSimilarity != tt.wantMaxSim {
				t.Errorf("MaxSimilarity = %v, want %v", features.MaxSimilarity, tt.wantMaxSim)
			}
			if features.HasMatchingEntry != tt.wantHasMatch {
				t.Errorf("HasMatchingEntry = %v, want %v", features.HasMatchingEntry, tt.wantHasMatch)
			}
		})
	}
}

func TestDefaultFeatureExtractor_CustomTopK(t *testing.T) {
	mem := &searchableMemory{
		docs: []schema.Document{{ID: "1", Score: 0.8}},
	}
	extractor := &DefaultFeatureExtractor{TopK: 10, SimilarityThreshold: 0.9}

	features, err := extractor.Extract(context.Background(), mem, schema.NewHumanMessage("in"), schema.NewAIMessage("out"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With threshold 0.9, score 0.8 should not match.
	if features.HasMatchingEntry {
		t.Error("expected no match with threshold 0.9 and score 0.8")
	}
}

func TestDefaultFeatureExtractor_MetadataExtraction(t *testing.T) {
	mem := &searchableMemory{
		docs: []schema.Document{
			{
				ID:    "1",
				Score: 0.8,
				Metadata: map[string]any{
					"entry_age":           0.5,
					"retrieval_frequency": 3,
					"turn_index":          7,
				},
			},
		},
	}
	extractor := &DefaultFeatureExtractor{}

	features, err := extractor.Extract(context.Background(), mem, schema.NewHumanMessage("in"), schema.NewAIMessage("out"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if features.EntryAge != 0.5 {
		t.Errorf("EntryAge = %v, want 0.5", features.EntryAge)
	}
	if features.RetrievalFrequency != 3 {
		t.Errorf("RetrievalFrequency = %v, want 3", features.RetrievalFrequency)
	}
	if features.TurnIndex != 7 {
		t.Errorf("TurnIndex = %v, want 7", features.TurnIndex)
	}
}

func TestApproximateTokenCount(t *testing.T) {
	tests := []struct {
		input   string
		wantGt  int
		wantLte int
	}{
		{"", 0, 0},
		{"hello", 1, 5},
		{"hello world foo bar", 3, 10},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := approximateTokenCount(tt.input)
			if got < tt.wantGt {
				t.Errorf("approximateTokenCount(%q) = %d, want > %d", tt.input, got, tt.wantGt)
			}
			if tt.wantLte > 0 && got > tt.wantLte {
				t.Errorf("approximateTokenCount(%q) = %d, want <= %d", tt.input, got, tt.wantLte)
			}
		})
	}
}

func TestMessageText(t *testing.T) {
	msg := schema.NewAIMessage("hello world")
	text := messageText(msg)
	if text != "hello world" {
		t.Errorf("messageText = %q, want %q", text, "hello world")
	}
}

func TestMessageText_Empty(t *testing.T) {
	msg := schema.NewAIMessage("")
	text := messageText(msg)
	if text != "" {
		t.Errorf("messageText = %q, want empty", text)
	}
}

func TestMeanScore(t *testing.T) {
	tests := []struct {
		name string
		docs []schema.Document
		want float64
	}{
		{"empty", nil, 0},
		{"single", []schema.Document{{Score: 0.8}}, 0.8},
		{"multiple", []schema.Document{{Score: 0.8}, {Score: 0.4}}, 0.6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := meanScore(tt.docs)
			if diff := got - tt.want; diff > 0.001 || diff < -0.001 {
				t.Errorf("meanScore = %v, want %v", got, tt.want)
			}
		})
	}
}
