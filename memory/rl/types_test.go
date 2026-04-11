package rl

import (
	"testing"
	"time"
)

func TestStep(t *testing.T) {
	s := Step{
		Features:   PolicyFeatures{StoreSize: 10, MaxSimilarity: 0.5},
		Action:     ActionAdd,
		Confidence: 0.9,
		Timestamp:  time.Now(),
	}

	if s.Action != ActionAdd {
		t.Errorf("expected ActionAdd, got %v", s.Action)
	}
	if s.Confidence != 0.9 {
		t.Errorf("expected confidence 0.9, got %v", s.Confidence)
	}
}

func TestEpisode(t *testing.T) {
	now := time.Now()
	ep := Episode{
		ID:        "test-1",
		StartTime: now,
		EndTime:   now.Add(time.Minute),
		Outcome:   true,
		Steps: []Step{
			{Action: ActionAdd, Confidence: 0.8, Timestamp: now},
			{Action: ActionNoop, Confidence: 0.6, Timestamp: now.Add(time.Second)},
		},
	}

	if len(ep.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(ep.Steps))
	}
	if ep.Steps[0].Action != ActionAdd {
		t.Errorf("step 0: expected ActionAdd, got %v", ep.Steps[0].Action)
	}
	if ep.Steps[1].Action != ActionNoop {
		t.Errorf("step 1: expected ActionNoop, got %v", ep.Steps[1].Action)
	}
}
