package optimizers

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/pareto"
)

func init() {
	optimize.RegisterOptimizer("gepa", func(cfg optimize.OptimizerConfig) (optimize.Optimizer, error) {
		return NewGEPA(), nil
	})
}

// GEPA implements the Genetic-Pareto Prompt Optimizer.
// It uses evolutionary search with Pareto frontier selection and natural language reflection.
type GEPA struct {
	// PopulationSize is the number of candidates to maintain.
	PopulationSize int

	// MaxGenerations is the maximum number of evolution generations.
	MaxGenerations int

	// MutationRate is the probability of mutation (0.0-1.0).
	MutationRate float64

	// CrossoverRate is the probability of crossover (0.0-1.0).
	CrossoverRate float64

	// ArchiveSize is the maximum size of the Pareto archive.
	ArchiveSize int

	// ReflectionInterval is how often to perform LLM reflection (in generations).
	ReflectionInterval int

	// Seed for reproducibility.
	Seed int64
}

// GEPAOption configures a GEPA optimizer.
type GEPAOption func(*GEPA)

// NewGEPA creates a new GEPA optimizer with defaults.
func NewGEPA(opts ...GEPAOption) *GEPA {
	g := &GEPA{
		PopulationSize:     10,
		MaxGenerations:     10,
		MutationRate:       0.3,
		CrossoverRate:      0.5,
		ArchiveSize:        50,
		ReflectionInterval: 3,
		Seed:               42,
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

// WithGEPASeed sets the random seed.
func WithGEPASeed(seed int64) GEPAOption {
	return func(g *GEPA) {
		g.Seed = seed
	}
}

// Candidate represents a GEPA candidate solution.
type gepaCandidate struct {
	ID         string
	Prompt     string
	Demos      []optimize.Example
	Score      float64
	Generation int
	Objectives []float64 // Multi-objective: [accuracy, cost, latency, etc.]
}

// Compile implements optimize.Optimizer.
func (g *GEPA) Compile(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) (optimize.Program, error) {
	if opts.Metric == nil {
		return nil, fmt.Errorf("metric is required")
	}

	if len(opts.Trainset) == 0 {
		return nil, fmt.Errorf("trainset is required")
	}

	rng := rand.New(rand.NewSource(g.Seed))

	// Initialize Pareto archive
	archive := pareto.NewArchive(g.ArchiveSize)

	// Initialize population
	population := g.initializePopulation(program, opts.Trainset, rng)

	// Evolution loop
	for generation := 0; generation < g.MaxGenerations; generation++ {
		// Evaluate population
		for i := range population {
			score, err := g.evaluateCandidate(ctx, program, population[i], opts)
			if err != nil {
				continue
			}
			population[i].Score = score
			population[i].Generation = generation
			// For now, single objective: accuracy
			population[i].Objectives = []float64{score}
		}

		// Add to Pareto archive
		for _, c := range population {
			p := pareto.Point{
				ID:         c.ID,
				Objectives: c.Objectives,
				Payload:    c,
			}
			archive.Add(p, generation)
		}

		// Check termination conditions
		if generation >= g.MaxGenerations-1 {
			break
		}

		// Selection (tournament)
		parents := g.tournamentSelect(population, g.PopulationSize, rng)

		// Crossover
		offspring := g.crossover(parents, rng)

		// Mutation
		g.mutate(offspring, opts.Trainset, rng)

		// Reflection (every N generations)
		if (generation+1)%g.ReflectionInterval == 0 {
			g.reflect(ctx, archive, generation)
		}

		// Replace population
		population = offspring

		// Check cost budget
		if opts.MaxCost != nil {
			// Approximate cost check
			if float64(generation) > opts.MaxCost.MaxDollars {
				break
			}
		}
	}

	// Return best from archive
	best := g.selectBestFromArchive(archive)
	if best == nil {
		return nil, fmt.Errorf("no valid candidate found")
	}

	return program.WithDemos(best.Demos), nil
}

// initializePopulation creates the initial population.
func (g *GEPA) initializePopulation(program optimize.Program, trainset []optimize.Example, rng *rand.Rand) []gepaCandidate {
	candidates := make([]gepaCandidate, 0, g.PopulationSize)

	// Create initial prompts
	basePrompts := []string{
		"Answer the question based on the context provided.",
		"Provide a concise answer.",
		"What is the answer?",
		"Based on the information:",
		"The answer is:",
	}

	for i := 0; i < g.PopulationSize; i++ {
		// Sample demos
		demos := make([]optimize.Example, 0, 4)
		for j := 0; j < 4 && j < len(trainset); j++ {
			idx := rng.Intn(len(trainset))
			demos = append(demos, trainset[idx])
		}

		candidates = append(candidates, gepaCandidate{
			ID:     fmt.Sprintf("gen0_%d", i),
			Prompt: basePrompts[i%len(basePrompts)],
			Demos:  demos,
		})
	}

	return candidates
}

// evaluateCandidate evaluates a candidate on the trainset.
func (g *GEPA) evaluateCandidate(ctx context.Context, program optimize.Program, c gepaCandidate, opts optimize.CompileOptions) (float64, error) {
	progWithDemos := program.WithDemos(c.Demos)

	var totalScore float64
	sampleSize := 10
	if len(opts.Trainset) < sampleSize {
		sampleSize = len(opts.Trainset)
	}

	// Sample for efficiency
	rng := rand.New(rand.NewSource(g.Seed))
	for i := 0; i < sampleSize; i++ {
		idx := rng.Intn(len(opts.Trainset))
		ex := opts.Trainset[idx]

		pred, err := progWithDemos.Run(ctx, ex.Inputs)
		if err != nil {
			continue
		}

		score, err := opts.Metric.Evaluate(ex, pred, nil)
		if err != nil {
			continue
		}

		totalScore += score
	}

	if sampleSize == 0 {
		return 0.0, nil
	}

	return totalScore / float64(sampleSize), nil
}

// tournamentSelect selects parents using tournament selection.
func (g *GEPA) tournamentSelect(population []gepaCandidate, n int, rng *rand.Rand) []gepaCandidate {
	parents := make([]gepaCandidate, 0, n)
	tournamentSize := 3

	for i := 0; i < n; i++ {
		// Tournament
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

// crossover performs crossover between parents.
func (g *GEPA) crossover(parents []gepaCandidate, rng *rand.Rand) []gepaCandidate {
	offspring := make([]gepaCandidate, 0, g.PopulationSize)

	for i := 0; i < g.PopulationSize; i++ {
		if rng.Float64() > g.CrossoverRate {
			// No crossover, copy parent
			offspring = append(offspring, parents[i%len(parents)])
			continue
		}

		// Crossover between two random parents
		p1 := parents[rng.Intn(len(parents))]
		p2 := parents[rng.Intn(len(parents))]

		child := gepaCandidate{
			ID:     fmt.Sprintf("cross_%d", i),
			Prompt: p1.Prompt, // Take prompt from parent 1
		}

		// Combine demos (50/50 from each parent)
		child.Demos = make([]optimize.Example, 0, 4)
		half := len(p1.Demos) / 2
		if half > len(p1.Demos) {
			half = len(p1.Demos)
		}
		child.Demos = append(child.Demos, p1.Demos[:half]...)
		if half < len(p2.Demos) {
			child.Demos = append(child.Demos, p2.Demos[half:]...)
		}

		offspring = append(offspring, child)
	}

	return offspring
}

// mutate performs mutation on offspring.
func (g *GEPA) mutate(offspring []gepaCandidate, trainset []optimize.Example, rng *rand.Rand) {
	mutations := []string{
		"Be concise.",
		"Explain your reasoning.",
		"Provide only the answer.",
		"Step by step:",
		"Therefore:",
	}

	for i := range offspring {
		if rng.Float64() > g.MutationRate {
			continue
		}

		// Mutate prompt
		if rng.Float64() < 0.5 {
			// Append mutation instruction
			offspring[i].Prompt = offspring[i].Prompt + " " + mutations[rng.Intn(len(mutations))]
		}

		// Mutate demos (replace one)
		if len(offspring[i].Demos) > 0 && len(trainset) > 0 {
			demoIdx := rng.Intn(len(offspring[i].Demos))
			trainIdx := rng.Intn(len(trainset))
			offspring[i].Demos[demoIdx] = trainset[trainIdx]
		}
	}
}

// reflect performs LLM reflection on the archive.
func (g *GEPA) reflect(ctx context.Context, archive *pareto.Archive, generation int) {
	// In a full implementation, this would:
	// 1. Analyze failures from the archive
	// 2. Use LLM to suggest improvements
	// 3. Generate new candidates based on insights
	
	// For now, this is a placeholder
	_ = generation
	_ = archive.Get()
}

// selectBestFromArchive selects the best candidate from the archive.
func (g *GEPA) selectBestFromArchive(archive *pareto.Archive) *gepaCandidate {
	points := archive.Get()
	if len(points) == 0 {
		return nil
	}

	// Select by coverage
	indices := (&pareto.Frontier{}).SelectByCoverage(1)
	if len(indices) == 0 {
		// Fallback: select highest scoring
		var best *gepaCandidate
		bestScore := -1.0
		for _, p := range points {
			if c, ok := p.Payload.(gepaCandidate); ok {
				if c.Score > bestScore {
					bestScore = c.Score
					best = &c
				}
			}
		}
		return best
	}

	if c, ok := points[indices[0]].Payload.(gepaCandidate); ok {
		return &c
	}
	return nil
}
