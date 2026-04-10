package routing

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
)

// ModelTier represents the capability tier of a model.
type ModelTier string

const (
	// TierSmall is for simple tasks (classification, extraction, short Q&A).
	TierSmall ModelTier = "small"
	// TierMedium is for moderate tasks (summarization, code generation).
	TierMedium ModelTier = "medium"
	// TierLarge is for complex tasks (multi-step reasoning, creative writing).
	TierLarge ModelTier = "large"
)

// ComplexityClassifier determines the complexity of a request.
type ComplexityClassifier interface {
	// Classify returns the estimated complexity tier for the given messages.
	Classify(ctx context.Context, msgs []schema.Message) (ModelTier, error)
}

// BudgetEnforcer tracks and enforces cost budgets.
type BudgetEnforcer interface {
	// Check returns whether the estimated cost is within budget.
	Check(ctx context.Context, estimatedCost float64) (bool, error)
	// CheckAndReserve atomically verifies that estimatedCost fits within the
	// remaining budget and reserves it. Returns (true, nil) when reserved.
	// Callers should later call Record with the actual cost; the difference
	// between reserved and actual spend can be reconciled by callers if
	// needed. This prevents TOCTOU races across concurrent callers.
	CheckAndReserve(ctx context.Context, estimatedCost float64) (bool, error)
	// Record records an actual cost expenditure.
	Record(ctx context.Context, cost float64) error
}

// CostRouter selects a model based on complexity and cost considerations.
type CostRouter interface {
	// SelectModel chooses the best model for the given messages and tier.
	SelectModel(ctx context.Context, msgs []schema.Message) (ModelSelection, error)
}

// ModelSelection is the result of a routing decision.
type ModelSelection struct {
	// ModelID is the selected model identifier.
	ModelID string
	// Tier is the model's capability tier.
	Tier ModelTier
	// EstimatedCost is the estimated cost for this request.
	EstimatedCost float64
	// Reason explains the routing decision.
	Reason string
}

// ModelConfig describes a model available for routing.
type ModelConfig struct {
	// ID is the model identifier.
	ID string
	// Tier is the model's capability tier.
	Tier ModelTier
	// CostPerInputToken is the cost per input token.
	CostPerInputToken float64
	// CostPerOutputToken is the cost per output token.
	CostPerOutputToken float64
	// MaxTokens is the model's maximum context window.
	MaxTokens int
	// Priority determines selection order within the same tier (lower = preferred).
	Priority int
}

// Option configures a DefaultCostRouter.
type Option func(*routerOptions)

type routerOptions struct {
	models     []ModelConfig
	classifier ComplexityClassifier
	enforcer   BudgetEnforcer
	fallback   ModelTier
}

// WithModels sets the available models.
func WithModels(models ...ModelConfig) Option {
	return func(o *routerOptions) { o.models = models }
}

// WithClassifier sets the complexity classifier.
func WithClassifier(c ComplexityClassifier) Option {
	return func(o *routerOptions) { o.classifier = c }
}

// WithBudgetEnforcer sets the budget enforcer.
func WithBudgetEnforcer(e BudgetEnforcer) Option {
	return func(o *routerOptions) { o.enforcer = e }
}

// WithFallbackTier sets the fallback tier when classification is uncertain.
func WithFallbackTier(t ModelTier) Option {
	return func(o *routerOptions) { o.fallback = t }
}

// DefaultCostRouter implements CostRouter with configurable complexity
// classification and budget enforcement.
type DefaultCostRouter struct {
	opts routerOptions
}

var _ CostRouter = (*DefaultCostRouter)(nil)

