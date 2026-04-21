package judge

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/eval"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

const judgePromptTemplate = `You are an expert evaluation judge. Score the following AI-generated response using the rubric below.

%s

Input/Question: %s

AI Response: %s

%s

For each criterion, respond with ONLY the criterion name followed by a colon and the numeric score.
Example format:
accuracy: 0.8
clarity: 1.0

Respond with scores for ALL criteria, one per line.`

// judgeNewOp is the operation name used in errors from NewJudgeMetric.
const judgeNewOp = "judge.new"

// Compile-time interface check.
var _ eval.Metric = (*JudgeMetric)(nil)

// judgeOptions holds configuration for JudgeMetric.
type judgeOptions struct {
	rubric     *Rubric
	model      llm.ChatModel
	metricName string
	systemMsg  string
}

// JudgeOption configures a JudgeMetric.
type JudgeOption func(*judgeOptions)

// WithRubric sets the evaluation rubric.
func WithRubric(r *Rubric) JudgeOption {
	return func(o *judgeOptions) {
		o.rubric = r
	}
}

// WithModel sets the LLM used as judge.
func WithModel(m llm.ChatModel) JudgeOption {
	return func(o *judgeOptions) {
		o.model = m
	}
}

// WithMetricName sets the name returned by Name(). Defaults to "judge".
func WithMetricName(name string) JudgeOption {
	return func(o *judgeOptions) {
		o.metricName = name
	}
}

// WithSystemMessage sets an optional system message prepended to the judge prompt.
func WithSystemMessage(msg string) JudgeOption {
	return func(o *judgeOptions) {
		o.systemMsg = msg
	}
}

// JudgeMetric evaluates samples using an LLM as judge against a structured
// rubric. It implements eval.Metric, returning a weighted average score
// across all rubric criteria.
type JudgeMetric struct {
	opts judgeOptions
}

// NewJudgeMetric creates a new JudgeMetric with the given options.
func NewJudgeMetric(opts ...JudgeOption) (*JudgeMetric, error) {
	o := judgeOptions{metricName: "judge"}
	for _, opt := range opts {
		opt(&o)
	}
	if o.model == nil {
		return nil, core.NewError(judgeNewOp, core.ErrInvalidInput, "model is required", nil)
	}
	if o.rubric == nil {
		return nil, core.NewError(judgeNewOp, core.ErrInvalidInput, "rubric is required", nil)
	}
	if err := o.rubric.Validate(); err != nil {
		return nil, core.NewError(judgeNewOp, core.ErrInvalidInput, "invalid rubric", err)
	}
	return &JudgeMetric{opts: o}, nil
}

// Name returns the metric name.
func (j *JudgeMetric) Name() string { return j.opts.metricName }

// Score evaluates a single sample using the LLM judge and returns a weighted
// average score in [0.0, 1.0].
func (j *JudgeMetric) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	expectedCtx := ""
	if sample.ExpectedOutput != "" {
		expectedCtx = fmt.Sprintf("Expected/Reference Answer: %s", sample.ExpectedOutput)
	}

	prompt := fmt.Sprintf(judgePromptTemplate,
		j.opts.rubric.ToPrompt(),
		sample.Input,
		sample.Output,
		expectedCtx,
	)

	var msgs []schema.Message
	if j.opts.systemMsg != "" {
		msgs = append(msgs, schema.NewSystemMessage(j.opts.systemMsg))
	}
	msgs = append(msgs, schema.NewHumanMessage(prompt))

	resp, err := j.opts.model.Generate(ctx, msgs)
	if err != nil {
		return 0, core.NewError("judge.score", core.ErrToolFailed, "llm generate failed", err)
	}

	scores, err := parseJudgeResponse(resp.Text(), j.opts.rubric)
	if err != nil {
		return 0, core.NewError("judge.score", core.ErrInvalidInput, "parse response failed", err)
	}

	return weightedAverage(scores, j.opts.rubric), nil
}

// parseJudgeResponse extracts per-criterion scores from the LLM response.
func parseJudgeResponse(text string, rubric *Rubric) (map[string]float64, error) {
	scores := make(map[string]float64)
	lines := strings.Split(strings.TrimSpace(text), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		val, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		if val < 0 {
			val = 0
		}
		if val > 1 {
			val = 1
		}
		scores[name] = val
	}

	// Verify we got scores for all criteria.
	for _, c := range rubric.Criteria {
		if _, ok := scores[c.Name]; !ok {
			return nil, fmt.Errorf("missing score for criterion %q", c.Name)
		}
	}
	return scores, nil
}

// weightedAverage computes the weighted average across all criteria.
func weightedAverage(scores map[string]float64, rubric *Rubric) float64 {
	totalWeight := rubric.totalWeight()
	if totalWeight == 0 {
		return 0
	}
	var sum float64
	for _, c := range rubric.Criteria {
		sum += scores[c.Name] * c.Weight
	}
	return sum / totalWeight
}
