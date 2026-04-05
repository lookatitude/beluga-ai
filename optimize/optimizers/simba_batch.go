package optimizers

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/optimize"
)

// BatchEvaluator provides memory-efficient, optionally concurrent batch evaluation
// of candidates over large example sets.
//
// Instead of loading the entire dataset into memory at once, BatchEvaluator
// processes examples in fixed-size chunks, making it suitable for large training
// sets that cannot fit in memory. The chunked approach also allows garbage
// collection of processed chunks, keeping peak memory usage bounded by
// ChunkSize rather than the total dataset size.
//
// Concurrency model: each chunk is evaluated sequentially within the goroutine
// that owns it, but multiple chunks are processed concurrently when NumWorkers > 1.
// A sync.Mutex protects score accumulators so the struct is safe for parallel use.
type BatchEvaluator struct {
	// ChunkSize is the number of examples processed per chunk.
	// Smaller values reduce peak memory; larger values reduce scheduling overhead.
	ChunkSize int

	// NumWorkers controls the number of concurrent chunk goroutines.
	// Defaults to 1 (sequential). Values > 1 enable parallel chunk processing.
	NumWorkers int
}

// BatchEvaluatorOption configures a BatchEvaluator.
type BatchEvaluatorOption func(*BatchEvaluator)

// NewBatchEvaluator creates a BatchEvaluator with sensible defaults.
func NewBatchEvaluator(opts ...BatchEvaluatorOption) *BatchEvaluator {
	b := &BatchEvaluator{
		ChunkSize:  50,
		NumWorkers: 1,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// WithBatchChunkSize sets the number of examples per processing chunk.
func WithBatchChunkSize(n int) BatchEvaluatorOption {
	return func(b *BatchEvaluator) {
		if n > 0 {
			b.ChunkSize = n
		}
	}
}

// WithBatchNumWorkers sets the number of parallel chunk workers.
func WithBatchNumWorkers(n int) BatchEvaluatorOption {
	return func(b *BatchEvaluator) {
		if n > 0 {
			b.NumWorkers = n
		}
	}
}

// chunkResult holds the accumulated score for a single chunk.
type chunkResult struct {
	totalScore float64
	count      int
	err        error
}

// EvaluateAll scores a single candidate over all examples in a memory-efficient,
// chunked fashion. The examples slice is not copied; individual chunk slices are
// views into it, so no additional heap allocation occurs for the examples
// themselves.
//
// When NumWorkers > 1, chunks are dispatched to a worker pool via a buffered
// channel. A sync.WaitGroup ensures all workers complete before results are
// aggregated. The approach is race-free because each goroutine writes into its
// own chunkResult value, and the results channel is read only after all workers
// have finished.
func (b *BatchEvaluator) EvaluateAll(
	ctx context.Context,
	program optimize.Program,
	candidate simbaCandidate,
	examples []optimize.Example,
	metric optimize.Metric,
) (float64, error) {
	if len(examples) == 0 {
		return 0, nil
	}

	progWithDemos := program.WithDemos(candidate.Demos)

	// Build chunk boundaries without allocating the chunks themselves.
	type bounds struct{ lo, hi int }
	var chunks []bounds
	for lo := 0; lo < len(examples); lo += b.ChunkSize {
		hi := lo + b.ChunkSize
		if hi > len(examples) {
			hi = len(examples)
		}
		chunks = append(chunks, bounds{lo, hi})
	}

	numWorkers := b.NumWorkers
	if numWorkers < 1 {
		numWorkers = 1
	}
	if numWorkers > len(chunks) {
		numWorkers = len(chunks)
	}

	// results slice: one entry per chunk, pre-allocated so workers write by index
	// (no shared map, no mutex needed on writes).
	results := make([]chunkResult, len(chunks))

	if numWorkers == 1 {
		// Fast path: sequential evaluation — no goroutine overhead.
		for i, bnd := range chunks {
			results[i] = b.evalChunk(ctx, progWithDemos, examples[bnd.lo:bnd.hi], metric)
		}
	} else {
		// Parallel path: worker pool over a channel of (chunkIndex, bounds).
		type work struct {
			idx int
			bnd bounds
		}
		workCh := make(chan work, len(chunks))
		for i, bnd := range chunks {
			workCh <- work{i, bnd}
		}
		close(workCh)

		var wg sync.WaitGroup
		wg.Add(numWorkers)
		for w := 0; w < numWorkers; w++ {
			go func() {
				defer wg.Done()
				for job := range workCh {
					// Each goroutine writes to its own index — no data race.
					results[job.idx] = b.evalChunk(ctx, progWithDemos, examples[job.bnd.lo:job.bnd.hi], metric)
				}
			}()
		}
		wg.Wait()
	}

	// Aggregate results.
	var totalScore float64
	var totalCount int
	for _, r := range results {
		if r.err != nil {
			continue
		}
		totalScore += r.totalScore
		totalCount += r.count
	}

	if totalCount == 0 {
		return 0, nil
	}
	return totalScore / float64(totalCount), nil
}

// evalChunk evaluates a candidate on a contiguous slice of examples.
// This is always called in a single goroutine and is free of shared state.
func (b *BatchEvaluator) evalChunk(
	ctx context.Context,
	prog optimize.Program,
	chunk []optimize.Example,
	metric optimize.Metric,
) chunkResult {
	var res chunkResult
	for _, ex := range chunk {
		// Respect context cancellation between examples.
		if ctx.Err() != nil {
			res.err = ctx.Err()
			return res
		}

		pred, err := prog.Run(ctx, ex.Inputs)
		if err != nil {
			continue
		}
		score, err := metric.Evaluate(ex, pred, nil)
		if err != nil {
			continue
		}
		res.totalScore += score
		res.count++
	}
	return res
}

// EvaluatePool scores every candidate in the pool over all examples using
// memory-efficient chunked evaluation. The pool is not modified; scores are
// returned in a new slice aligned with the input pool.
//
// Each candidate is evaluated independently so the function is safe to call
// from a single goroutine without external synchronisation.
func (b *BatchEvaluator) EvaluatePool(
	ctx context.Context,
	program optimize.Program,
	pool []simbaCandidate,
	examples []optimize.Example,
	metric optimize.Metric,
) ([]float64, error) {
	scores := make([]float64, len(pool))
	for i, c := range pool {
		score, err := b.EvaluateAll(ctx, program, c, examples, metric)
		if err != nil {
			return nil, err
		}
		scores[i] = score
	}
	return scores, nil
}
