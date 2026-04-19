package metacognitive

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/runtime"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// DefaultMaxHeuristics is the maximum number of heuristics injected per turn.
const DefaultMaxHeuristics = 5

// DefaultEMAAlpha is the exponential moving average smoothing factor for
// capability score updates. Higher values weight recent observations more.
const DefaultEMAAlpha = 0.3

// DefaultMaxStoredHeuristics is the default upper bound on the total
// heuristics persisted per self-model. When exceeded, the lowest-utility
// heuristics are pruned. Set via WithMaxStoredHeuristics.
const DefaultMaxStoredHeuristics = 200

// Compile-time check.
var _ runtime.Plugin = (*Plugin)(nil)

// Plugin is a runtime.Plugin that implements cross-session metacognitive
// learning. It loads relevant heuristics before each turn, injects them as
// system context, and extracts new heuristics after each turn.
type Plugin struct {
	store               SelfModelStore
	extractor           HeuristicExtractor
	maxHeuristics       int
	maxStoredHeuristics int
	emaAlpha            float64
	hooks               Hooks

	// monitor collects signals from agent hooks during execution.
	monitor *Monitor
}

// PluginOption configures a Plugin.
type PluginOption func(*Plugin)

// WithExtractor sets the heuristic extractor. Defaults to SimpleExtractor.
func WithExtractor(e HeuristicExtractor) PluginOption {
	return func(p *Plugin) {
		if e != nil {
			p.extractor = e
		}
	}
}

// WithMaxHeuristics sets the maximum number of heuristics to inject per turn.
// Defaults to DefaultMaxHeuristics.
func WithMaxHeuristics(n int) PluginOption {
	return func(p *Plugin) {
		if n > 0 {
			p.maxHeuristics = n
		}
	}
}

// WithEMAAlpha sets the exponential moving average alpha for capability
// score updates. Must be between 0 and 1. Defaults to DefaultEMAAlpha.
func WithEMAAlpha(alpha float64) PluginOption {
	return func(p *Plugin) {
		if alpha > 0 && alpha <= 1 {
			p.emaAlpha = alpha
		}
	}
}

// WithHooks sets the metacognitive hooks for observing learning events.
func WithHooks(h Hooks) PluginOption {
	return func(p *Plugin) {
		p.hooks = h
	}
}

// WithMaxStoredHeuristics caps the number of heuristics persisted per
// self-model. When AfterTurn causes the total to exceed this value, the
// lowest-utility heuristics are pruned. Defaults to DefaultMaxStoredHeuristics.
func WithMaxStoredHeuristics(n int) PluginOption {
	return func(p *Plugin) {
		if n > 0 {
			p.maxStoredHeuristics = n
		}
	}
}

