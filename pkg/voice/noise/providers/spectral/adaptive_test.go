package spectral

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEstimateNoiseMagnitude(t *testing.T) {
	signalMagnitude := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	percentile := 0.5

	noiseEstimate := EstimateNoiseMagnitude(signalMagnitude, percentile)
	assert.NotNil(t, noiseEstimate)
	assert.Equal(t, len(signalMagnitude), len(noiseEstimate))

	// All values should be <= threshold
	threshold := percentile * maxFloat64(signalMagnitude)
	for i := range noiseEstimate {
		assert.LessOrEqual(t, noiseEstimate[i], threshold)
	}
}

func TestEstimateNoiseMagnitude_Empty(t *testing.T) {
	signalMagnitude := []float64{}
	percentile := 0.5

	noiseEstimate := EstimateNoiseMagnitude(signalMagnitude, percentile)
	assert.NotNil(t, noiseEstimate)
	assert.Empty(t, noiseEstimate)
}

func TestEstimateNoiseMagnitude_ZeroPercentile(t *testing.T) {
	signalMagnitude := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	percentile := 0.0

	noiseEstimate := EstimateNoiseMagnitude(signalMagnitude, percentile)
	assert.NotNil(t, noiseEstimate)
	// All values should be 0 or signal value
	for i := range noiseEstimate {
		assert.GreaterOrEqual(t, noiseEstimate[i], 0.0)
	}
}

func TestMaxFloat64(t *testing.T) {
	values := []float64{1.0, 5.0, 3.0, 9.0, 2.0}
	max := maxFloat64(values)
	assert.Equal(t, 9.0, max)
}

func TestMaxFloat64_SingleValue(t *testing.T) {
	values := []float64{42.0}
	max := maxFloat64(values)
	assert.Equal(t, 42.0, max)
}

func TestMaxFloat64_Empty(t *testing.T) {
	values := []float64{}
	max := maxFloat64(values)
	assert.Equal(t, 0.0, max)
}

func TestMaxFloat64_NegativeValues(t *testing.T) {
	values := []float64{-5.0, -1.0, -10.0, -2.0}
	max := maxFloat64(values)
	assert.Equal(t, -1.0, max)
}

func TestAdaptiveNoiseProfile_Update(t *testing.T) {
	anp := NewAdaptiveNoiseProfile(10, 1, 0.95)

	// Initial state
	initial := anp.GetNoiseMagnitude()
	assert.Equal(t, 10, len(initial))

	// Update with noise estimate
	noiseMagnitude := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	anp.Update(noiseMagnitude)

	// Should update after updateRate calls
	for i := 0; i < 10; i++ {
		anp.Update(noiseMagnitude)
	}

	// Get updated noise magnitude
	updated := anp.GetNoiseMagnitude()
	assert.NotNil(t, updated)
	assert.Equal(t, 10, len(updated))
}

func TestAdaptiveNoiseProfile_Update_Partial(t *testing.T) {
	anp := NewAdaptiveNoiseProfile(10, 1, 0.95)

	// Update with smaller noise estimate
	noiseMagnitude := []float64{1.0, 2.0, 3.0}
	anp.Update(noiseMagnitude)

	// Should not panic and should handle partial data
	updated := anp.GetNoiseMagnitude()
	assert.NotNil(t, updated)
}

func TestAdaptiveNoiseProfile_Update_Larger(t *testing.T) {
	anp := NewAdaptiveNoiseProfile(5, 1, 0.95)

	// Update with larger noise estimate
	noiseMagnitude := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}
	anp.Update(noiseMagnitude)

	// Should handle larger data
	updated := anp.GetNoiseMagnitude()
	assert.NotNil(t, updated)
	assert.Equal(t, 5, len(updated))
}

