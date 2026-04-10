package redteam

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
)

// runnerOptions holds configuration for a RedTeamRunner.
type runnerOptions struct {
	target    agent.Agent
	patterns  []string
	generator *AttackGenerator
	parallel  int
	timeout   time.Duration
	hooks     Hooks
	scorer    *DefenseScorer
}

// RunnerOption configures a RedTeamRunner.
type RunnerOption func(*runnerOptions)

// WithTarget sets the agent to test.
func WithTarget(a agent.Agent) RunnerOption {
	return func(o *runnerOptions) {
		o.target = a
	}
}

// WithPatterns sets which registered attack patterns to use by name.
func WithPatterns(names ...string) RunnerOption {
	return func(o *runnerOptions) {
		o.patterns = names
	}
}

// WithGenerator sets the dynamic attack generator.
func WithGenerator(g *AttackGenerator) RunnerOption {
	return func(o *runnerOptions) {
		o.generator = g
	}
}

// WithParallel sets the number of attacks to run concurrently.
func WithParallel(n int) RunnerOption {
	return func(o *runnerOptions) {
		if n > 0 {
			o.parallel = n
		}
	}
}

// WithTimeout sets the maximum duration for the entire red team exercise.
func WithTimeout(d time.Duration) RunnerOption {
	return func(o *runnerOptions) {
		o.timeout = d
	}
}

// WithHooks sets the lifecycle hooks for the red team exercise.
func WithHooks(h Hooks) RunnerOption {
	return func(o *runnerOptions) {
		o.hooks = h
	}
}

// WithScorer sets a custom DefenseScorer. If not provided, NewDefenseScorer()
// is used.
func WithScorer(s *DefenseScorer) RunnerOption {
	return func(o *runnerOptions) {
		o.scorer = s
	}
}

// RedTeamRunner orchestrates a red team exercise against a target agent.
type RedTeamRunner struct {
	opts runnerOptions
}

// NewRunner creates a new RedTeamRunner with the given options.
func NewRunner(opts ...RunnerOption) *RedTeamRunner {
	o := runnerOptions{
		parallel: 1,
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.scorer == nil {
		o.scorer = NewDefenseScorer()
	}
	return &RedTeamRunner{opts: o}
}

// attackItem pairs a category with a prompt for execution.
type attackItem struct {
	category AttackCategory
	prompt   string
}

// Run executes the red team exercise and returns a report.
func (r *RedTeamRunner) Run(ctx context.Context) (*RedTeamReport, error) {
	if r.opts.target == nil {
		return nil, fmt.Errorf("redteam: target agent is required (use WithTarget)")
	}

	if r.opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.opts.timeout)
		defer cancel()
	}

	start := time.Now()

	// Collect all attack prompts.
	attacks, err := r.collectAttacks(ctx)
	if err != nil {
		return nil, fmt.Errorf("redteam: collect attacks: %w", err)
	}

	if len(attacks) == 0 {
		return nil, fmt.Errorf("redteam: no attack prompts generated (configure patterns or generator)")
	}

	// Execute attacks with bounded concurrency.
	results := r.executeAttacks(ctx, attacks)

	// Build report.
	report := r.buildReport(results, time.Since(start))
	return report, nil
}

// collectAttacks gathers attack prompts from registered patterns and the generator.
func (r *RedTeamRunner) collectAttacks(ctx context.Context) ([]attackItem, error) {
	var attacks []attackItem

	// Collect from registered patterns.
	for _, name := range r.opts.patterns {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		pattern, err := NewPattern(name)
		if err != nil {
			return nil, err
		}

		prompts, err := pattern.Generate(ctx)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", name, err)
		}

		for _, prompt := range prompts {
			attacks = append(attacks, attackItem{
				category: pattern.Category(),
				prompt:   prompt,
			})
		}
	}

	// Collect from generator if configured.
	if r.opts.generator != nil {
		generated, err := r.opts.generator.Generate(ctx)
		if err != nil {
			return nil, fmt.Errorf("generator: %w", err)
		}
		for cat, prompts := range generated {
			for _, prompt := range prompts {
				attacks = append(attacks, attackItem{
					category: cat,
					prompt:   prompt,
				})
			}
		}
	}

	return attacks, nil
}

