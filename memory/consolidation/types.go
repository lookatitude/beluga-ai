package consolidation

import "time"

// UtilityScore holds the individual dimensions that contribute to a memory
// record's overall utility. Each dimension is normalised to [0, 1].
type UtilityScore struct {
	// Recency is the time-decay component, computed via exponential half-life.
	Recency float64

	// Importance captures the intrinsic significance of the memory content.
	Importance float64

	// Relevance measures how related the memory is to current context.
	Relevance float64

	// EmotionalSalience captures the emotional weight of the memory.
	EmotionalSalience float64

	// AccessCount is the total number of times this record has been accessed.
	AccessCount int

	// LastAccessedAt is the most recent time this record was read.
	LastAccessedAt time.Time
}

// Weights controls how the individual UtilityScore dimensions are combined
// into a single composite score. All weights should sum to 1.0 but this is
// not enforced — they are normalised at scoring time.
type Weights struct {
	Recency           float64
	Importance        float64
	Relevance         float64
	EmotionalSalience float64
}

// DefaultWeights returns the recommended weighting: Recency 0.4,
// Importance 0.3, Relevance 0.2, EmotionalSalience 0.1.
func DefaultWeights() Weights {
	return Weights{
		Recency:           0.4,
		Importance:        0.3,
		Relevance:         0.2,
		EmotionalSalience: 0.1,
	}
}

// Record is a memory entry subject to consolidation evaluation.
type Record struct {
	// ID uniquely identifies the record within the store.
	ID string

	// Content is the textual content of the memory.
	Content string

	// Utility holds the scoring dimensions for this record.
	Utility UtilityScore

	// CreatedAt is when the record was first persisted.
	CreatedAt time.Time

	// Metadata carries arbitrary key-value pairs.
	Metadata map[string]any
}

// Action describes what a consolidation policy decided for a record.
type Action int

const (
	// ActionKeep means the record should be retained as-is.
	ActionKeep Action = iota

	// ActionCompress means the record should be summarised/compressed.
	ActionCompress

	// ActionPrune means the record should be deleted.
	ActionPrune
)

// Decision pairs a record with the action the policy chose for it.
type Decision struct {
	Record Record
	Action Action
}

// CycleMetrics captures statistics for a single consolidation cycle.
type CycleMetrics struct {
	// CycleStart is when the cycle began.
	CycleStart time.Time

	// CycleEnd is when the cycle finished.
	CycleEnd time.Time

	// RecordsEvaluated is the total number of records scored.
	RecordsEvaluated int

	// RecordsPruned is how many records were deleted.
	RecordsPruned int

	// RecordsCompressed is how many records were summarised.
	RecordsCompressed int

	// Errors collects any non-fatal errors encountered during the cycle.
	Errors []error
}

// Hooks provides optional callbacks that fire during consolidation.
// A nil function field is silently skipped.
type Hooks struct {
	// OnPruned is called after records have been pruned in a cycle.
	OnPruned func(records []Record)

	// OnCompressed is called after records have been compressed in a cycle.
	OnCompressed func(original, compressed []Record)

	// OnCycleComplete is called at the end of every consolidation cycle.
	OnCycleComplete func(metrics CycleMetrics)
}

// ComposeHooks merges multiple Hooks into one. Each hook function chains
// through all non-nil implementations in order.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnPruned: func(records []Record) {
			for _, h := range hooks {
				if h.OnPruned != nil {
					h.OnPruned(records)
				}
			}
		},
		OnCompressed: func(original, compressed []Record) {
			for _, h := range hooks {
				if h.OnCompressed != nil {
					h.OnCompressed(original, compressed)
				}
			}
		},
		OnCycleComplete: func(metrics CycleMetrics) {
			for _, h := range hooks {
				if h.OnCycleComplete != nil {
					h.OnCycleComplete(metrics)
				}
			}
		},
	}
}
