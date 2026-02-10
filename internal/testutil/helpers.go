package testutil

import (
	"iter"
	"reflect"
	"strings"
	"testing"
)

// AssertNoError fails the test immediately if err is non-nil, reporting the
// error message along with the caller's location.
func AssertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test immediately if err is nil, indicating that an
// error was expected but not received.
func AssertError(t testing.TB, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// AssertEqual performs a deep equality comparison between expected and actual.
// If they differ, the test fails with a message showing both values.
func AssertEqual(t testing.TB, expected, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("values not equal\nexpected: %v\n  actual: %v", expected, actual)
	}
}

// AssertContains fails the test if s does not contain substr.
func AssertContains(t testing.TB, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("expected %q to contain %q", s, substr)
	}
}

// CollectStream drains an iter.Seq2 iterator, collecting all values into a
// slice. If the iterator yields an error, collection stops immediately and
// the error is returned along with all values collected so far.
func CollectStream[T any](seq iter.Seq2[T, error]) ([]T, error) {
	var results []T
	for v, err := range seq {
		if err != nil {
			return results, err
		}
		results = append(results, v)
	}
	return results, nil
}