// executeAttacks runs all attacks against the target with bounded concurrency.
func (r *RedTeamRunner) executeAttacks(ctx context.Context, attacks []attackItem) []AttackResult {
	results := make([]AttackResult, len(attacks))
	sem := make(chan struct{}, r.opts.parallel)
	var wg sync.WaitGroup

	for i, atk := range attacks {
		if ctx.Err() != nil {
			// Fill remaining with context error results.
			for j := i; j < len(attacks); j++ {
				results[j] = AttackResult{
					Category: attacks[j].category,
					Prompt:   attacks[j].prompt,
					Success:  false,
					Severity: SeverityLow,
					Details:  "skipped: context cancelled",
				}
			}
			break
		}

		wg.Add(1)
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Done()
			for j := i; j < len(attacks); j++ {
				results[j] = AttackResult{
					Category: attacks[j].category,
					Prompt:   attacks[j].prompt,
					Success:  false,
					Severity: SeverityLow,
					Details:  "skipped: context cancelled",
				}
			}
			wg.Wait()
			return results
		}

		go func(idx int, item attackItem) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = r.executeOneAttack(ctx, item)
		}(i, atk)
	}

	wg.Wait()
	return results
}

// executeOneAttack runs a single attack, invoking hooks and scoring.
func (r *RedTeamRunner) executeOneAttack(ctx context.Context, item attackItem) AttackResult {
	// BeforeAttack hook.
	if r.opts.hooks.BeforeAttack != nil {
		if err := r.opts.hooks.BeforeAttack(ctx, item.category, item.prompt); err != nil {
			return AttackResult{
				Category: item.category,
				Prompt:   item.prompt,
				Success:  false,
				Severity: SeverityLow,
				Details:  fmt.Sprintf("BeforeAttack hook error: %v", err),
			}
		}
	}

	// Invoke the target agent.
	response, err := r.opts.target.Invoke(ctx, item.prompt)
	if err != nil {
		// Agent error is treated as a defense (the agent refused or the guard blocked).
		response = fmt.Sprintf("error: %v", err)
	}

	// Score the response.
	result := r.opts.scorer.Score(item.category, item.prompt, response)

	// AfterAttack hook.
	if r.opts.hooks.AfterAttack != nil {
		r.opts.hooks.AfterAttack(ctx, result)
	}

	// OnVulnerabilityFound hook.
	if result.Success && r.opts.hooks.OnVulnerabilityFound != nil {
		r.opts.hooks.OnVulnerabilityFound(ctx, result)
	}

	return result
}

// buildReport aggregates attack results into a RedTeamReport.
func (r *RedTeamRunner) buildReport(results []AttackResult, duration time.Duration) *RedTeamReport {
	report := &RedTeamReport{
		Results:        results,
		CategoryScores: make(map[AttackCategory]float64),
		Timestamp:      time.Now(),
		Duration:       duration,
		TotalAttacks:   len(results),
	}

	// Count attacks and defenses per category.
	catTotal := make(map[AttackCategory]int)
	catDefended := make(map[AttackCategory]int)

	for _, res := range results {
		catTotal[res.Category]++
		if res.Success {
			report.SuccessfulAttacks++
		} else {
			catDefended[res.Category]++
		}
	}

	// Compute per-category defense scores.
	for cat, total := range catTotal {
		if total > 0 {
			report.CategoryScores[cat] = float64(catDefended[cat]) / float64(total)
		}
	}

	// Compute overall defense score.
	if report.TotalAttacks > 0 {
		defended := report.TotalAttacks - report.SuccessfulAttacks
		report.OverallScore = float64(defended) / float64(report.TotalAttacks)
	}

	return report
}