// NewCostRouter creates a cost-aware model router.
func NewCostRouter(opts ...Option) *DefaultCostRouter {
	o := routerOptions{
		classifier: &HeuristicClassifier{},
		fallback:   TierMedium,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &DefaultCostRouter{opts: o}
}

// SelectModel chooses the best model based on complexity and budget.
func (r *DefaultCostRouter) SelectModel(ctx context.Context, msgs []schema.Message) (ModelSelection, error) {
	if len(r.opts.models) == 0 {
		return ModelSelection{}, fmt.Errorf("routing: no models configured")
	}

	tier, err := r.opts.classifier.Classify(ctx, msgs)
	if err != nil {
		tier = r.opts.fallback
	}

	// Find models matching the tier, sorted by priority.
	candidates := r.modelsForTier(tier)
	if len(candidates) == 0 {
		// Fall back to any model.
		candidates = r.allModelsSorted()
	}

	// Estimate cost and check budget.
	inputTokens := estimateTokens(msgs)
	tierDowngraded := len(r.modelsForTier(tier)) == 0
	for _, m := range candidates {
		estimated := float64(inputTokens)*m.CostPerInputToken + (float64(inputTokens)/2.0)*m.CostPerOutputToken

		if r.opts.enforcer != nil {
			allowed, err := r.opts.enforcer.CheckAndReserve(ctx, estimated)
			if err != nil {
				return ModelSelection{}, fmt.Errorf("routing: budget check: %w", err)
			}
			if !allowed {
				continue // Try cheaper model.
			}
		}

		reason := fmt.Sprintf("selected %s (tier=%s) for complexity=%s (requested=%s)", m.ID, m.Tier, tier, tier)
		if tierDowngraded {
			reason = fmt.Sprintf("selected %s (tier=%s) for complexity=%s (requested=%s, downgraded: no models available for requested tier)", m.ID, m.Tier, tier, tier)
		}
		return ModelSelection{
			ModelID:       m.ID,
			Tier:          m.Tier,
			EstimatedCost: estimated,
			Reason:        reason,
		}, nil
	}

	return ModelSelection{}, fmt.Errorf("routing: no model within budget for tier %s", tier)
}

// Enforcer returns the configured budget enforcer, if any. Callers should
// call Record on the enforcer after the actual LLM call completes to
// reconcile reservations with actual spend.
func (r *DefaultCostRouter) Enforcer() BudgetEnforcer {
	return r.opts.enforcer
}

func (r *DefaultCostRouter) modelsForTier(tier ModelTier) []ModelConfig {
	var result []ModelConfig
	for _, m := range r.opts.models {
		if m.Tier == tier {
			result = append(result, m)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Priority < result[j].Priority })
	return result
}

func (r *DefaultCostRouter) allModelsSorted() []ModelConfig {
	result := make([]ModelConfig, len(r.opts.models))
	copy(result, r.opts.models)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CostPerInputToken < result[j].CostPerInputToken
	})
	return result
}

func estimateTokens(msgs []schema.Message) int {
	total := 0
	for _, m := range msgs {
		for _, p := range m.GetContent() {
			if tp, ok := p.(schema.TextPart); ok {
				total += len(tp.Text) / 4
			} // Rough estimate: 4 chars per token.
		}
	}
	if total == 0 {
		total = 100 // Minimum estimate.
	}
	return total
}

// HeuristicClassifier classifies complexity using simple heuristics
// based on message count, total length, and presence of system prompts.
type HeuristicClassifier struct{}

var _ ComplexityClassifier = (*HeuristicClassifier)(nil)

// Classify returns a tier based on heuristic analysis of the messages.
func (c *HeuristicClassifier) Classify(_ context.Context, msgs []schema.Message) (ModelTier, error) {
	totalChars := 0
	hasSystem := false
	msgCount := len(msgs)

	for _, m := range msgs {
		if m.GetRole() == schema.RoleSystem {
			hasSystem = true
		}
		for _, p := range m.GetContent() {
			if tp, ok := p.(schema.TextPart); ok {
				totalChars += len(tp.Text)
			}
		}
	}

	// Complex: many messages, long content, or system prompt with instructions.
	if msgCount > 10 || totalChars > 5000 || (hasSystem && totalChars > 2000) {
		return TierLarge, nil
	}

	// Simple: short single-turn.
	if msgCount <= 2 && totalChars < 500 {
		return TierSmall, nil
	}

	return TierMedium, nil
}

// InMemoryBudgetEnforcer tracks spend in memory with daily budget limits.
type InMemoryBudgetEnforcer struct {
	mu         sync.Mutex
	dailyLimit float64
	spent      float64
	resetTime  time.Time
}

var _ BudgetEnforcer = (*InMemoryBudgetEnforcer)(nil)

// NewBudgetEnforcer creates an in-memory budget enforcer with a daily limit.
func NewBudgetEnforcer(dailyLimit float64) *InMemoryBudgetEnforcer {
	return &InMemoryBudgetEnforcer{
		dailyLimit: dailyLimit,
		resetTime:  nextMidnight(),
	}
}

// Check returns whether the estimated cost fits within the remaining budget.
// Note: Check is non-reserving and therefore races with concurrent callers.
// Prefer CheckAndReserve for routing decisions.
func (e *InMemoryBudgetEnforcer) Check(_ context.Context, estimatedCost float64) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.maybeReset()
	return e.spent+estimatedCost <= e.dailyLimit, nil
}

// CheckAndReserve atomically verifies that estimatedCost fits within the
// remaining budget and reserves it against current spend.
func (e *InMemoryBudgetEnforcer) CheckAndReserve(_ context.Context, estimatedCost float64) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.maybeReset()
	if e.spent+estimatedCost > e.dailyLimit {
		return false, nil
	}
	e.spent += estimatedCost
	return true, nil
}

// Record adds the actual cost to the daily spend.
func (e *InMemoryBudgetEnforcer) Record(_ context.Context, cost float64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.maybeReset()
	e.spent += cost
	return nil
}

// Remaining returns the remaining daily budget.
func (e *InMemoryBudgetEnforcer) Remaining() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.maybeReset()
	return math.Max(0, e.dailyLimit-e.spent)
}

func (e *InMemoryBudgetEnforcer) maybeReset() {
	if time.Now().After(e.resetTime) {
		e.spent = 0
		e.resetTime = nextMidnight()
	}
}

func nextMidnight() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
}
