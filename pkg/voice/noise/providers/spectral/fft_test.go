package spectral

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyHamming(t *testing.T) {
	signal := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	original := make([]float64, len(signal))
	copy(original, signal)

	ApplyHamming(signal)

	// Signal should be modified
	assert.NotEqual(t, original, signal)
	assert.Equal(t, len(original), len(signal))

	// Check that at least some values are modified (windowed)
	// Window functions may not modify all values equally, especially at edges
	modified := false
	for i := range signal {
		if original[i] != signal[i] {
			modified = true
			break
		}
	}
	assert.True(t, modified, "At least some values should be modified by the window function")
}

func TestApplyHamming_Empty(t *testing.T) {
	signal := []float64{}
	ApplyHamming(signal)
	assert.Empty(t, signal)
}

func TestApplyHamming_SingleValue(t *testing.T) {
	signal := []float64{1.0}
	ApplyHamming(signal)
	assert.Equal(t, 1, len(signal))
}

func TestApplyBlackman(t *testing.T) {
	signal := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	original := make([]float64, len(signal))
	copy(original, signal)

	ApplyBlackman(signal)

	// Signal should be modified
	assert.NotEqual(t, original, signal)
	assert.Equal(t, len(original), len(signal))

	// Check that values are modified (windowed)
	for i := range signal {
		assert.NotEqual(t, original[i], signal[i])
	}
}

func TestApplyBlackman_Empty(t *testing.T) {
	signal := []float64{}
	ApplyBlackman(signal)
	assert.Empty(t, signal)
}

func TestApplyBlackman_SingleValue(t *testing.T) {
	signal := []float64{1.0}
	ApplyBlackman(signal)
	assert.Equal(t, 1, len(signal))
}

func TestGetWindowFunction(t *testing.T) {
	// Test hamming
	fn := GetWindowFunction("hamming")
	assert.NotNil(t, fn)
	signal := []float64{1.0, 2.0, 3.0}
	original := make([]float64, len(signal))
	copy(original, signal)
	fn(signal)
	assert.NotEqual(t, original, signal)

	// Test blackman
	fn = GetWindowFunction("blackman")
	assert.NotNil(t, fn)
	signal = []float64{1.0, 2.0, 3.0}
	original = make([]float64, len(signal))
	copy(original, signal)
	fn(signal)
	assert.NotEqual(t, original, signal)

	// Test default (hann)
	fn = GetWindowFunction("unknown")
	assert.NotNil(t, fn)
	signal = []float64{1.0, 2.0, 3.0}
	original = make([]float64, len(signal))
	copy(original, signal)
	fn(signal)
	assert.NotEqual(t, original, signal)

	// Test hann explicitly
	fn = GetWindowFunction("hann")
	assert.NotNil(t, fn)
	signal = []float64{1.0, 2.0, 3.0}
	original = make([]float64, len(signal))
	copy(original, signal)
	fn(signal)
	assert.NotEqual(t, original, signal)
}

func TestGetWindowFunction_EmptyString(t *testing.T) {
	fn := GetWindowFunction("")
	assert.NotNil(t, fn)
	signal := []float64{1.0, 2.0, 3.0}
	fn(signal)
	// Should not panic
}
