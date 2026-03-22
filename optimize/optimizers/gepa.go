package optimizers

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/pareto"
)

func init() {
	optimize.RegisterOptimizer("gepa", func(cfg optimize.OptimizerConfig) (optimize.Optimizer, error) {
		var opts []GEPAOption
		if cfg.LLM != nil {
			opts = append(opts, WithGEPALLMClient(cfg.LLM))
		}
		return NewGEPA(opts...), nil
	})
}

// GEPA implements the Genetic-Pareto Prompt Optimizer.
//
// GEPA evolves a population of prompt+demo candidates using genetic operators
// (selection, crossover, mutation) guided by multi-objective Pareto dominance.
// Objectives tracked simultaneously are:
//   - Accuracy:    metric score on the trainset (higher is better)
//   - Efficiency:  inverse of token usage (higher = fewer tokens used)
//   - Consistency: how reproducible results are across multiple runs
//
// The Pareto archive accumulates non-dominated candidates across all generations.
// Reflection uses an LLM (when available) to analyze failure cases and produce
// targeted prompt mutations, closing the feedback loop.
type GEPA struct {
	// PopulationSize is the number of candidates maintained per generation.
	PopulationSize int

	// MaxGenerations is the maximum number of evolution generations.
	MaxGenerations int

	// MutationRate is the probability of mutation per candidate (0.0–1.0).
	MutationRate float64

	// CrossoverRate is the probability of crossover being applied (0.0–1.0).
	CrossoverRate float64

	// ArchiveSize is the maximum size of the Pareto archive.
	ArchiveSize int

	// ReflectionInterval is how many generations between LLM reflection passes.
	ReflectionInterval int

	// EvalSampleSize is the number of training examples used to score each candidate.
	// Smaller values speed up evaluation; 0 means use all examples.
	EvalSampleSize int

	// ConsistencyRuns is the number of repeated evaluations used to compute the
	// consistency objective. Must be >= 1; higher values improve accuracy but cost more.
	ConsistencyRuns int

	// ConvergenceWindow is how many recent best scores to track for early stopping.
	ConvergenceWindow int

	// ConvergenceThreshold is the variance threshold below which evolution stops early.
	ConvergenceThreshold float64

	// NumWorkers controls parallel candidate evaluation (default: 1, safe for -race).
	NumWorkers int

	// Seed for reproducibility.
	Seed int64

	// llm is an optional LLM client used for reflection-based mutation.
	llm optimize.LLMClient
}

// GEPAOption configures a GEPA optimizer.
type GEPAOption func(*GEPA)

