package rl

import "testing"

func TestMemoryAction_String(t *testing.T) {
	tests := []struct {
		action MemoryAction
		want   string
	}{
		{ActionAdd, "add"},
		{ActionUpdate, "update"},
		{ActionDelete, "delete"},
		{ActionNoop, "noop"},
		{MemoryAction(99), "unknown(99)"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMemoryAction_Valid(t *testing.T) {
	tests := []struct {
		action MemoryAction
		valid  bool
	}{
		{ActionAdd, true},
		{ActionUpdate, true},
		{ActionDelete, true},
		{ActionNoop, true},
		{MemoryAction(-1), false},
		{MemoryAction(4), false},
		{MemoryAction(99), false},
	}
	for _, tt := range tests {
		t.Run(tt.action.String(), func(t *testing.T) {
			if got := tt.action.Valid(); got != tt.valid {
				t.Errorf("Valid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestNumActions(t *testing.T) {
	if NumActions != 4 {
		t.Errorf("NumActions = %d, want 4", NumActions)
	}
}
