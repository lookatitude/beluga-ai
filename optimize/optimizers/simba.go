package optimizers

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/lookatitude/beluga-ai/optimize"
)

func init() {
	optimize.RegisterOptimizer("simba", func(cfg optimize.OptimizerConfig) (optimize.Optimizer, error) {
		return NewSIMBA(), nil
	})
}

// SIMBA implements Stochastic Introspective Mini-Batch Ascent optimization.
//
// SIMBA systematically improves language model programs through:
//   - Stochastic mini-batch sampling: evaluates candidates on random subsets of training data
//     for efficient approximate scoring without full dataset passes.
//   - Softmax candidate selection: uses temperature-controlled softmax sampling so
//     higher-scoring candidates are selected more often while still allowing exploration.
//   - Challenging example identification: detects examples with high output variability
//     across candidates, focusing optimization effort where it matters most.
//   - Introspective reflection: analyzes performance patterns across iterations to generate
//     improvement rules that guide candidate mutation.
//   - Adaptive convergence detection: stops early when score variance drops below a threshold,
//     preventing wasted computation on diminishing returns.
//
// The algorithm iterates: sample mini-batch → evaluate candidates → identify hard examples →
// reflect on patterns → generate improved candidates → check convergence.
type SIMBA struct {
	// MaxIterations is the maximum number of optimization iterations.
	MaxIterations int

	// MinibatchSize is the number of examples sampled per evaluation.
	MinibatchSize int

	// CandidatePoolSize is the number of candidates maintained in the pool.
	CandidatePoolSize int

	// SamplingTemperature controls exploration vs exploitation in softmax selection.
	// Lower values (closer to 0) exploit best candidates; higher values explore more.
	// Range: 0.0-1.0, default 0.2.
	SamplingTemperature float64

	// ConvergenceThreshold is the minimum score variance to continue optimization.
	// When variance of recent scores drops below this, optimization stops.
	ConvergenceThreshold float64

	// MinVariabilityThreshold is the minimum output variability for an example to be
	// considered "challenging". Higher values are more selective.
	MinVariabilityThreshold float64

	// Seed for reproducibility.
	Seed int64
}

// SIMBAOption configures a SIMBA optimizer.
type SIMBAOption func(*SIMBA)

