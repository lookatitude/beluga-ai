package rl

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
)

func TestTrajectoryCollector_RecordAndEnd(t *testing.T) {
	collector := NewTrajectoryCollector()

	// Record steps.
	collector.RecordStep(PolicyFeatures{StoreSize: 1}, ActionAdd, 0.9)
	collector.RecordStep(PolicyFeatures{StoreSize: 2}, ActionNoop, 0.5)

	// End episode.
	err := collector.EndEpisode(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	episodes := collector.Episodes()
	if len(episodes) != 1 {
		t.Fatalf("expected 1 episode, got %d", len(episodes))
	}

	ep := episodes[0]
	if len(ep.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(ep.Steps))
	}
	if ep.Steps[0].Action != ActionAdd {
		t.Errorf("step 0 action = %v, want ActionAdd", ep.Steps[0].Action)
	}
	if ep.Steps[1].Action != ActionNoop {
		t.Errorf("step 1 action = %v, want ActionNoop", ep.Steps[1].Action)
	}
	if ep.Outcome != true {
		t.Errorf("outcome = %v, want true", ep.Outcome)
	}
	if ep.ID == "" {
		t.Error("episode ID should not be empty")
	}
	if ep.StartTime.IsZero() {
		t.Error("start time should not be zero")
	}
	if ep.EndTime.IsZero() {
		t.Error("end time should not be zero")
	}
}

func TestTrajectoryCollector_EndEpisode_NoActive(t *testing.T) {
	collector := NewTrajectoryCollector()

	err := collector.EndEpisode(context.Background(), true)
	if err == nil {
		t.Fatal("expected error when ending non-existent episode")
	}
}

func TestTrajectoryCollector_MultipleEpisodes(t *testing.T) {
	collector := NewTrajectoryCollector()

	// Episode 1.
	collector.RecordStep(PolicyFeatures{}, ActionAdd, 0.8)
	if err := collector.EndEpisode(context.Background(), true); err != nil {
		t.Fatal(err)
	}

	// Episode 2.
	collector.RecordStep(PolicyFeatures{}, ActionDelete, 0.6)
	collector.RecordStep(PolicyFeatures{}, ActionUpdate, 0.7)
	if err := collector.EndEpisode(context.Background(), false); err != nil {
		t.Fatal(err)
	}

	if collector.Len() != 2 {
		t.Errorf("Len() = %d, want 2", collector.Len())
	}

	episodes := collector.Episodes()
	if len(episodes[0].Steps) != 1 {
		t.Errorf("episode 0 steps = %d, want 1", len(episodes[0].Steps))
	}
	if len(episodes[1].Steps) != 2 {
		t.Errorf("episode 1 steps = %d, want 2", len(episodes[1].Steps))
	}
}

func TestTrajectoryCollector_Export(t *testing.T) {
	collector := NewTrajectoryCollector()

	collector.RecordStep(PolicyFeatures{StoreSize: 5, MaxSimilarity: 0.3}, ActionAdd, 0.9)
	if err := collector.EndEpisode(context.Background(), true); err != nil {
		t.Fatal(err)
	}

	data, err := collector.Export()
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}

	// Verify it's valid JSON.
	var episodes []json.RawMessage
	if err := json.Unmarshal(data, &episodes); err != nil {
		t.Fatalf("exported data is not valid JSON: %v", err)
	}
	if len(episodes) != 1 {
		t.Errorf("expected 1 episode in export, got %d", len(episodes))
	}
}

func TestTrajectoryCollector_ExportEmpty(t *testing.T) {
	collector := NewTrajectoryCollector()

	data, err := collector.Export()
	if err != nil {
		t.Fatalf("Export error: %v", err)
	}

	// Should export as empty JSON array.
	if string(data) != "[]" && string(data) != "null" {
		t.Errorf("expected empty JSON, got %s", string(data))
	}
}

func TestTrajectoryCollector_Hooks(t *testing.T) {
	var received Episode
	hooks := Hooks{
		OnEpisodeEnd: func(_ context.Context, ep Episode) {
			received = ep
		},
	}

	collector := NewTrajectoryCollector(hooks)
	collector.RecordStep(PolicyFeatures{}, ActionAdd, 0.8)
	if err := collector.EndEpisode(context.Background(), true); err != nil {
		t.Fatal(err)
	}

	if received.ID == "" {
		t.Error("OnEpisodeEnd hook was not called")
	}
	if len(received.Steps) != 1 {
		t.Errorf("received episode steps = %d, want 1", len(received.Steps))
	}
}

func TestTrajectoryCollector_AutoStartsEpisode(t *testing.T) {
	collector := NewTrajectoryCollector()

	// RecordStep without explicit start should auto-create an episode.
	collector.RecordStep(PolicyFeatures{}, ActionAdd, 0.5)

	// Len is 0 because episode is not ended yet.
	if collector.Len() != 0 {
		t.Errorf("Len() = %d, want 0 (episode not ended)", collector.Len())
	}

	if err := collector.EndEpisode(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if collector.Len() != 1 {
		t.Errorf("Len() = %d, want 1", collector.Len())
	}
}

func TestTrajectoryCollector_ConcurrentRecording(t *testing.T) {
	collector := NewTrajectoryCollector()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			collector.RecordStep(PolicyFeatures{StoreSize: float64(i)}, ActionAdd, 0.5)
		}()
	}
	wg.Wait()

	if err := collector.EndEpisode(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if collector.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", collector.Len())
	}

	ep := collector.Episodes()[0]
	if len(ep.Steps) != 100 {
		t.Errorf("steps = %d, want 100", len(ep.Steps))
	}
}

func TestTrajectoryCollector_EpisodesCopy(t *testing.T) {
	collector := NewTrajectoryCollector()

	collector.RecordStep(PolicyFeatures{}, ActionAdd, 0.5)
	if err := collector.EndEpisode(context.Background(), true); err != nil {
		t.Fatal(err)
	}

	// Modifying returned slice should not affect collector.
	episodes := collector.Episodes()
	episodes[0].ID = "modified"

	original := collector.Episodes()
	if original[0].ID == "modified" {
		t.Error("Episodes() should return a copy")
	}
}
