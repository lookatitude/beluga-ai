package clustering

import (
	"context"
	"testing"
)

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name    string
		a, b    Conversation
		wantMin float64
		wantMax float64
	}{
		{
			name:    "identical conversations",
			a:       Conversation{Turns: []Turn{{Content: "hello world"}}},
			b:       Conversation{Turns: []Turn{{Content: "hello world"}}},
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name:    "completely different",
			a:       Conversation{Turns: []Turn{{Content: "alpha beta"}}},
			b:       Conversation{Turns: []Turn{{Content: "gamma delta"}}},
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "partial overlap",
			a:       Conversation{Turns: []Turn{{Content: "hello world foo"}}},
			b:       Conversation{Turns: []Turn{{Content: "hello world bar"}}},
			wantMin: 0.4,
			wantMax: 0.6,
		},
		{
			name:    "both empty",
			a:       Conversation{Turns: []Turn{{Content: ""}}},
			b:       Conversation{Turns: []Turn{{Content: ""}}},
			wantMin: 1.0,
			wantMax: 1.0,
		},
	}

	metric := &JaccardSimilarity{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim, err := metric.Similarity(context.Background(), tt.a, tt.b)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sim < tt.wantMin || sim > tt.wantMax {
				t.Errorf("similarity = %v, want [%v, %v]", sim, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestAgglomerativeClusterer(t *testing.T) {
	convs := []Conversation{
		{ID: "c1", Turns: []Turn{{Role: "user", Content: "hello how are you"}}},
		{ID: "c2", Turns: []Turn{{Role: "user", Content: "hello how is it going"}}},
		{ID: "c3", Turns: []Turn{{Role: "user", Content: "calculate the sum of 5 and 3"}}},
		{ID: "c4", Turns: []Turn{{Role: "user", Content: "what is the sum of 10 and 20"}}},
	}

	clusterer := NewAgglomerative(
		WithThreshold(0.2),
		WithMaxClusters(2),
	)

	clusters, err := clusterer.Cluster(context.Background(), convs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(clusters) == 0 {
		t.Fatal("expected at least one cluster")
	}

	// Verify all conversations are accounted for.
	total := 0
	for _, cl := range clusters {
		total += len(cl.Conversations)
	}
	if total != len(convs) {
		t.Errorf("total conversations in clusters = %d, want %d", total, len(convs))
	}
}

func TestAgglomerativeClusterer_EmptyInput(t *testing.T) {
	clusterer := NewAgglomerative()
	clusters, err := clusterer.Cluster(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clusters != nil {
		t.Errorf("expected nil for empty input, got %v", clusters)
	}
}

func TestAgglomerativeClusterer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	convs := make([]Conversation, 20)
	for i := range convs {
		convs[i] = Conversation{
			ID:    "c",
			Turns: []Turn{{Content: "test"}},
		}
	}

	clusterer := NewAgglomerative(WithMaxClusters(2))
	_, err := clusterer.Cluster(ctx, convs)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestTurnPatternDetector(t *testing.T) {
	convs := []Conversation{
		{ID: "c1", Turns: []Turn{{Role: "user", Content: "hi"}, {Role: "assistant", Content: "hello"}}},
		{ID: "c2", Turns: []Turn{{Role: "user", Content: "hey"}, {Role: "assistant", Content: "hi there"}}},
		{ID: "c3", Turns: []Turn{{Role: "user", Content: "question"}, {Role: "assistant", Content: "answer"}, {Role: "user", Content: "thanks"}}},
	}

	detector := &TurnPatternDetector{}
	patterns, err := detector.Detect(context.Background(), convs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(patterns) == 0 {
		t.Fatal("expected at least one pattern")
	}

	// Should detect the 2-turn pattern (c1 and c2 both have 2 turns).
	found := false
	for _, p := range patterns {
		if p.Name == "2-turn-conversation" {
			found = true
			if p.Frequency != 2 {
				t.Errorf("expected frequency 2, got %d", p.Frequency)
			}
		}
	}
	if !found {
		t.Error("expected to find 2-turn-conversation pattern")
	}
}

func TestTurnPatternDetector_EmptyInput(t *testing.T) {
	detector := &TurnPatternDetector{}
	patterns, err := detector.Detect(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patterns != nil {
		t.Errorf("expected nil for empty input, got %v", patterns)
	}
}

func TestRegistry(t *testing.T) {
	names := List()
	found := false
	for _, n := range names {
		if n == "agglomerative" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'agglomerative' in registry")
	}

	clusterer, err := New("agglomerative", Config{MaxClusters: 5, Threshold: 0.5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clusterer == nil {
		t.Error("expected non-nil clusterer")
	}

	_, err = New("nonexistent", Config{})
	if err == nil {
		t.Error("expected error for unknown clusterer")
	}
}
