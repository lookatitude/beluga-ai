// Package bayesian provides Bayesian optimization components for the optimize package.
//
// This implementation includes the Tree-structured Parzen Estimator (TPE) algorithm
// used by MIPROv2 for efficient hyperparameter search.
package bayesian

import (
	"math"
	"math/rand"
	"sort"
)

// TPE implements the Tree-structured Parzen Estimator algorithm.
// TPE models the distribution of good vs bad hyperparameters
// and samples new points that are likely to be good.
type TPE struct {
	// Gamma is the quantile threshold for separating good/bad trials (default: 0.25).
	Gamma float64
	// Random state for reproducibility.
	Rand *rand.Rand
}

// NewTPE creates a new TPE sampler with defaults.
func NewTPE() *TPE {
	return &TPE{
		Gamma: 0.25,
		Rand:  rand.New(rand.NewSource(42)),
	}
}

// WithSeed sets the random seed for reproducibility.
func (t *TPE) WithSeed(seed int64) *TPE {
	t.Rand = rand.New(rand.NewSource(seed))
	return t
}

// WithGamma sets the quantile threshold.
func (t *TPE) WithGamma(gamma float64) *TPE {
	t.Gamma = gamma
	return t
}

// Trial represents a completed optimization trial.
type Trial struct {
	Params map[string]interface{}
	Score  float64
}

// CategoricalChoice represents a choice from categorical options.
type CategoricalChoice struct {
	Options []string
	Weights []float64 // Optional weights (nil = uniform)
}

// SampleInt samples an integer from the TPE distribution given past trials.
func (t *TPE) SampleInt(name string, low, high int, trials []Trial) int {
	if len(trials) < 5 {
		// Not enough trials, sample uniformly
		return low + t.Rand.Intn(high-low+1)
	}

	// Separate good and bad trials
	goodTrials, badTrials := t.splitTrials(trials, name)

	// Build distributions
	goodDist := buildIntDistribution(goodTrials, low, high)
	badDist := buildIntDistribution(badTrials, low, high)

	// Sample from the ratio distribution
	return t.sampleFromRatio(goodDist, badDist, low, high)
}

// SampleFloat samples a float from the TPE distribution given past trials.
func (t *TPE) SampleFloat(name string, low, high float64, logScale bool, trials []Trial) float64 {
	if len(trials) < 5 {
		// Not enough trials, sample uniformly
		if logScale {
			return math.Exp(math.Log(low) + t.Rand.Float64()*(math.Log(high)-math.Log(low)))
		}
		return low + t.Rand.Float64()*(high-low)
	}

	// Separate good and bad trials
	goodTrials, badTrials := t.splitTrials(trials, name)

	// Transform to log scale if needed
	if logScale {
		low = math.Log(low)
		high = math.Log(high)
		for i := range goodTrials {
			if v, ok := goodTrials[i].Params[name].(float64); ok {
				goodTrials[i].Params[name] = math.Log(v)
			}
		}
		for i := range badTrials {
			if v, ok := badTrials[i].Params[name].(float64); ok {
				badTrials[i].Params[name] = math.Log(v)
			}
		}
	}

	// Build distributions
	goodDist := buildFloatDistribution(goodTrials, low, high, name)
	badDist := buildFloatDistribution(badTrials, low, high, name)

	// Sample from the ratio distribution
	value := t.sampleFloatFromRatio(goodDist, badDist, low, high)

	if logScale {
		return math.Exp(value)
	}
	return value
}

// SampleCategorical samples from categorical options.
func (t *TPE) SampleCategorical(name string, choices CategoricalChoice, trials []Trial) string {
	if len(trials) < 5 || len(choices.Options) == 0 {
		// Not enough trials or no options, sample uniformly
		if len(choices.Options) == 0 {
			return ""
		}
		return choices.Options[t.Rand.Intn(len(choices.Options))]
	}

	// Count occurrences in good trials
	goodTrials, _ := t.splitTrials(trials, name)
	counts := make(map[string]int)
	for _, t := range goodTrials {
		if s, ok := t.Params[name].(string); ok {
			counts[s]++
		}
	}

	// Use counts as weights (with smoothing)
	weights := make([]float64, len(choices.Options))
	for i, opt := range choices.Options {
		weights[i] = float64(counts[opt] + 1) // Add-1 smoothing
	}

	// Sample according to weights
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}
	r := t.Rand.Float64() * totalWeight
	cumsum := 0.0
	for i, w := range weights {
		cumsum += w
		if r <= cumsum {
			return choices.Options[i]
		}
	}
	return choices.Options[len(choices.Options)-1]
}

