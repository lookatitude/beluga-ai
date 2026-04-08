package metacognitive

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// HeuristicExtractor derives heuristics from execution signals and the
// current self-model. Implementations range from simple rule-based extraction
// to LLM-powered analysis.
type HeuristicExtractor interface {
	// Extract analyzes the signals and model to produce new heuristics.
	Extract(ctx context.Context, signals MonitoringSignals, model *SelfModel) ([]Heuristic, error)
}

// Compile-time check.
var _ HeuristicExtractor = (*SimpleExtractor)(nil)

// SimpleExtractor is a rule-based heuristic extractor that derives learnings
// from failures and successes without requiring an LLM.
//
// On failure: extracts "avoid" heuristics from error messages and tool failures.
// On success with notable patterns: extracts "prefer" heuristics.
type SimpleExtractor struct{}

// NewSimpleExtractor creates a new rule-based SimpleExtractor.
func NewSimpleExtractor() *SimpleExtractor {
	return &SimpleExtractor{}
}

// Extract derives heuristics from the execution signals.
func (e *SimpleExtractor) Extract(ctx context.Context, signals MonitoringSignals, model *SelfModel) ([]Heuristic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var heuristics []Heuristic

	taskType := signals.TaskType
	if taskType == "" {
		taskType = "general"
	}

	if !signals.Success {
		// Failure-derived heuristics.
		heuristics = append(heuristics, e.extractFromFailure(signals, taskType)...)
	} else {
		// Success-derived heuristics.
		heuristics = append(heuristics, e.extractFromSuccess(signals, taskType, model)...)
	}

	return heuristics, nil
}

// extractFromFailure generates "avoid" heuristics from failure signals.
func (e *SimpleExtractor) extractFromFailure(signals MonitoringSignals, taskType string) []Heuristic {
	var heuristics []Heuristic

	// Extract from errors.
	seen := make(map[string]bool)
	for _, errMsg := range signals.Errors {
		// Deduplicate similar errors.
		key := normalizeError(errMsg)
		if seen[key] {
			continue
		}
		seen[key] = true

		h := Heuristic{
			ID:        generateID(),
			Content:   fmt.Sprintf("Avoid: encountered error during %s task: %s", taskType, summarizeError(errMsg)),
			Source:    "failure",
			TaskType:  taskType,
			Utility:   0.7,
			CreatedAt: time.Now(),
		}
		heuristics = append(heuristics, h)
	}

	// High iteration count suggests inefficient approach.
	if signals.IterationCount > 5 {
		h := Heuristic{
			ID:        generateID(),
			Content:   fmt.Sprintf("Avoid: excessive iterations (%d) on %s tasks. Consider a more direct approach.", signals.IterationCount, taskType),
			Source:    "failure",
			TaskType:  taskType,
			Utility:   0.6,
			CreatedAt: time.Now(),
		}
		heuristics = append(heuristics, h)
	}

	return heuristics
}

// extractFromSuccess generates "prefer" heuristics from successful executions.
func (e *SimpleExtractor) extractFromSuccess(signals MonitoringSignals, taskType string, model *SelfModel) []Heuristic {
	var heuristics []Heuristic

	// Fast completion is worth noting.
	if signals.IterationCount <= 2 && len(signals.ToolCalls) > 0 {
		toolList := strings.Join(signals.ToolCalls, ", ")
		h := Heuristic{
			ID:        generateID(),
			Content:   fmt.Sprintf("Prefer: for %s tasks, using tools [%s] resolved the task efficiently in %d iterations.", taskType, toolList, signals.IterationCount),
			Source:    "success",
			TaskType:  taskType,
			Utility:   0.8,
			CreatedAt: time.Now(),
		}
		heuristics = append(heuristics, h)
	}

	// Novel tool combination (not seen in existing heuristics).
	if len(signals.ToolCalls) >= 2 && !hasToolCombinationHeuristic(model, signals.ToolCalls) {
		toolList := strings.Join(signals.ToolCalls, " -> ")
		h := Heuristic{
			ID:        generateID(),
			Content:   fmt.Sprintf("Prefer: successful tool sequence for %s: %s", taskType, toolList),
			Source:    "success",
			TaskType:  taskType,
			Utility:   0.6,
			CreatedAt: time.Now(),
		}
		heuristics = append(heuristics, h)
	}

	return heuristics
}

// hasToolCombinationHeuristic checks if the model already has a heuristic
// mentioning the same tool combination.
func hasToolCombinationHeuristic(model *SelfModel, tools []string) bool {
	if model == nil {
		return false
	}
	toolList := strings.Join(tools, " -> ")
	for _, h := range model.Heuristics {
		if strings.Contains(h.Content, toolList) {
			return true
		}
	}
	return false
}

// normalizeError produces a deduplication key for an error message.
func normalizeError(errMsg string) string {
	lower := strings.ToLower(errMsg)
	// Truncate to first 80 chars for dedup key.
	if len(lower) > 80 {
		lower = lower[:80]
	}
	return lower
}

// summarizeError truncates an error message to a reasonable length.
func summarizeError(errMsg string) string {
	const maxLen = 200
	if len(errMsg) <= maxLen {
		return errMsg
	}
	return errMsg[:maxLen] + "..."
}

// generateID creates a short random hex ID for heuristics.
func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use timestamp-based ID.
		return fmt.Sprintf("h_%d", time.Now().UnixNano())
	}
	return "h_" + hex.EncodeToString(b)
}
