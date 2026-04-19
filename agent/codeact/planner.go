package codeact

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// ActionCode is the action type for code execution actions.
const ActionCode agent.ActionType = "code"

func init() {
	agent.RegisterPlanner("codeact", func(cfg agent.PlannerConfig) (agent.Planner, error) {
		if cfg.LLM == nil {
			return nil, fmt.Errorf("codeact planner requires an LLM")
		}
		opts := optsFromExtra(cfg.Extra)
		return NewCodeActPlanner(cfg.LLM, opts...), nil
	})
}

// optsFromExtra extracts CodeActPlanner options from PlannerConfig.Extra.
func optsFromExtra(extra map[string]any) []PlannerOption {
	var opts []PlannerOption
	if extra == nil {
		return opts
	}
	if lang, ok := extra["language"].(string); ok && lang != "" {
		opts = append(opts, WithPlannerLanguage(lang))
	}
	if imports, ok := extra["allowed_imports"].([]string); ok {
		opts = append(opts, WithAllowedImports(imports))
	}
	return opts
}

// codeBlockPattern matches fenced code blocks: ```lang\ncode\n```
var codeBlockPattern = regexp.MustCompile("(?s)```(\\w*)\\n(.*?)\\n```")

// CodeActPlanner implements agent.Planner using the CodeAct pattern.
// It instructs the LLM to emit code blocks and parses them into ActionCode actions.
type CodeActPlanner struct {
	llm            llm.ChatModel
	language       string
	allowedImports []string
}

// Compile-time interface check.
var _ agent.Planner = (*CodeActPlanner)(nil)

// PlannerOption configures a CodeActPlanner.
type PlannerOption func(*CodeActPlanner)

// WithPlannerLanguage sets the preferred programming language.
func WithPlannerLanguage(lang string) PlannerOption {
	return func(p *CodeActPlanner) {
		p.language = lang
	}
}

// WithAllowedImports sets the list of allowed imports/modules.
func WithAllowedImports(imports []string) PlannerOption {
	return func(p *CodeActPlanner) {
		p.allowedImports = imports
	}
}

// NewCodeActPlanner creates a new CodeActPlanner with the given LLM and options.
func NewCodeActPlanner(model llm.ChatModel, opts ...PlannerOption) *CodeActPlanner {
	p := &CodeActPlanner{
		llm:      model,
		language: "python",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Plan generates actions from the initial state by querying the LLM.
func (p *CodeActPlanner) Plan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	return p.generate(ctx, state)
}

// Replan generates updated actions based on new observations.
func (p *CodeActPlanner) Replan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	return p.generate(ctx, state)
}

// generate queries the LLM and parses the response into actions.
func (p *CodeActPlanner) generate(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
	messages := p.buildMessages(state)

	resp, err := p.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("codeact planner: generate failed: %w", err)
	}

	return p.parseResponse(resp), nil
}

// buildMessages constructs the message list, injecting the CodeAct system prompt.
func (p *CodeActPlanner) buildMessages(state agent.PlannerState) []schema.Message {
	msgs := make([]schema.Message, 0, len(state.Messages)+2)

	// Inject CodeAct system instruction
	msgs = append(msgs, schema.NewSystemMessage(p.systemPrompt()))

	// Add conversation history, skipping any existing system messages to avoid conflicts.
	for _, m := range state.Messages {
		if _, ok := m.(*schema.SystemMessage); ok {
			continue
		}
		msgs = append(msgs, m)
	}

	// Add observation results from previous code executions.
	for _, obs := range state.Observations {
		if obs.Action.Type != ActionCode {
			continue
		}
		msgs = append(msgs, observationToMessage(obs))
	}

	return msgs
}

// observationToMessage renders a code-execution observation as a human message
// so the LLM can see the executed code and its result on the next planner turn.
func observationToMessage(obs agent.Observation) schema.Message {
	code, _ := obs.Action.Metadata["code"].(string)
	lang, _ := obs.Action.Metadata["language"].(string)

	resultText := extractResultText(obs.Result)
	if resultText == "" && obs.Error != nil {
		resultText = obs.Error.Error()
	}

	obsMsg := fmt.Sprintf("Code executed (%s):\n```%s\n%s\n```\n\nResult:\n%s", lang, lang, code, resultText)
	return schema.NewHumanMessage(obsMsg)
}

// extractResultText returns the first text part from a tool result, or "".
func extractResultText(result *tool.Result) string {
	if result == nil {
		return ""
	}
	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}

// systemPrompt returns the system message instructing the LLM to use CodeAct.
func (p *CodeActPlanner) systemPrompt() string {
	var sb strings.Builder
	sb.WriteString("You are a code-executing assistant. To solve tasks, write executable code in fenced code blocks.\n\n")
	sb.WriteString("Rules:\n")
	sb.WriteString(fmt.Sprintf("- Use %s as the programming language.\n", p.language))
	sb.WriteString("- Wrap all code in fenced code blocks with the language tag: ```" + p.language + "\n")
	sb.WriteString("- Print results to stdout so they can be captured.\n")
	sb.WriteString("- If you need multiple steps, write one code block at a time and wait for the result.\n")
	sb.WriteString("- When you have the final answer, respond with plain text (no code block).\n")

	if len(p.allowedImports) > 0 {
		sb.WriteString(fmt.Sprintf("- Only use these imports/modules: %s\n", strings.Join(p.allowedImports, ", ")))
	}

	return sb.String()
}

// parseResponse extracts code blocks from the LLM response and creates actions.
// If no code blocks are found, the response text is treated as a final answer.
func (p *CodeActPlanner) parseResponse(resp *schema.AIMessage) []agent.Action {
	text := resp.Text()
	blocks := ExtractCodeBlocks(text)

	if len(blocks) == 0 {
		return []agent.Action{{
			Type:    agent.ActionFinish,
			Message: text,
		}}
	}

	actions := make([]agent.Action, 0, len(blocks))
	for _, block := range blocks {
		lang := block.Language
		if lang == "" {
			lang = p.language
		}
		actions = append(actions, agent.Action{
			Type:    ActionCode,
			Message: block.Code,
			Metadata: map[string]any{
				"language": lang,
				"code":     block.Code,
			},
		})
	}
	return actions
}

// CodeBlock represents a parsed fenced code block.
type CodeBlock struct {
	// Language is the language tag from the code fence (may be empty).
	Language string
	// Code is the source code content.
	Code string
}

// ExtractCodeBlocks parses fenced code blocks from text.
// Returns all matched blocks in order.
func ExtractCodeBlocks(text string) []CodeBlock {
	matches := codeBlockPattern.FindAllStringSubmatch(text, -1)
	blocks := make([]CodeBlock, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 3 {
			blocks = append(blocks, CodeBlock{
				Language: match[1],
				Code:     match[2],
			})
		}
	}
	return blocks
}
