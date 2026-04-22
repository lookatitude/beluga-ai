package eval

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/v2/schema"
)

// Metric is the interface that all evaluation metrics implement.
// A Metric scores a single evaluation sample and returns a float64 in [0, 1].
type Metric interface {
	// Name returns the unique name of this metric (e.g., "faithfulness").
	Name() string

	// Score evaluates a single sample and returns a score in [0.0, 1.0].
	// Higher scores indicate better quality for quality metrics.
	Score(ctx context.Context, sample EvalSample) (float64, error)
}

// Turn represents a single turn in a multi-turn evaluation trajectory.
// Distinct from eval/clustering.Turn, which is a conversation-pattern type
// used by the clustering sub-package.
type Turn struct {
	// Role is the speaker role for this turn ("user", "assistant", "tool").
	Role string `json:",omitempty"`
	// Content is the textual content of the turn.
	Content string `json:",omitempty"`
	// ToolCalls are tool invocations requested by the assistant in this turn.
	ToolCalls []schema.ToolCall `json:",omitempty"`
}

// EvalSample represents a single evaluation sample containing the input
// question, the generated output, the expected output, and any additional
// context such as retrieved documents and metadata.
type EvalSample struct {
	// Input is the original question or prompt.
	Input string
	// Output is the AI-generated response to evaluate.
	Output string
	// ExpectedOutput is the ground-truth or reference answer.
	ExpectedOutput string
	// RetrievedDocs are the documents used as context for generation.
	RetrievedDocs []schema.Document
	// Metadata holds arbitrary key-value pairs for metric-specific data,
	// such as latency_ms, input_tokens, output_tokens, model, etc.
	Metadata map[string]any
	// Turns is the optional multi-turn history for trajectory evaluation.
	// Leading-comma json tag preserves the existing PascalCase wire format
	// while omitting the field when unset for backward-compat with pre-S4
	// dataset files.
	Turns []Turn `json:",omitempty"`
	// ExpectedTools is the list of tool names expected in the trajectory
	// for tool-use evaluation. Omitted when unset for backward-compat.
	ExpectedTools []string `json:",omitempty"`
}

// SampleResult holds the evaluation scores for a single sample across
// all configured metrics.
type SampleResult struct {
	// Sample is the original evaluation sample.
	Sample EvalSample
	// Scores maps metric names to their scores for this sample.
	Scores map[string]float64
	// Error is set if any metric failed for this sample.
	Error error
}

// EvalReport is the aggregate result of running an evaluation suite.
type EvalReport struct {
	// Samples contains the per-sample results.
	Samples []SampleResult
	// Metrics contains the average score for each metric across all samples.
	Metrics map[string]float64
	// Duration is the total wall-clock time of the evaluation run.
	Duration time.Duration
	// Errors collects all errors encountered during evaluation.
	Errors []error
}
