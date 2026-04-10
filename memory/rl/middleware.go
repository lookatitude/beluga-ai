package rl

import (
	"context"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
)

// FeatureExtractor computes PolicyFeatures from the current memory state
// and the incoming messages. Implementations may query the underlying
// memory to assess similarity, count entries, etc.
type FeatureExtractor interface {
	// Extract derives observation features from the memory state and the
	// input/output messages being saved.
	Extract(ctx context.Context, mem memory.Memory, input, output schema.Message) (PolicyFeatures, error)
}

// SizedMemory is an optional capability interface: memories that can
// report their current total entry count. The DefaultFeatureExtractor
// uses Size() when available so that HeuristicPolicy can make informed
// ActionDelete decisions based on MaxStoreSize.
type SizedMemory interface {
	Size(ctx context.Context) (int, error)
}

// DeletableMemory is an optional capability interface: memories that
// support granular entry deletion by ID. PolicyMemory routes ActionDelete
// through this interface when available. Memories that do not implement
// it cause ActionDelete to return an error explaining the missing
// capability.
type DeletableMemory interface {
	Delete(ctx context.Context, id string) error
}

// Option configures a PolicyMemory.
type Option func(*policyOpts)

type policyOpts struct {
	policy    MemoryPolicy
	hooks     Hooks
	extractor FeatureExtractor
	collector *TrajectoryCollector
}

// WithPolicy sets the MemoryPolicy used for action decisions.
func WithPolicy(p MemoryPolicy) Option {
	return func(o *policyOpts) { o.policy = p }
}

// WithHooks sets the lifecycle hooks for observing policy decisions.
func WithHooks(h Hooks) Option {
	return func(o *policyOpts) { o.hooks = h }
}

// WithFeatureExtractor sets the FeatureExtractor used to compute
// observations from memory state.
func WithFeatureExtractor(e FeatureExtractor) Option {
	return func(o *policyOpts) { o.extractor = e }
}

// WithCollector sets the TrajectoryCollector for recording decision
// episodes for offline training.
func WithCollector(c *TrajectoryCollector) Option {
	return func(o *policyOpts) { o.collector = c }
}

// PolicyMemory wraps a memory.Memory with an RL policy that decides whether
// to add, update, delete, or skip when Save is called. Load, Search, and
// Clear are passed through to the underlying memory.
type PolicyMemory struct {
	inner memory.Memory
	opts  policyOpts
}

// New creates a PolicyMemory that wraps inner with the given options.
// At minimum, a policy should be provided via WithPolicy. If no policy is
// set, the default HeuristicPolicy is used. If no feature extractor is set,
// a DefaultFeatureExtractor is used.
func New(inner memory.Memory, opts ...Option) *PolicyMemory {
	o := policyOpts{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.policy == nil {
		o.policy = NewHeuristicPolicy()
	}
	if o.extractor == nil {
		o.extractor = &DefaultFeatureExtractor{}
	}
	return &PolicyMemory{inner: inner, opts: o}
}

// Save extracts features from the current memory state and the incoming
// messages, runs the policy to decide an action, invokes any hooks, and
// routes to the appropriate underlying memory operation.
func (m *PolicyMemory) Save(ctx context.Context, input, output schema.Message) error {
	// Extract features.
	features, err := m.opts.extractor.Extract(ctx, m.inner, input, output)
	if err != nil {
		return err
	}

	// Run policy.
	action, confidence, err := m.opts.policy.Decide(ctx, features)
	if err != nil {
		return err
	}

	// Invoke OnDecision hook.
	if m.opts.hooks.OnDecision != nil {
		if err := m.opts.hooks.OnDecision(ctx, features, action, confidence); err != nil {
			return err
		}
	}

	// Record step if collector is set.
	if m.opts.collector != nil {
		m.opts.collector.RecordStep(features, action, confidence)
	}

	// Route action.
	switch action {
	case ActionAdd:
		return m.inner.Save(ctx, input, output)
	case ActionUpdate:
		// Update semantics: delete the closest existing entry (when the
		// backing store supports deletion) and save the new pair in its
		// place. If the backing store is not DeletableMemory, we fall
		// back to Save — which will create a duplicate — and return an
		// error so callers can detect the semantic gap.
		if del, ok := m.inner.(DeletableMemory); ok {
			if id, lookupErr := m.closestEntryID(ctx, output); lookupErr == nil && id != "" {
				if derr := del.Delete(ctx, id); derr != nil {
					return derr
				}
			}
			return m.inner.Save(ctx, input, output)
		}
		return core.NewError(
			"rl.policy_memory.update",
			core.ErrInvalidInput,
			"ActionUpdate requires inner memory to implement DeletableMemory",
			nil,
		)
	case ActionDelete:
		if del, ok := m.inner.(DeletableMemory); ok {
			id, lookupErr := m.closestEntryID(ctx, output)
			if lookupErr != nil {
				return lookupErr
			}
			if id == "" {
				return nil
			}
			return del.Delete(ctx, id)
		}
		return core.NewError(
			"rl.policy_memory.delete",
			core.ErrInvalidInput,
			"ActionDelete requires inner memory to implement DeletableMemory",
			nil,
		)
	case ActionNoop:
		return nil
	default:
		return nil
	}
}

// closestEntryID looks up the single most-similar entry to the output
// message and returns its ID from the document metadata. Returns an empty
// string if no candidate is found.
func (m *PolicyMemory) closestEntryID(ctx context.Context, output schema.Message) (string, error) {
	query := messageText(output)
	docs, err := m.inner.Search(ctx, query, 1)
	if err != nil {
		return "", err
	}
	if len(docs) == 0 {
		return "", nil
	}
	if id, ok := docs[0].Metadata["id"].(string); ok {
		return id, nil
	}
	return docs[0].ID, nil
}

// Load passes through to the underlying memory.
func (m *PolicyMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	return m.inner.Load(ctx, query)
}

// Search passes through to the underlying memory.
func (m *PolicyMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	return m.inner.Search(ctx, query, k)
}

// Clear passes through to the underlying memory.
func (m *PolicyMemory) Clear(ctx context.Context) error {
	return m.inner.Clear(ctx)
}

// Compile-time interface check.
var _ memory.Memory = (*PolicyMemory)(nil)
