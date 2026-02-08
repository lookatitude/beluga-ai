package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	RegisterPlanner("reflexion", func(cfg PlannerConfig) (Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("reflexion planner requires an LLM")
		}
		opts := []ReflexionOption{}
		if eval, ok := cfg.Extra["evaluator"].(llm.ChatModel); ok {
			opts = append(opts, WithEvaluator(eval))
		}
		if threshold, ok := cfg.Extra["threshold"].(float64); ok {
			opts = append(opts, WithThreshold(threshold))
		}
		if maxReflections, ok := cfg.Extra["max_reflections"].(int); ok {
			opts = append(opts, WithMaxReflections(maxReflections))
		}
		return NewReflexionPlanner(cfg.LLM, opts...), nil
	})
}

// ReflexionPlanner implements the Reflexion strategy with Actor-Evaluator-Reflector.
// The actor generates a response, the evaluator scores it, and if the score is
// below a threshold, the reflector produces self-reflection that guides the next
// attempt.
type ReflexionPlanner struct {
	actor          llm.ChatModel
	evaluator      llm.ChatModel
	threshold      float64
	maxReflections int
	reflections    []string
}

// ReflexionOption configures a ReflexionPlanner.
type ReflexionOption func(*ReflexionPlanner)

// WithEvaluator sets a separate LLM for evaluation. If not set, the actor LLM
// is used for evaluation.
func WithEvaluator(model llm.ChatModel) ReflexionOption {
	return func(p *ReflexionPlanner) {
		p.evaluator = model
	}
}

// WithThreshold sets the minimum score threshold (0.0-1.0). Responses scoring
// below this trigger self-reflection.
func WithThreshold(t float64) ReflexionOption {
	return func(p *ReflexionPlanner) {
		p.threshold = t
	}
}

// WithMaxReflections sets the maximum number of reflection attempts.
func WithMaxReflections(n int) ReflexionOption {
	return func(p *ReflexionPlanner) {
		if n > 0 {
			p.maxReflections = n
		}
	}
}

// NewReflexionPlanner creates a new Reflexion planner.
func NewReflexionPlanner(actor llm.ChatModel, opts ...ReflexionOption) *ReflexionPlanner {
	p := &ReflexionPlanner{
		actor:          actor,
		threshold:      0.7,
		maxReflections: 3,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.evaluator == nil {
		p.evaluator = actor
	}
	return p
}

// Plan generates actions using the actor, evaluates, and reflects if needed.
func (p *ReflexionPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.generateWithReflection(ctx, state)
}

// Replan re-generates with accumulated reflections.
func (p *ReflexionPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	return p.generateWithReflection(ctx, state)
}

func (p *ReflexionPlanner) generateWithReflection(ctx context.Context, state PlannerState) ([]Action, error) {
	messages := buildMessagesFromState(state)

	// Add prior reflections as system context
	if len(p.reflections) > 0 {
		reflectionText := "Previous reflections on your responses:\n" + strings.Join(p.reflections, "\n---\n")
		messages = append([]schema.Message{
			schema.NewSystemMessage(reflectionText),
		}, messages...)
	}

	// Bind tools
	model := p.actor
	if len(state.Tools) > 0 {
		model = model.BindTools(toolDefinitions(state.Tools))
	}

	// Actor generates
	resp, err := model.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("reflexion actor: %w", err)
	}

	// If tool calls, skip evaluation and return tool actions directly
	if len(resp.ToolCalls) > 0 {
		return parseAIResponse(resp), nil
	}

	// Evaluate the response
	score, err := p.evaluate(ctx, state.Input, resp.Text())
	if err != nil {
		// If evaluation fails, proceed with the response anyway
		return parseAIResponse(resp), nil
	}

	// If score is above threshold, accept the response
	if score >= p.threshold {
		return parseAIResponse(resp), nil
	}

	// Reflect and retry (up to maxReflections)
	if len(p.reflections) >= p.maxReflections {
		// Max reflections reached, accept current response
		return parseAIResponse(resp), nil
	}

	reflection, err := p.reflect(ctx, state.Input, resp.Text(), score)
	if err != nil {
		return parseAIResponse(resp), nil
	}

	p.reflections = append(p.reflections, reflection)

	// Retry with reflection
	retryMsgs := make([]schema.Message, 0, len(messages)+2)
	retryMsgs = append(retryMsgs, messages...)
	retryMsgs = append(retryMsgs, resp)
	retryMsgs = append(retryMsgs, schema.NewHumanMessage(
		fmt.Sprintf("Your previous response scored %.2f/1.0. Reflection: %s\nPlease try again with an improved response.", score, reflection),
	))

	retryResp, err := model.Generate(ctx, retryMsgs)
	if err != nil {
		// If retry fails, return original response
		return parseAIResponse(resp), nil
	}

	return parseAIResponse(retryResp), nil
}

// evaluate scores the response quality using the evaluator LLM.
func (p *ReflexionPlanner) evaluate(ctx context.Context, input, response string) (float64, error) {
	evalPrompt := fmt.Sprintf(
		"Evaluate the following response to the given input on a scale of 0.0 to 1.0.\n"+
			"Input: %s\n\nResponse: %s\n\n"+
			"Reply with ONLY a number between 0.0 and 1.0 representing the quality score.",
		input, response,
	)

	resp, err := p.evaluator.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("You are a response quality evaluator. Output only a decimal number between 0.0 and 1.0."),
		schema.NewHumanMessage(evalPrompt),
	})
	if err != nil {
		return 0, fmt.Errorf("evaluate: %w", err)
	}

	text := strings.TrimSpace(resp.Text())
	var score float64
	if _, err := fmt.Sscanf(text, "%f", &score); err != nil {
		return 0.5, nil // Default to middle score if parse fails
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score, nil
}

// reflect generates a self-reflection on why the response was inadequate.
func (p *ReflexionPlanner) reflect(ctx context.Context, input, response string, score float64) (string, error) {
	reflectPrompt := fmt.Sprintf(
		"The following response scored %.2f/1.0 for the given input.\n"+
			"Input: %s\n\nResponse: %s\n\n"+
			"Provide a brief, actionable reflection on how to improve the response. "+
			"Focus on what was wrong and how to fix it.",
		score, input, response,
	)

	resp, err := p.evaluator.Generate(ctx, []schema.Message{
		schema.NewSystemMessage("You are a self-reflection assistant. Provide concise, actionable feedback."),
		schema.NewHumanMessage(reflectPrompt),
	})
	if err != nil {
		return "", fmt.Errorf("reflect: %w", err)
	}

	return resp.Text(), nil
}

// Reflections returns the accumulated reflections (for inspection/testing).
func (p *ReflexionPlanner) Reflections() []string {
	return p.reflections
}

// ResetReflections clears accumulated reflections.
func (p *ReflexionPlanner) ResetReflections() {
	p.reflections = nil
}