// splitTrials separates trials into good (top gamma quantile) and bad (rest).
func (t *TPE) splitTrials(trials []Trial, paramName string) ([]Trial, []Trial) {
	// Sort by score (descending)
	sorted := make([]Trial, len(trials))
	copy(sorted, trials)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	// Split by gamma quantile
	nGood := int(math.Max(1, float64(len(sorted))*t.Gamma))
	goodTrials := sorted[:nGood]
	badTrials := sorted[nGood:]

	return goodTrials, badTrials
}

// intDistribution represents a discrete probability distribution over integers.
type intDistribution struct {
	counts map[int]int
	total  int
}

// buildIntDistribution builds a distribution from trial values.
func buildIntDistribution(trials []Trial, low, high int) intDistribution {
	d := intDistribution{
		counts: make(map[int]int),
	}
	for _, t := range trials {
		if v, ok := t.Params["value"].(int); ok {
			if v >= low && v <= high {
				d.counts[v]++
				d.total++
			}
		}
	}
	// Add smoothing
	for i := low; i <= high; i++ {
		if d.counts[i] == 0 {
			d.counts[i] = 1
			d.total++
		}
	}
	return d
}

// sampleFromRatio samples from p(good) / p(bad).
func (t *TPE) sampleFromRatio(goodDist, badDist intDistribution, low, high int) int {
	// Calculate ratio for each value
	type ratio struct {
		value int
		score float64
	}
	ratios := make([]ratio, 0, high-low+1)
	for i := low; i <= high; i++ {
		pGood := float64(goodDist.counts[i]) / float64(goodDist.total)
		pBad := float64(badDist.counts[i]) / float64(badDist.total)
		if pBad == 0 {
			pBad = 0.0001 // Avoid division by zero
		}
		ratios = append(ratios, ratio{i, pGood / pBad})
	}

	// Sort by ratio descending
	sort.Slice(ratios, func(i, j int) bool {
		return ratios[i].score > ratios[j].score
	})

	// Sample top candidates with some exploration
	nTop := int(math.Max(1, float64(len(ratios))*0.3))
	if nTop > len(ratios) {
		nTop = len(ratios)
	}
	return ratios[t.Rand.Intn(nTop)].value
}

// floatDistribution represents a continuous distribution (modeled as Gaussian KDE).
type floatDistribution struct {
	values []float64
	mean   float64
	std    float64
}

// buildFloatDistribution builds a Gaussian KDE from trial values.
func buildFloatDistribution(trials []Trial, low, high float64, name string) floatDistribution {
	values := make([]float64, 0, len(trials))
	var sum float64
	for _, t := range trials {
		if v, ok := t.Params[name].(float64); ok {
			if v >= low && v <= high {
				values = append(values, v)
				sum += v
			}
		}
	}

	if len(values) == 0 {
		// Return uniform distribution
		return floatDistribution{
			values: []float64{low, high},
			mean:   (low + high) / 2,
			std:    (high - low) / 4,
		}
	}

	mean := sum / float64(len(values))
	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	std := math.Sqrt(variance / float64(len(values)))
	if std == 0 {
		std = (high - low) / 10 // Minimum std
	}

	return floatDistribution{
		values: values,
		mean:   mean,
		std:    std,
	}
}

// sampleFloatFromRatio samples from the ratio of two Gaussian distributions.
func (t *TPE) sampleFloatFromRatio(goodDist, badDist floatDistribution, low, high float64) float64 {
	// Sample multiple candidates and pick best by ratio
	bestValue := goodDist.mean
	bestRatio := -1.0

	for i := 0; i < 100; i++ {
		// Sample from good distribution
		value := goodDist.mean + t.Rand.NormFloat64()*goodDist.std

		// Clip to bounds
		if value < low {
			value = low
		}
		if value > high {
			value = high
		}

		// Calculate ratio
		pGood := gaussianPDF(value, goodDist.mean, goodDist.std)
		pBad := gaussianPDF(value, badDist.mean, badDist.std)
		if pBad < 0.0001 {
			pBad = 0.0001
		}
		ratio := pGood / pBad

		if ratio > bestRatio {
			bestRatio = ratio
			bestValue = value
		}
	}

	return bestValue
}

// gaussianPDF computes the probability density of a Gaussian distribution.
func gaussianPDF(x, mean, std float64) float64 {
	if std == 0 {
		if x == mean {
			return 1.0
		}
		return 0.0
	}
	variance := std * std
	return math.Exp(-0.5*math.Pow(x-mean, 2)/variance) / (std * math.Sqrt(2*math.Pi))
}