// NewSIMBA creates a new SIMBA optimizer with defaults.
func NewSIMBA(opts ...SIMBAOption) *SIMBA {
	s := &SIMBA{
		MaxIterations:           15,
		MinibatchSize:           20,
		CandidatePoolSize:       8,
		SamplingTemperature:     0.2,
		ConvergenceThreshold:    0.001,
		MinVariabilityThreshold: 0.3,
		Seed:                    42,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// WithSIMBAMaxIterations sets the maximum number of optimization iterations.
func WithSIMBAMaxIterations(n int) SIMBAOption {
	return func(s *SIMBA) {
		if n > 0 {
			s.MaxIterations = n
		}
	}
}

// WithSIMBAMinibatchSize sets the minibatch size for evaluation.
func WithSIMBAMinibatchSize(n int) SIMBAOption {
	return func(s *SIMBA) {
		if n > 0 {
			s.MinibatchSize = n
		}
	}
}

// WithSIMBACandidatePoolSize sets the number of candidates in the pool.
func WithSIMBACandidatePoolSize(n int) SIMBAOption {
	return func(s *SIMBA) {
		if n > 0 {
			s.CandidatePoolSize = n
		}
	}
}

// WithSIMBASamplingTemperature sets the softmax sampling temperature.
func WithSIMBASamplingTemperature(temp float64) SIMBAOption {
	return func(s *SIMBA) {
		if temp > 0 && temp <= 1.0 {
			s.SamplingTemperature = temp
		}
	}
}

// WithSIMBAConvergenceThreshold sets the convergence detection threshold.
func WithSIMBAConvergenceThreshold(threshold float64) SIMBAOption {
	return func(s *SIMBA) {
		if threshold > 0 {
			s.ConvergenceThreshold = threshold
		}
	}
}

// WithSIMBAMinVariabilityThreshold sets the minimum variability for challenging examples.
func WithSIMBAMinVariabilityThreshold(threshold float64) SIMBAOption {
	return func(s *SIMBA) {
		if threshold >= 0 {
			s.MinVariabilityThreshold = threshold
		}
	}
}

// WithSIMBASeed sets the random seed for reproducibility.
func WithSIMBASeed(seed int64) SIMBAOption {
	return func(s *SIMBA) {
		s.Seed = seed
	}
}

// simbaCandidate represents a candidate solution in the SIMBA pool.
type simbaCandidate struct {
	ID        string
	Prompt    string
	Demos     []optimize.Example
	Score     float64
	Iteration int
}

// Compile implements optimize.Optimizer.
//
// The optimization loop:
//  1. Initialize a pool of diverse candidates from training data.
//  2. For each iteration:
//     a. Sample a stochastic mini-batch from training data.
//     b. Evaluate all candidates on the mini-batch.
//     c. Identify challenging examples (high output variability across candidates).
//     d. Perform introspective reflection to derive improvement rules.
//     e. Generate new candidates biased toward challenging examples.
//     f. Use softmax sampling to select the next candidate pool.
//     g. Check for convergence (low score variance across recent iterations).
//  3. Return the best-scoring candidate as the optimized program.
func (s *SIMBA) Compile(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) (optimize.Program, error) {
	if opts.Metric == nil {
		return nil, fmt.Errorf("metric is required")
	}

	if len(opts.Trainset) == 0 {
		return nil, fmt.Errorf("trainset is required")
	}

	rng := rand.New(rand.NewSource(s.Seed))

	// Initialize candidate pool
	pool := s.initializeCandidatePool(opts.Trainset, rng)

	convergence := &optimize.ConvergenceChecker{
		WindowSize: 5,
		Threshold:  s.ConvergenceThreshold,
	}

	var bestCandidate *simbaCandidate

	for iter := 0; iter < s.MaxIterations; iter++ {
		// Step 1: Sample stochastic mini-batch
		minibatch := s.sampleMinibatch(opts.Trainset, rng)

		// Step 2: Evaluate all candidates on mini-batch
		for i := range pool {
			score, err := s.evaluateCandidate(ctx, program, pool[i], minibatch, opts.Metric)
			if err != nil {
				continue
			}
			pool[i].Score = score
			pool[i].Iteration = iter
		}

		// Track best candidate
		for i := range pool {
			if bestCandidate == nil || pool[i].Score > bestCandidate.Score {
				c := pool[i]
				bestCandidate = &c
			}
		}

		// Step 3: Identify challenging examples
		challenging := s.identifyChallengingExamples(ctx, program, pool, minibatch, opts.Metric)

		// Step 4: Introspective reflection (generates improvement rules)
		rules := s.reflect(ctx, pool, challenging, iter)

		// Step 5: Generate new candidates biased toward challenging examples
		newCandidates := s.generateImprovedCandidates(pool, challenging, opts.Trainset, rules, iter, rng)

		// Step 6: Merge new candidates and select via softmax sampling
		allCandidates := append(pool, newCandidates...)
		pool = s.softmaxSelect(allCandidates, s.CandidatePoolSize, rng)

		// Step 7: Check convergence
		bestIterScore := 0.0
		for _, c := range pool {
			if c.Score > bestIterScore {
				bestIterScore = c.Score
			}
		}
		if convergence.Update(bestIterScore) {
			break
		}

		// Check cost budget
		if opts.MaxCost != nil && opts.MaxCost.Exceeded(0, 0, iter+1) {
			break
		}
	}

	if bestCandidate == nil {
		return nil, fmt.Errorf("no valid candidate found during optimization")
	}

	return program.WithDemos(bestCandidate.Demos), nil
}

// initializeCandidatePool creates an initial diverse set of candidates.
func (s *SIMBA) initializeCandidatePool(trainset []optimize.Example, rng *rand.Rand) []simbaCandidate {
	pool := make([]simbaCandidate, 0, s.CandidatePoolSize)

	basePrompts := []string{
		"Answer the question based on the context provided.",
		"Provide a concise answer.",
		"What is the answer?",
		"Based on the information:",
		"Step by step, determine the answer:",
		"Using the context below, answer:",
		"Think carefully and respond:",
		"The answer is:",
	}

	for i := 0; i < s.CandidatePoolSize; i++ {
		// Sample diverse demo subsets
		demos := s.sampleDemos(trainset, 4, rng)

		pool = append(pool, simbaCandidate{
			ID:     fmt.Sprintf("init_%d", i),
			Prompt: basePrompts[i%len(basePrompts)],
			Demos:  demos,
		})
	}

	return pool
}

// sampleMinibatch draws a random mini-batch from the training set.
func (s *SIMBA) sampleMinibatch(trainset []optimize.Example, rng *rand.Rand) []optimize.Example {
	n := s.MinibatchSize
	if n > len(trainset) {
		n = len(trainset)
	}

	// Fisher-Yates partial shuffle for sampling without replacement
	indices := make([]int, len(trainset))
	for i := range indices {
		indices[i] = i
	}
	for i := 0; i < n; i++ {
		j := i + rng.Intn(len(indices)-i)
		indices[i], indices[j] = indices[j], indices[i]
	}

	batch := make([]optimize.Example, n)
	for i := 0; i < n; i++ {
		batch[i] = trainset[indices[i]]
	}
	return batch
}

// sampleDemos draws a random subset of training examples as demonstrations.
func (s *SIMBA) sampleDemos(trainset []optimize.Example, n int, rng *rand.Rand) []optimize.Example {
	if n > len(trainset) {
		n = len(trainset)
	}
	demos := make([]optimize.Example, n)
	perm := rng.Perm(len(trainset))
	for i := 0; i < n; i++ {
		demos[i] = trainset[perm[i]]
	}
	return demos
}

// evaluateCandidate scores a candidate on a set of examples.
func (s *SIMBA) evaluateCandidate(ctx context.Context, program optimize.Program, c simbaCandidate, examples []optimize.Example, metric optimize.Metric) (float64, error) {
	progWithDemos := program.WithDemos(c.Demos)

	var totalScore float64
	var count int

	for _, ex := range examples {
		pred, err := progWithDemos.Run(ctx, ex.Inputs)
		if err != nil {
			continue
		}

		score, err := metric.Evaluate(ex, pred, nil)
		if err != nil {
			continue
		}

		totalScore += score
		count++
	}

	if count == 0 {
		return 0.0, nil
	}

	return totalScore / float64(count), nil
}

// identifyChallengingExamples finds examples with high output variability across candidates.
// An example is "challenging" when different candidates produce significantly different scores,
// indicating the optimization hasn't converged on how to handle it.
func (s *SIMBA) identifyChallengingExamples(ctx context.Context, program optimize.Program, pool []simbaCandidate, examples []optimize.Example, metric optimize.Metric) []optimize.Example {
	challenging := make([]optimize.Example, 0)

	for _, ex := range examples {
		scores := make([]float64, 0, len(pool))

		for _, c := range pool {
			progWithDemos := program.WithDemos(c.Demos)
			pred, err := progWithDemos.Run(ctx, ex.Inputs)
			if err != nil {
				continue
			}

			score, err := metric.Evaluate(ex, pred, nil)
			if err != nil {
				continue
			}
			scores = append(scores, score)
		}

		if len(scores) < 2 {
			continue
		}

		// Calculate variance as a measure of output variability
		variability := variance(scores)
		if variability >= s.MinVariabilityThreshold {
			challenging = append(challenging, ex)
		}
	}

	return challenging
}

// reflect performs introspective analysis on pool performance.
// It analyzes score distributions and challenging examples to generate improvement rules.
// In a full implementation, this would use an LLM to analyze patterns and suggest improvements.
func (s *SIMBA) reflect(ctx context.Context, pool []simbaCandidate, challenging []optimize.Example, iteration int) []string {
	// Analyze pool performance distribution
	scores := make([]float64, len(pool))
	for i, c := range pool {
		scores[i] = c.Score
	}

	rules := make([]string, 0)

	// Rule: if many challenging examples, focus on diversity
	if len(challenging) > 0 {
		rules = append(rules, "focus_challenging")
	}

	// Rule: if scores are tightly clustered, increase exploration
	if len(scores) > 1 && variance(scores) < 0.01 {
		rules = append(rules, "increase_exploration")
	}

	// Rule: if best score is low, try different approaches
	bestScore := 0.0
	for _, sc := range scores {
		if sc > bestScore {
			bestScore = sc
		}
	}
	if bestScore < 0.5 {
		rules = append(rules, "diversify_prompts")
	}

	return rules
}

// generateImprovedCandidates creates new candidates informed by reflection rules
// and biased toward challenging examples.
func (s *SIMBA) generateImprovedCandidates(pool []simbaCandidate, challenging []optimize.Example, trainset []optimize.Example, rules []string, iteration int, rng *rand.Rand) []simbaCandidate {
	numNew := s.CandidatePoolSize / 2
	if numNew < 1 {
		numNew = 1
	}

	mutations := []string{
		"Be concise and precise.",
		"Explain your reasoning step by step.",
		"Consider the context carefully.",
		"Focus on the key information.",
		"Provide only the answer.",
	}

	newCandidates := make([]simbaCandidate, 0, numNew)

	for i := 0; i < numNew; i++ {
		// Select a parent via softmax from current pool
		parent := s.softmaxSelectOne(pool, rng)

		// Create child with mutated prompt
		child := simbaCandidate{
			ID:     fmt.Sprintf("iter%d_%d", iteration, i),
			Prompt: parent.Prompt + " " + mutations[rng.Intn(len(mutations))],
		}

		// Bias demos toward challenging examples if available and rules suggest it
		if len(challenging) > 0 && containsRule(rules, "focus_challenging") {
			child.Demos = s.sampleBiasedDemos(challenging, trainset, 4, rng)
		} else {
			child.Demos = s.sampleDemos(trainset, 4, rng)
		}

		newCandidates = append(newCandidates, child)
	}

	return newCandidates
}

// sampleBiasedDemos creates a demo set biased toward challenging examples.
// It includes at least one challenging example and fills the rest from the general trainset.
func (s *SIMBA) sampleBiasedDemos(challenging, trainset []optimize.Example, n int, rng *rand.Rand) []optimize.Example {
	demos := make([]optimize.Example, 0, n)

	// Include at least one challenging example
	numChallenging := 1
	if numChallenging > len(challenging) {
		numChallenging = len(challenging)
	}
	if numChallenging > n {
		numChallenging = n
	}

	perm := rng.Perm(len(challenging))
	for i := 0; i < numChallenging; i++ {
		demos = append(demos, challenging[perm[i]])
	}

	// Fill remaining from trainset
	remaining := n - len(demos)
	if remaining > 0 && len(trainset) > 0 {
		perm = rng.Perm(len(trainset))
		for i := 0; i < remaining && i < len(trainset); i++ {
			demos = append(demos, trainset[perm[i]])
		}
	}

	return demos
}

// softmaxSelect selects n candidates from the pool using temperature-controlled softmax sampling.
// Candidates with higher scores have higher selection probability.
// P(candidate_i) = exp(score_i / temperature) / sum(exp(score_j / temperature))
func (s *SIMBA) softmaxSelect(candidates []simbaCandidate, n int, rng *rand.Rand) []simbaCandidate {
	if len(candidates) <= n {
		return candidates
	}

	probs := s.softmaxProbabilities(candidates)

	selected := make([]simbaCandidate, 0, n)
	used := make(map[int]bool)

	for len(selected) < n {
		idx := weightedSample(probs, rng)
		if used[idx] {
			// If already selected, try next best
			for j := range candidates {
				if !used[j] {
					idx = j
					break
				}
			}
		}
		used[idx] = true
		selected = append(selected, candidates[idx])
	}

	return selected
}

// softmaxSelectOne selects a single candidate using softmax sampling.
func (s *SIMBA) softmaxSelectOne(candidates []simbaCandidate, rng *rand.Rand) simbaCandidate {
	if len(candidates) == 1 {
		return candidates[0]
	}

	probs := s.softmaxProbabilities(candidates)
	idx := weightedSample(probs, rng)
	return candidates[idx]
}

// softmaxProbabilities computes softmax probabilities for the candidate pool.
func (s *SIMBA) softmaxProbabilities(candidates []simbaCandidate) []float64 {
	scores := make([]float64, len(candidates))
	for i, c := range candidates {
		scores[i] = c.Score
	}
	return softmax(scores, s.SamplingTemperature)
}

// softmax computes softmax probabilities: exp(x_i/temp) / sum(exp(x_j/temp)).
// Uses the log-sum-exp trick for numerical stability.
func softmax(scores []float64, temperature float64) []float64 {
	if len(scores) == 0 {
		return nil
	}

	if temperature <= 0 {
		temperature = 0.01 // Prevent division by zero
	}

	// Find max for numerical stability (log-sum-exp trick)
	maxScore := scores[0]
	for _, s := range scores[1:] {
		if s > maxScore {
			maxScore = s
		}
	}

	// Compute exp((score - max) / temperature)
	exps := make([]float64, len(scores))
	sum := 0.0
	for i, s := range scores {
		exps[i] = math.Exp((s - maxScore) / temperature)
		sum += exps[i]
	}

	// Normalize
	probs := make([]float64, len(scores))
	for i := range exps {
		if sum > 0 {
			probs[i] = exps[i] / sum
		} else {
			probs[i] = 1.0 / float64(len(scores))
		}
	}

	return probs
}

// weightedSample draws a single index from a probability distribution.
func weightedSample(probs []float64, rng *rand.Rand) int {
	r := rng.Float64()
	cumulative := 0.0
	for i, p := range probs {
		cumulative += p
		if r <= cumulative {
			return i
		}
	}
	return len(probs) - 1
}

// variance computes the population variance of a slice of values.
func variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	v := 0.0
	for _, val := range values {
		diff := val - mean
		v += diff * diff
	}
	return v / float64(len(values))
}

// containsRule checks if a rule exists in the rule set.
func containsRule(rules []string, rule string) bool {
	for _, r := range rules {
		if r == rule {
			return true
		}
	}
	return false
}

// sortCandidatesByScore sorts candidates by score in descending order.
func sortCandidatesByScore(candidates []simbaCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
}
