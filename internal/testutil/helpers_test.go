package testutil

import (
	"errors"
	"iter"
	"testing"
)

// mockT is a minimal testing.TB implementation that captures Fatal/Fatalf calls.
type mockT struct {
	testing.TB
	failed  bool
	message string
}

func (m *mockT) Helper()                          {}
func (m *mockT) Fatalf(format string, args ...any) { m.failed = true }
func (m *mockT) Fatal(args ...any)                 { m.failed = true }

func TestAssertNoError(t *testing.T) {
	mt := &mockT{}
	AssertNoError(mt, nil)
	if mt.failed {
		t.Error("AssertNoError should not fail for nil error")
	}

	mt2 := &mockT{}
	AssertNoError(mt2, errors.New("oops"))
	if !mt2.failed {
		t.Error("AssertNoError should fail for non-nil error")
	}
}

func TestAssertError(t *testing.T) {
	mt := &mockT{}
	AssertError(mt, errors.New("oops"))
	if mt.failed {
		t.Error("AssertError should not fail for non-nil error")
	}

	mt2 := &mockT{}
	AssertError(mt2, nil)
	if !mt2.failed {
		t.Error("AssertError should fail for nil error")
	}
}

func TestAssertEqual(t *testing.T) {
	mt := &mockT{}
	AssertEqual(mt, 42, 42)
	if mt.failed {
		t.Error("AssertEqual should not fail for equal values")
	}

	mt2 := &mockT{}
	AssertEqual(mt2, 42, 43)
	if !mt2.failed {
		t.Error("AssertEqual should fail for different values")
	}

	mt3 := &mockT{}
	AssertEqual(mt3, []int{1, 2, 3}, []int{1, 2, 3})
	if mt3.failed {
		t.Error("AssertEqual should not fail for equal slices")
	}
}

func TestAssertContains(t *testing.T) {
	mt := &mockT{}
	AssertContains(mt, "hello world", "world")
	if mt.failed {
		t.Error("AssertContains should not fail when substring present")
	}

	mt2 := &mockT{}
	AssertContains(mt2, "hello world", "xyz")
	if !mt2.failed {
		t.Error("AssertContains should fail when substring absent")
	}
}

func TestCollectStream(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		seq := func(yield func(int, error) bool) {
			for i := 1; i <= 3; i++ {
				if !yield(i, nil) {
					return
				}
			}
		}

		results, err := CollectStream[int](iter.Seq2[int, error](seq))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		for i, v := range results {
			if v != i+1 {
				t.Errorf("result[%d]: expected %d, got %d", i, i+1, v)
			}
		}
	})

	t.Run("error mid-stream", func(t *testing.T) {
		testErr := errors.New("stream error")
		seq := func(yield func(int, error) bool) {
			if !yield(1, nil) {
				return
			}
			if !yield(0, testErr) {
				return
			}
			yield(3, nil)
		}

		results, err := CollectStream[int](iter.Seq2[int, error](seq))
		if !errors.Is(err, testErr) {
			t.Fatalf("expected testErr, got %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result before error, got %d", len(results))
		}
	})

	t.Run("empty stream", func(t *testing.T) {
		seq := func(yield func(string, error) bool) {}

		results, err := CollectStream[string](iter.Seq2[string, error](seq))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected 0 results, got %d", len(results))
		}
	})
}