// NewPlugin creates a MetacognitivePlugin with the given store and options.
func NewPlugin(store SelfModelStore, opts ...PluginOption) *Plugin {
	p := &Plugin{
		store:               store,
		extractor:           NewSimpleExtractor(),
		maxHeuristics:       DefaultMaxHeuristics,
		maxStoredHeuristics: DefaultMaxStoredHeuristics,
		emaAlpha:            DefaultEMAAlpha,
		monitor:             NewMonitor(),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "metacognitive" }

// Monitor returns the plugin's monitor so callers can attach its hooks to
// agents for signal collection.
func (p *Plugin) Monitor() *Monitor { return p.monitor }

// BeforeTurn loads the agent's self-model, retrieves relevant heuristics
// based on the input, and prepends them as system context to the message.
func (p *Plugin) BeforeTurn(ctx context.Context, session *runtime.Session, input schema.Message) (schema.Message, error) {
	if session == nil {
		return input, nil
	}

	agentID := session.AgentID
	if agentID == "" {
		return input, nil
	}

	// Reset the monitor for this turn.
	p.monitor.Reset()

	model, err := p.store.Load(ctx, agentID)
	if err != nil {
		// Non-fatal: proceed without metacognitive context.
		return input, nil
	}

	if p.hooks.OnSelfModelLoaded != nil {
		p.hooks.OnSelfModelLoaded(model)
	}

	// Extract query from input message.
	query := extractTextFromMessage(input)
	if query == "" {
		return input, nil
	}

	// Search for relevant heuristics.
	heuristics, err := p.store.SearchHeuristics(ctx, agentID, query, p.maxHeuristics)
	if err != nil || len(heuristics) == 0 {
		return input, nil
	}

	// Build metacognitive context message.
	heuristicCtx := buildHeuristicContext(heuristics)

	// Store the context in session state so AfterTurn can read it.
	if session.State == nil {
		session.State = make(map[string]any)
	}
	session.State["metacognitive.input"] = query
	session.State["metacognitive.model_loaded"] = true
	session.State["metacognitive.context"] = heuristicCtx

	// Prepend the heuristic context to the user message so the LLM actually
	// sees the learned heuristics. Preserves the original user text verbatim
	// after a newline separator.
	originalText := extractTextFromMessage(input)
	augmented := schema.NewHumanMessage(heuristicCtx + "\n" + originalText)
	return augmented, nil
}

// AfterTurn collects monitoring signals, extracts new heuristics, updates
// capability scores, and persists the updated self-model.
func (p *Plugin) AfterTurn(ctx context.Context, session *runtime.Session, events []agent.Event) ([]agent.Event, error) {
	if session == nil {
		return events, nil
	}

	agentID := session.AgentID
	if agentID == "" {
		return events, nil
	}

	// Collect signals from monitor.
	signals := p.monitor.Signals()

	// Supplement signals from events if monitor didn't capture everything.
	signals = enrichSignalsFromEvents(signals, events)

	// Read task type from session state if set.
	if session.State != nil {
		if tt, ok := session.State["metacognitive.task_type"].(string); ok {
			signals.TaskType = tt
		}
	}

	// Load the model.
	model, err := p.store.Load(ctx, agentID)
	if err != nil {
		return events, nil
	}

	// Extract new heuristics.
	newHeuristics, err := p.extractor.Extract(ctx, signals, model)
	if err != nil {
		return events, nil
	}

	// Add new heuristics to the model.
	for _, h := range newHeuristics {
		model.Heuristics = append(model.Heuristics, h)
		if p.hooks.OnHeuristicExtracted != nil {
			p.hooks.OnHeuristicExtracted(h)
		}
	}

	// Prune stored heuristics to stay within the configured cap. Keeping the
	// highest-utility entries bounds SearchHeuristics scan cost and memory.
	if p.maxStoredHeuristics > 0 && len(model.Heuristics) > p.maxStoredHeuristics {
		pruneHeuristicsByUtility(model, p.maxStoredHeuristics)
	}

	// Update capability scores.
	p.updateCapabilityScore(model, signals)

	// Persist.
	model.UpdatedAt = time.Now()
	if err := p.store.Save(ctx, model); err != nil {
		// Non-fatal: heuristics are lost but execution continues.
		return events, nil
	}

	return events, nil
}

// OnError passes through the error unchanged. The monitor's OnError hook
// captures error signals independently.
func (p *Plugin) OnError(_ context.Context, err error) error {
	return err
}

// updateCapabilityScore applies an EMA update to the capability score
// for the task type in the signals.
func (p *Plugin) updateCapabilityScore(model *SelfModel, signals MonitoringSignals) {
	taskType := signals.TaskType
	if taskType == "" {
		taskType = "general"
	}

	score, ok := model.Capabilities[taskType]
	if !ok {
		score = &CapabilityScore{TaskType: taskType}
		model.Capabilities[taskType] = score
	}

	// EMA update: score_new = score_old + alpha * (outcome - score_old)
	outcome := 0.0
	if signals.Success {
		outcome = 1.0
	}

	if score.SampleCount == 0 {
		score.SuccessRate = outcome
	} else {
		score.SuccessRate = score.SuccessRate + p.emaAlpha*(outcome-score.SuccessRate)
	}

	// Latency EMA.
	if score.SampleCount == 0 {
		score.AvgLatency = signals.TotalLatency
	} else {
		oldNanos := float64(score.AvgLatency.Nanoseconds())
		newNanos := float64(signals.TotalLatency.Nanoseconds())
		score.AvgLatency = time.Duration(oldNanos + p.emaAlpha*(newNanos-oldNanos))
	}

	score.SampleCount++
	score.LastUpdated = time.Now()

	if p.hooks.OnCapabilityUpdated != nil {
		p.hooks.OnCapabilityUpdated(taskType, *score)
	}
}

// extractTextFromMessage extracts text content from a schema.Message.
func extractTextFromMessage(msg schema.Message) string {
	if msg == nil {
		return ""
	}
	parts := msg.GetContent()
	var texts []string
	for _, part := range parts {
		if tp, ok := part.(schema.TextPart); ok {
			texts = append(texts, tp.Text)
		}
	}
	return strings.Join(texts, " ")
}

// pruneHeuristicsByUtility trims the model's heuristics slice to the top
// maxStored entries by Utility (descending).
func pruneHeuristicsByUtility(model *SelfModel, maxStored int) {
	// Use a simple in-place selection to avoid importing sort here; caller
	// invokes this only on the pruning boundary.
	hs := model.Heuristics
	// Partial sort: bubble the top maxStored by utility to the front.
	for i := 0; i < maxStored && i < len(hs); i++ {
		best := i
		for j := i + 1; j < len(hs); j++ {
			if hs[j].Utility > hs[best].Utility {
				best = j
			}
		}
		if best != i {
			hs[i], hs[best] = hs[best], hs[i]
		}
	}
	model.Heuristics = hs[:maxStored]
}

// buildHeuristicContext formats heuristics as a system context string.
func buildHeuristicContext(heuristics []Heuristic) string {
	var sb strings.Builder
	sb.WriteString("[Metacognitive Heuristics]\n")
	sb.WriteString("The following learned heuristics may help with this task:\n")
	for i, h := range heuristics {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, h.Source, h.Content))
	}
	return sb.String()
}

// enrichSignalsFromEvents supplements monitoring signals with data from
// agent events if the monitor hooks did not capture everything.
func enrichSignalsFromEvents(signals MonitoringSignals, events []agent.Event) MonitoringSignals {
	// If monitor captured outcome, keep it.
	if signals.Outcome != "" {
		return signals
	}

	var texts []string
	hasError := false
	for _, evt := range events {
		switch evt.Type {
		case agent.EventText:
			texts = append(texts, evt.Text)
		case agent.EventError:
			hasError = true
			signals.Errors = append(signals.Errors, evt.Text)
		case agent.EventToolCall:
			if evt.ToolCall != nil {
				signals.ToolCalls = append(signals.ToolCalls, evt.ToolCall.Name)
			}
		}
	}
	if len(texts) > 0 {
		signals.Outcome = strings.Join(texts, "")
	}
	if hasError {
		signals.Success = false
	}
	return signals
}
