package context

import (
	gocontext "context"
	"fmt"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// ContextStep is a single step in the context engineering pipeline.
// Each step transforms the context items in some way.
type ContextStep interface {
	// Process transforms the context items and returns the result.
	Process(ctx gocontext.Context, items []ContextItem) ([]ContextItem, error)
	// Name returns the step name for logging and debugging.
	Name() string
}

// Executor orchestrates a sequence of ContextSteps to prepare
// optimal LLM context.
type Executor interface {
	// Execute runs all pipeline steps in order.
	Execute(ctx gocontext.Context, input PipelineInput) (PipelineOutput, error)
}

// ContextItem is a single piece of context that may be included in the
// LLM prompt. Items have a score for ranking.
type ContextItem struct {
	// ID uniquely identifies this item.
	ID string
	// Type classifies the item (e.g., "message", "document", "tool_result").
	Type string
	// Content is the textual content.
	Content string
	// Score is the relevance score in [0.0, 1.0].
	Score float64
	// Metadata holds extra attributes.
	Metadata map[string]any
}

// PipelineInput is the input to a context pipeline.
type PipelineInput struct {
	// Query is the current user query.
	Query string
	// Messages are the existing conversation messages.
	Messages []schema.Message
	// MaxTokens is the target context window budget.
	MaxTokens int
	// Metadata holds extra attributes.
	Metadata map[string]any
}

// PipelineOutput is the result of running the context pipeline.
type PipelineOutput struct {
	// Items are the selected and ordered context items.
	Items []ContextItem
	// Messages are the final prepared messages for the LLM, reconstructed
	// from Items whose "source" metadata key is "message". Items added by
	// pipeline steps that do not originate from a schema.Message are not
	// included here and must be consumed via Items directly.
	Messages []schema.Message
	// TokenEstimate is the estimated token count.
	TokenEstimate int
	// StepsExecuted lists the steps that ran.
	StepsExecuted []string
}

// Option configures a DefaultPipeline.
type Option func(*pipelineOptions)

type pipelineOptions struct {
	steps     []ContextStep
	maxTokens int
}

// WithStep adds a step to the pipeline.
func WithStep(step ContextStep) Option {
	return func(o *pipelineOptions) { o.steps = append(o.steps, step) }
}

// WithMaxTokens sets the default maximum token budget.
func WithMaxTokens(n int) Option {
	return func(o *pipelineOptions) { o.maxTokens = n }
}

// DefaultPipeline is the standard context pipeline implementation.
type DefaultPipeline struct {
	opts pipelineOptions
}

var _ Executor = (*DefaultPipeline)(nil)

// NewPipeline creates a context pipeline with the given options.
func NewPipeline(opts ...Option) *DefaultPipeline {
	o := pipelineOptions{maxTokens: 4096}
	for _, opt := range opts {
		opt(&o)
	}
	return &DefaultPipeline{opts: o}
}

// Execute runs all pipeline steps in sequence.
func (p *DefaultPipeline) Execute(ctx gocontext.Context, input PipelineInput) (PipelineOutput, error) {
	if input.MaxTokens <= 0 {
		input.MaxTokens = p.opts.maxTokens
	}

	// Convert messages to context items.
	items := messagesToItems(input.Messages)

	var stepsExecuted []string

	for _, step := range p.opts.steps {
		if err := ctx.Err(); err != nil {
			return PipelineOutput{}, err
		}

		var err error
		items, err = step.Process(ctx, items)
		if err != nil {
			return PipelineOutput{}, core.Errorf(core.ErrInvalidInput, "context: step %q: %w", step.Name(), err)
		}
		stepsExecuted = append(stepsExecuted, step.Name())
	}

	tokenEstimate := estimateTokens(items)

	return PipelineOutput{
		Items:         items,
		Messages:      itemsToMessages(items, input.Messages),
		TokenEstimate: tokenEstimate,
		StepsExecuted: stepsExecuted,
	}, nil
}

// itemsToMessages rebuilds the ordered message slice from the pipeline's
// remaining items. Only items that originated from the input messages
// (identified by the "source"="message" and "index" metadata set in
// messagesToItems) are included, preserving the order in which the pipeline
// left them.
func itemsToMessages(items []ContextItem, original []schema.Message) []schema.Message {
	result := make([]schema.Message, 0, len(items))
	for _, it := range items {
		if it.Metadata == nil {
			continue
		}
		src, _ := it.Metadata["source"].(string)
		if src != "message" {
			continue
		}
		idx, ok := it.Metadata["index"].(int)
		if !ok || idx < 0 || idx >= len(original) {
			continue
		}
		result = append(result, original[idx])
	}
	return result
}

func messagesToItems(msgs []schema.Message) []ContextItem {
	items := make([]ContextItem, 0, len(msgs))
	for i, m := range msgs {
		content := ""
		for _, p := range m.GetContent() {
			if tp, ok := p.(schema.TextPart); ok {
				content += tp.Text
			}
		}
		items = append(items, ContextItem{
			ID:      fmt.Sprintf("msg-%d", i),
			Type:    "message",
			Content: content,
			Score:   1.0,
			Metadata: map[string]any{
				"source": "message",
				"role":   string(m.GetRole()),
				"index":  i,
			},
		})
	}
	return items
}

func estimateTokens(items []ContextItem) int {
	total := 0
	for _, item := range items {
		total += len(item.Content) / 4 // Rough: 4 chars per token.
	}
	if total == 0 {
		total = 1
	}
	return total
}

// PipelineBuilder fluently constructs a Executor.
type PipelineBuilder struct {
	opts []Option
}

// NewBuilder creates a pipeline builder.
func NewBuilder() *PipelineBuilder {
	return &PipelineBuilder{}
}

// addStep appends a pipeline step option and returns the builder for chaining.
func (b *PipelineBuilder) addStep(step ContextStep) *PipelineBuilder {
	b.opts = append(b.opts, WithStep(step))
	return b
}

// WithRetrieve adds a retrieve step.
func (b *PipelineBuilder) WithRetrieve(step ContextStep) *PipelineBuilder {
	return b.addStep(step)
}

// WithRank adds a ranking step.
func (b *PipelineBuilder) WithRank(step ContextStep) *PipelineBuilder {
	return b.addStep(step)
}

// WithFilter adds a filtering step.
func (b *PipelineBuilder) WithFilter(step ContextStep) *PipelineBuilder {
	return b.addStep(step)
}

// WithStructure adds a structuring step.
func (b *PipelineBuilder) WithStructure(step ContextStep) *PipelineBuilder {
	return b.addStep(step)
}

// SetMaxTokens sets the token budget.
func (b *PipelineBuilder) SetMaxTokens(n int) *PipelineBuilder {
	b.opts = append(b.opts, WithMaxTokens(n))
	return b
}

// Build creates the configured pipeline.
func (b *PipelineBuilder) Build() *DefaultPipeline {
	return NewPipeline(b.opts...)
}

// RelevanceRanker ranks context items by their relevance score.
type RelevanceRanker struct{}

var _ ContextStep = (*RelevanceRanker)(nil)

// Name returns the step name.
func (r *RelevanceRanker) Name() string { return "relevance_rank" }

// Process sorts items by score descending.
func (r *RelevanceRanker) Process(_ gocontext.Context, items []ContextItem) ([]ContextItem, error) {
	result := make([]ContextItem, len(items))
	copy(result, items)
	sort.Slice(result, func(i, j int) bool { return result[i].Score > result[j].Score })
	return result, nil
}

// TokenBudgetFilter removes items that exceed the token budget.
type TokenBudgetFilter struct {
	maxTokens int
}

var _ ContextStep = (*TokenBudgetFilter)(nil)

// NewTokenBudgetFilter creates a filter with the given token limit.
func NewTokenBudgetFilter(maxTokens int) *TokenBudgetFilter {
	return &TokenBudgetFilter{maxTokens: maxTokens}
}

// Name returns the step name.
func (f *TokenBudgetFilter) Name() string { return "token_budget_filter" }

// Process keeps items in their original order until the token budget is
// exhausted. Items that individually exceed the remaining budget are
// skipped; subsequent smaller items may still be included (greedy packing).
func (f *TokenBudgetFilter) Process(_ gocontext.Context, items []ContextItem) ([]ContextItem, error) {
	var result []ContextItem
	used := 0
	for _, item := range items {
		tokens := len(item.Content) / 4
		if tokens == 0 {
			tokens = 1
		}
		if used+tokens > f.maxTokens {
			continue
		}
		used += tokens
		result = append(result, item)
	}
	return result, nil
}

// DuplicateFilter removes items with duplicate content.
type DuplicateFilter struct{}

var _ ContextStep = (*DuplicateFilter)(nil)

// Name returns the step name.
func (f *DuplicateFilter) Name() string { return "duplicate_filter" }

// Process removes duplicate items based on content.
func (f *DuplicateFilter) Process(_ gocontext.Context, items []ContextItem) ([]ContextItem, error) {
	seen := make(map[string]bool)
	var result []ContextItem
	for _, item := range items {
		if seen[item.Content] {
			continue
		}
		seen[item.Content] = true
		result = append(result, item)
	}
	return result, nil
}

// RecencyBooster increases scores for more recent items.
type RecencyBooster struct {
	boost float64
}

var _ ContextStep = (*RecencyBooster)(nil)

// NewRecencyBooster creates a booster that adds the given value to recent items.
func NewRecencyBooster(boost float64) *RecencyBooster {
	return &RecencyBooster{boost: boost}
}

// Name returns the step name.
func (b *RecencyBooster) Name() string { return "recency_boost" }

// Process increases scores for items later in the list.
func (b *RecencyBooster) Process(_ gocontext.Context, items []ContextItem) ([]ContextItem, error) {
	result := make([]ContextItem, len(items))
	copy(result, items)
	for i := range result {
		// More recent items (higher index) get more boost.
		fraction := float64(i) / float64(max(1, len(result)-1))
		result[i].Score += b.boost * fraction
		if result[i].Score > 1.0 {
			result[i].Score = 1.0
		}
	}
	return result, nil
}

// Factory creates a ContextStep.
type Factory func() (ContextStep, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a step factory to the global registry.
//
// Register MUST only be called from package init() functions. Calling it
// after program start is unsupported and may race with concurrent NewStep
// lookups. Third-party steps should self-register via their own init().
func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

// NewStep creates a ContextStep by name from the registry.
func NewStep(name string) (ContextStep, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, core.Errorf(core.ErrNotFound, "context: unknown step %q (registered: %v)", name, ListSteps())
	}
	return f()
}

// ListSteps returns the names of all registered steps, sorted alphabetically.
func ListSteps() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func init() {
	Register("relevance_rank", func() (ContextStep, error) { return &RelevanceRanker{}, nil })
	Register("token_budget_filter", func() (ContextStep, error) { return NewTokenBudgetFilter(4096), nil })
	Register("duplicate_filter", func() (ContextStep, error) { return &DuplicateFilter{}, nil })
	Register("recency_boost", func() (ContextStep, error) { return NewRecencyBooster(0.2), nil })
}