// NewGEPA creates a new GEPA optimizer with sensible defaults.
func NewGEPA(opts ...GEPAOption) *GEPA {
	g := &GEPA{
		PopulationSize:       10,
		MaxGenerations:       10,
		MutationRate:         0.3,
		CrossoverRate:        0.5,
		ArchiveSize:          50,
		ReflectionInterval:   3,
		EvalSampleSize:       10,
		ConsistencyRuns:      2,
		ConvergenceWindow:    5,
		ConvergenceThreshold: 1e-4,
		NumWorkers:           1,
		Seed:                 42,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// WithPopulationSize sets the population size.
func WithPopulationSize(n int) GEPAOption {
	return func(g *GEPA) {
		if n > 0 {
			g.PopulationSize = n
		}
	}
}

// WithMaxGenerations sets the maximum number of generations.
func WithMaxGenerations(n int) GEPAOption {
	return func(g *GEPA) {
		if n > 0 {
			g.MaxGenerations = n
		}
	}
}

// WithMutationRate sets the mutation rate.
func WithMutationRate(rate float64) GEPAOption {
	return func(g *GEPA) {
		if rate >= 0 && rate <= 1 {
			g.MutationRate = rate
		}
	}
}

// WithCrossoverRate sets the crossover rate.
func WithCrossoverRate(rate float64) GEPAOption {
	return func(g *GEPA) {
		if rate >= 0 && rate <= 1 {
			g.CrossoverRate = rate
		}
	}
}

// WithGEPASeed sets the random seed for reproducibility.
func WithGEPASeed(seed int64) GEPAOption {
	return func(g *GEPA) {
		g.Seed = seed
	}
}

// WithGEPAArchiveSize sets the maximum Pareto archive size.
func WithGEPAArchiveSize(n int) GEPAOption {
	return func(g *GEPA) {
		if n > 0 {
			g.ArchiveSize = n
		}
	}
}

// WithGEPAReflectionInterval sets how many generations elapse between LLM reflection passes.
func WithGEPAReflectionInterval(n int) GEPAOption {
	return func(g *GEPA) {
		if n > 0 {
			g.ReflectionInterval = n
		}
	}
}

// WithGEPAEvalSampleSize sets how many training examples are sampled per candidate evaluation.
// Use 0 to evaluate on the full trainset.
func WithGEPAEvalSampleSize(n int) GEPAOption {
	return func(g *GEPA) {
		if n >= 0 {
			g.EvalSampleSize = n
		}
	}
}

// WithGEPAConsistencyRuns sets the number of repeated evaluations per candidate used to
// compute the consistency objective.
func WithGEPAConsistencyRuns(n int) GEPAOption {
	return func(g *GEPA) {
		if n >= 1 {
			g.ConsistencyRuns = n
		}
	}
}

// WithGEPANumWorkers sets the number of workers for parallel candidate evaluation.
func WithGEPANumWorkers(n int) GEPAOption {
	return func(g *GEPA) {
		if n > 0 {
			g.NumWorkers = n
		}
	}
}

// WithGEPALLMClient wires an LLM client for reflection-based mutation.
func WithGEPALLMClient(llm optimize.LLMClient) GEPAOption {
	return func(g *GEPA) {
		g.llm = llm
	}
}

// WithGEPAConvergenceWindow sets the number of recent best scores checked for convergence.
func WithGEPAConvergenceWindow(n int) GEPAOption {
	return func(g *GEPA) {
		if n >= 2 {
			g.ConvergenceWindow = n
		}
	}
}

// WithGEPAConvergenceThreshold sets the variance threshold for early stopping.
func WithGEPAConvergenceThreshold(v float64) GEPAOption {
	return func(g *GEPA) {
		if v >= 0 {
			g.ConvergenceThreshold = v
		}
	}
}

// -------------------------------------------------------------------------
// Internal types
// -------------------------------------------------------------------------

// gepaCandidate is a single individual in the GEPA population.
type gepaCandidate struct {
	// ID uniquely identifies this candidate within the run.
	ID string

	// Prompt is the instruction text prepended to demos.
	Prompt string

	// Demos are the few-shot examples carried by this candidate.
	Demos []optimize.Example

	// Score is the primary (accuracy) objective value.
	Score float64

	// Generation is when this candidate was created.
	Generation int

	// Objectives holds all multi-objective values used by the Pareto archive:
	//   [0] accuracy     – metric score (higher is better)
	//   [1] efficiency   – 1 / (1 + avgTokens/1000)  (higher = fewer tokens)
	//   [2] consistency  – 1 - stdDev(scores)         (higher = more consistent)
	Objectives []float64

	// evalMetrics stores raw per-run scores for computing consistency.
	evalScores []float64

	// avgTokens is the mean token usage across evaluation runs.
	avgTokens float64
}

// -------------------------------------------------------------------------
// Compile — main entry point
// -------------------------------------------------------------------------

// Compile implements optimize.Optimizer.
func (g *GEPA) Compile(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) (optimize.Program, error) {
	if opts.Metric == nil {
		return nil, fmt.Errorf("gepa: metric is required")
	}
	if len(opts.Trainset) == 0 {
		return nil, fmt.Errorf("gepa: trainset is required")
	}

	rng := rand.New(rand.NewSource(g.Seed))

	// Determine evaluation sample size.
	sampleSize := g.EvalSampleSize
	if sampleSize == 0 || sampleSize > len(opts.Trainset) {
		sampleSize = len(opts.Trainset)
	}

	// Pareto archive shared across all generations.
	archive := pareto.NewArchive(g.ArchiveSize)

	// Initialise population.
	population := g.initializePopulation(program, opts.Trainset, rng)

	// Track best scores per generation for convergence detection.
	convergence := &optimize.ConvergenceChecker{
		WindowSize: g.ConvergenceWindow,
		Threshold:  g.ConvergenceThreshold,
	}

	// Evolution loop.
	for generation := 0; generation < g.MaxGenerations; generation++ {
		// Check context cancellation.
		select {
		case <-ctx.Done():
			break
		default:
		}

		// Evaluate population (possibly in parallel).
		g.evaluatePopulation(ctx, program, population, opts, sampleSize, generation)

		// Add evaluated candidates to the Pareto archive.
		for _, c := range population {
			if len(c.Objectives) == 0 {
				continue
			}
			archive.Add(pareto.Point{
				ID:         c.ID,
				Objectives: c.Objectives,
				Payload:    c,
			}, generation)
		}

		// Report best score this generation.
		bestScore := bestScoreInPopulation(population)

		// Notify callbacks.
		for _, cb := range opts.Callbacks {
			cb.OnTrialComplete(optimize.Trial{
				ID:    generation,
				Score: bestScore,
			})
		}

		// Check early termination.
		if generation >= g.MaxGenerations-1 {
			break
		}
		if convergence.Update(bestScore) {
			break
		}

		// Check cost budget.
		if opts.MaxCost != nil && opts.MaxCost.Exceeded(0, 0, generation+1) {
			break
		}

		// Selection → Crossover → Mutation.
		parents := g.tournamentSelect(population, g.PopulationSize, rng)
		offspring := g.crossover(parents, generation+1, rng)
		g.mutate(offspring, opts.Trainset, rng)

		// LLM reflection every N generations.
		if (generation+1)%g.ReflectionInterval == 0 {
			g.reflect(ctx, archive, offspring, generation, rng)
		}

		population = offspring
	}

	// Select best from the Pareto archive.
	best := g.selectBestFromArchive(archive)
	if best == nil {
		return nil, fmt.Errorf("gepa: no valid candidate found")
	}

	return program.WithDemos(best.Demos), nil
}

// -------------------------------------------------------------------------
// Population initialisation
// -------------------------------------------------------------------------

// basePrompts is the seed instruction pool used to initialise the population.
var basePrompts = []string{
	"Answer the question based on the context provided.",
	"Provide a concise, accurate answer.",
	"Think step by step, then give your final answer.",
	"Based on the information given:",
	"The answer is:",
	"Analyze the question carefully before responding.",
	"Use the examples to guide your answer.",
	"Respond with precision and clarity.",
	"Draw on the context to formulate your response.",
	"Provide only the requested information.",
}

// initializePopulation creates the initial population by sampling demos from the trainset.
func (g *GEPA) initializePopulation(program optimize.Program, trainset []optimize.Example, rng *rand.Rand) []gepaCandidate {
	candidates := make([]gepaCandidate, 0, g.PopulationSize)

	numDemos := 4
	if len(trainset) < numDemos {
		numDemos = len(trainset)
	}

	for i := 0; i < g.PopulationSize; i++ {
		demos := make([]optimize.Example, numDemos)
		for j := 0; j < numDemos; j++ {
			demos[j] = trainset[rng.Intn(len(trainset))]
		}

		candidates = append(candidates, gepaCandidate{
			ID:     fmt.Sprintf("gen0_%d", i),
			Prompt: basePrompts[i%len(basePrompts)],
			Demos:  demos,
		})
	}

	return candidates
}

// -------------------------------------------------------------------------
// Evaluation
// -------------------------------------------------------------------------

// evaluatePopulation evaluates all candidates, optionally in parallel.
func (g *GEPA) evaluatePopulation(
	ctx context.Context,
	program optimize.Program,
	population []gepaCandidate,
	opts optimize.CompileOptions,
	sampleSize int,
	generation int,
) {
	workers := g.NumWorkers
	if workers <= 1 {
		// Sequential path — no goroutines, safe for race detector.
		for i := range population {
			g.evalOneCandidate(ctx, program, &population[i], opts, sampleSize, generation)
		}
		return
	}

	// Parallel path.
	type work struct {
		idx int
	}
	jobs := make(chan work, len(population))
	for i := range population {
		jobs <- work{i}
	}
	close(jobs)

	var wg sync.WaitGroup
	var mu sync.Mutex
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Work on a local copy.
				c := population[job.idx]
				g.evalOneCandidate(ctx, program, &c, opts, sampleSize, generation)
				mu.Lock()
				population[job.idx] = c
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

// evalOneCandidate evaluates a single candidate and populates its objective values.
func (g *GEPA) evalOneCandidate(
	ctx context.Context,
	program optimize.Program,
	c *gepaCandidate,
	opts optimize.CompileOptions,
	sampleSize int,
	generation int,
) {
	progWithDemos := program.WithDemos(c.Demos)

	// Build a deterministic sample using the candidate seed + generation.
	sampleRng := rand.New(rand.NewSource(g.Seed + int64(generation*1000)))

	// Run ConsistencyRuns evaluations and collect per-run scores.
	allRunScores := make([]float64, 0, g.ConsistencyRuns)
	var totalTokens float64

	for run := 0; run < g.ConsistencyRuns; run++ {
		var runScore float64
		var runTokens float64
		evaluated := 0

		for i := 0; i < sampleSize; i++ {
			idx := sampleRng.Intn(len(opts.Trainset))
			ex := opts.Trainset[idx]

			pred, err := progWithDemos.Run(ctx, ex.Inputs)
			if err != nil {
				continue
			}

			score, err := opts.Metric.Evaluate(ex, pred, nil)
			if err != nil {
				continue
			}

			runScore += score
			runTokens += float64(pred.Usage.TotalTokens)
			evaluated++
		}

		if evaluated > 0 {
			allRunScores = append(allRunScores, runScore/float64(evaluated))
			totalTokens += runTokens / float64(evaluated)
		}
	}

	if len(allRunScores) == 0 {
		c.Objectives = []float64{0, 0, 0}
		c.Score = 0
		c.Generation = generation
		return
	}

	// Objective 0: accuracy (mean score across runs).
	accuracy := mean(allRunScores)

	// Objective 1: efficiency (fewer tokens → higher score).
	avgTok := totalTokens / float64(len(allRunScores))
	efficiency := 1.0 / (1.0 + avgTok/1000.0)

	// Objective 2: consistency (lower std-dev → higher score).
	sd := stddev(allRunScores)
	consistency := 1.0 - sd
	if consistency < 0 {
		consistency = 0
	}

	c.Score = accuracy
	c.avgTokens = avgTok
	c.evalScores = allRunScores
	c.Generation = generation
	c.Objectives = []float64{accuracy, efficiency, consistency}
}

// -------------------------------------------------------------------------
// Genetic operators
// -------------------------------------------------------------------------

// tournamentSelect performs tournament selection to choose parents for the next generation.
func (g *GEPA) tournamentSelect(population []gepaCandidate, n int, rng *rand.Rand) []gepaCandidate {
	const tournamentSize = 3
	parents := make([]gepaCandidate, 0, n)

	for i := 0; i < n; i++ {
		best := population[rng.Intn(len(population))]
		for j := 1; j < tournamentSize; j++ {
			candidate := population[rng.Intn(len(population))]
			if candidate.Score > best.Score {
				best = candidate
			}
		}
		parents = append(parents, best)
	}

	return parents
}

// crossover produces offspring from a parent pool using uniform crossover.
// When crossover fires, each offspring inherits the prompt from one randomly
// chosen parent and interleaved demos from two parents.
func (g *GEPA) crossover(parents []gepaCandidate, generation int, rng *rand.Rand) []gepaCandidate {
	offspring := make([]gepaCandidate, 0, g.PopulationSize)

	for i := 0; i < g.PopulationSize; i++ {
		if rng.Float64() > g.CrossoverRate || len(parents) < 2 {
			// No crossover: inherit a single parent.
			parent := parents[i%len(parents)]
			child := gepaCandidate{
				ID:     fmt.Sprintf("gen%d_%d", generation, i),
				Prompt: parent.Prompt,
				Demos:  cloneDemos(parent.Demos),
			}
			offspring = append(offspring, child)
			continue
		}

		// Select two distinct parents.
		idxA := rng.Intn(len(parents))
		idxB := rng.Intn(len(parents))
		for idxB == idxA && len(parents) > 1 {
			idxB = rng.Intn(len(parents))
		}
		p1 := parents[idxA]
		p2 := parents[idxB]

		// Inherit prompt from the higher-scoring parent.
		prompt := p1.Prompt
		if p2.Score > p1.Score {
			prompt = p2.Prompt
		}

		// Interleave demos from both parents.
		maxDemos := len(p1.Demos)
		if len(p2.Demos) > maxDemos {
			maxDemos = len(p2.Demos)
		}
		demos := make([]optimize.Example, 0, maxDemos)
		for j := 0; j < maxDemos; j++ {
			if j%2 == 0 && j < len(p1.Demos) {
				demos = append(demos, p1.Demos[j])
			} else if j < len(p2.Demos) {
				demos = append(demos, p2.Demos[j])
			} else if j < len(p1.Demos) {
				demos = append(demos, p1.Demos[j])
			}
		}

		offspring = append(offspring, gepaCandidate{
			ID:     fmt.Sprintf("gen%d_%d_cross", generation, i),
			Prompt: prompt,
			Demos:  demos,
		})
	}

	return offspring
}

// promptMutations is the built-in set of instruction modifiers used when no LLM is available.
var promptMutations = []string{
	"Be concise.",
	"Explain your reasoning step by step.",
	"Provide only the final answer.",
	"Think carefully before responding.",
	"Therefore:",
	"Given the examples, answer precisely.",
	"Use logical deduction.",
	"Respond directly without preamble.",
	"Consider edge cases.",
	"Prioritize accuracy over speed.",
}

// mutate applies stochastic perturbations to offspring prompts and demos.
func (g *GEPA) mutate(offspring []gepaCandidate, trainset []optimize.Example, rng *rand.Rand) {
	for i := range offspring {
		if rng.Float64() > g.MutationRate {
			continue
		}

		// Prompt mutation: append, replace, or trim a modifier phrase.
		switch rng.Intn(3) {
		case 0: // Append modifier.
			offspring[i].Prompt = offspring[i].Prompt + " " + promptMutations[rng.Intn(len(promptMutations))]
		case 1: // Replace prompt with a base prompt.
			offspring[i].Prompt = basePrompts[rng.Intn(len(basePrompts))]
		case 2: // Trim to first sentence (simplify).
			if idx := strings.Index(offspring[i].Prompt, "."); idx > 0 {
				offspring[i].Prompt = offspring[i].Prompt[:idx+1]
			}
		}

		// Demo mutation: replace a random demo with a trainset example.
		if len(offspring[i].Demos) > 0 && len(trainset) > 0 {
			demoIdx := rng.Intn(len(offspring[i].Demos))
			trainIdx := rng.Intn(len(trainset))
			offspring[i].Demos[demoIdx] = trainset[trainIdx]
		}
	}
}

// -------------------------------------------------------------------------
// Reflection
// -------------------------------------------------------------------------

// reflect analyses the current Pareto archive and, if an LLM client is available,
// generates targeted prompt mutations for low-performing offspring.
func (g *GEPA) reflect(
	ctx context.Context,
	archive *pareto.Archive,
	offspring []gepaCandidate,
	generation int,
	rng *rand.Rand,
) {
	archivePoints := archive.Get()
	if len(archivePoints) == 0 {
		return
	}

	// Find failure candidates: those with accuracy below archive median.
	scores := make([]float64, 0, len(archivePoints))
	for _, p := range archivePoints {
		if c, ok := p.Payload.(gepaCandidate); ok {
			scores = append(scores, c.Score)
		}
	}
	sort.Float64s(scores)
	median := scores[len(scores)/2]

	// If we have an LLM client, use it to suggest improvements.
	if g.llm != nil {
		g.llmReflect(ctx, archive, offspring, median, generation, rng)
		return
	}

	// Heuristic fallback: inject a fresh high-diversity candidate into offspring.
	if len(offspring) > 0 {
		targetIdx := rng.Intn(len(offspring))
		if offspring[targetIdx].Score < median {
			offspring[targetIdx].Prompt = basePrompts[rng.Intn(len(basePrompts))] +
				" " + promptMutations[rng.Intn(len(promptMutations))]
		}
	}
}

// llmReflect uses the LLM to analyse archive failures and mutate low-scoring offspring.
func (g *GEPA) llmReflect(
	ctx context.Context,
	archive *pareto.Archive,
	offspring []gepaCandidate,
	medianScore float64,
	generation int,
	rng *rand.Rand,
) {
	// Collect a representative set of prompts from the archive frontier.
	archivePoints := archive.Get()
	var topPrompts []string
	for _, p := range archivePoints {
		if c, ok := p.Payload.(gepaCandidate); ok && c.Score >= medianScore {
			topPrompts = append(topPrompts, c.Prompt)
		}
		if len(topPrompts) >= 3 {
			break
		}
	}

	if len(topPrompts) == 0 {
		return
	}

	// Build reflection prompt.
	reflectionPrompt := fmt.Sprintf(
		"You are optimizing prompts for an AI system. Generation %d. "+
			"The following prompts performed well:\n%s\n\n"+
			"Suggest a single improved instruction (one sentence) that builds on these strengths. "+
			"Respond with only the instruction, no explanation.",
		generation,
		strings.Join(topPrompts, "\n- "),
	)

	suggestion, err := g.llm.Complete(ctx, reflectionPrompt, optimize.CompletionOptions{
		Temperature: 0.7,
		MaxTokens:   60,
	})
	if err != nil || strings.TrimSpace(suggestion) == "" {
		return
	}

	suggestion = strings.TrimSpace(suggestion)

	// Inject the LLM-suggested prompt into a low-scoring offspring slot.
	for i := range offspring {
		if offspring[i].Score < medianScore {
			offspring[i].Prompt = suggestion
			offspring[i].ID = fmt.Sprintf("gen%d_%d_reflect", generation, i)
			return
		}
	}

	// All offspring are above median — inject into a random slot.
	if len(offspring) > 0 {
		idx := rng.Intn(len(offspring))
		offspring[idx].Prompt = suggestion
		offspring[idx].ID = fmt.Sprintf("gen%d_%d_reflect", generation, idx)
	}
}

// -------------------------------------------------------------------------
// Archive selection
// -------------------------------------------------------------------------

// selectBestFromArchive picks the best candidate from the Pareto archive.
// Priority order:
//  1. Use SelectByCoverage on the archive's own frontier to pick the most
//     diverse non-dominated point.
//  2. Fall back to the candidate with the highest accuracy score.
func (g *GEPA) selectBestFromArchive(archive *pareto.Archive) *gepaCandidate {
	points := archive.Get()
	if len(points) == 0 {
		return nil
	}

	// Build a temporary frontier from current archive points for coverage selection.
	f := pareto.NewFrontier()
	for _, p := range points {
		f.Add(p)
	}

	frontierPoints := f.Get()
	if len(frontierPoints) == 0 {
		// Shouldn't happen, but guard anyway.
		return fallbackBestCandidate(points)
	}

	// Select the point with the best coverage contribution.
	indices := f.SelectByCoverage(1)
	if len(indices) > 0 {
		if c, ok := frontierPoints[indices[0]].Payload.(gepaCandidate); ok {
			return &c
		}
	}

	// Fallback: highest accuracy.
	return fallbackBestCandidate(points)
}

// fallbackBestCandidate returns the candidate with the highest accuracy from a point slice.
func fallbackBestCandidate(points []pareto.Point) *gepaCandidate {
	var best *gepaCandidate
	bestScore := -1.0
	for _, p := range points {
		if c, ok := p.Payload.(gepaCandidate); ok {
			if c.Score > bestScore {
				bestScore = c.Score
				cp := c
				best = &cp
			}
		}
	}
	return best
}

// -------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------

// bestScoreInPopulation returns the highest accuracy score in a population.
func bestScoreInPopulation(pop []gepaCandidate) float64 {
	best := -1.0
	for _, c := range pop {
		if c.Score > best {
			best = c.Score
		}
	}
	return best
}

// mean computes the arithmetic mean of a float64 slice.
func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// stddev computes the population standard deviation of a float64 slice.
func stddev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := mean(xs)
	v := 0.0
	for _, x := range xs {
		d := x - m
		v += d * d
	}
	v /= float64(len(xs))
	// integer sqrt approximation — avoid math import; use Newton's method.
	return math.Sqrt(v)
}

// cloneDemos makes a shallow copy of a demo slice.
func cloneDemos(demos []optimize.Example) []optimize.Example {
	if demos == nil {
		return nil
	}
	cp := make([]optimize.Example, len(demos))
	copy(cp, demos)
	return cp
}
