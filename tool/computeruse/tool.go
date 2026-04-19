package computeruse

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// Compile-time interface check.
var _ tool.Tool = (*ComputerUseTool)(nil)

// toolOptions holds configuration for ComputerUseTool.
type toolOptions struct {
	action   ComputerAction
	browser  BrowserBackend
	analyzer *ScreenAnalyzer
	guard    *SafetyGuard
}

// ToolOption configures a ComputerUseTool.
type ToolOption func(*toolOptions)

// WithAction sets the computer action backend.
func WithAction(a ComputerAction) ToolOption {
	return func(o *toolOptions) {
		o.action = a
	}
}

// WithBrowser sets the browser backend.
func WithBrowser(b BrowserBackend) ToolOption {
	return func(o *toolOptions) {
		o.browser = b
	}
}

// WithScreenAnalyzer sets the screen analyzer for describing screenshots.
func WithScreenAnalyzer(a *ScreenAnalyzer) ToolOption {
	return func(o *toolOptions) {
		o.analyzer = a
	}
}

// WithSafetyGuard sets the safety guard for URL and rate limiting checks.
func WithSafetyGuard(g *SafetyGuard) ToolOption {
	return func(o *toolOptions) {
		o.guard = g
	}
}

// ComputerUseTool implements tool.Tool for computer use and browser
// automation. It dispatches actions to the configured backend and applies
// safety guards.
type ComputerUseTool struct {
	opts toolOptions
}

// NewComputerUseTool creates a new ComputerUseTool with the given options.
func NewComputerUseTool(opts ...ToolOption) (*ComputerUseTool, error) {
	o := toolOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	if o.action == nil && o.browser == nil {
		return nil, core.NewError("computeruse.tool.new", core.ErrInvalidInput,
			"at least one of action or browser backend is required", nil)
	}
	return &ComputerUseTool{opts: o}, nil
}

// Name returns "computer_use".
func (t *ComputerUseTool) Name() string { return "computer_use" }

// Description returns a description of the computer use tool.
func (t *ComputerUseTool) Description() string {
	return "Interact with a computer screen: take screenshots, click, type, scroll, navigate web pages, and execute browser actions."
}

// InputSchema returns the JSON Schema for the tool's input. The set of
// advertised actions is narrowed to those the configured backends can
// actually execute — for example, key_press is excluded when only a
// browser backend is wired up, because key_press dispatches exclusively
// through a ComputerAction backend.
func (t *ComputerUseTool) InputSchema() map[string]any {
	actions := []string{"screenshot", "click", "type", "scroll"}
	if t.opts.action != nil {
		actions = append(actions, "key_press")
	}
	if t.opts.browser != nil {
		actions = append(actions, "navigate")
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        actions,
				"description": "The action to perform",
			},
			"x":            map[string]any{"type": "integer", "description": "X coordinate for click/scroll"},
			"y":            map[string]any{"type": "integer", "description": "Y coordinate for click/scroll"},
			"text":         map[string]any{"type": "string", "description": "Text to type or key to press"},
			"url":          map[string]any{"type": "string", "description": "URL to navigate to"},
			"scroll_delta": map[string]any{"type": "integer", "description": "Scroll amount (positive=down)"},
		},
		"required": []string{"action"},
	}
}

// Execute runs the requested computer action.
func (t *ComputerUseTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	actionStr, _ := input["action"].(string)
	if actionStr == "" {
		return tool.ErrorResult(fmt.Errorf("action is required")), nil
	}

	// Check rate limit via safety guard.
	if t.opts.guard != nil {
		if err := t.opts.guard.CheckAction(); err != nil {
			return tool.ErrorResult(err), nil
		}
	}

	switch actionStr {
	case "navigate":
		return t.handleNavigate(ctx, input)
	case "screenshot":
		return t.handleScreenshot(ctx)
	case "click":
		return t.handleClick(ctx, input)
	case "type":
		return t.handleType(ctx, input)
	case "scroll":
		return t.handleScroll(ctx, input)
	case "key_press":
		return t.handleKeyPress(ctx, input)
	default:
		return tool.ErrorResult(fmt.Errorf("unknown action: %s", actionStr)), nil
	}
}

func (t *ComputerUseTool) handleNavigate(ctx context.Context, input map[string]any) (*tool.Result, error) {
	rawURL, _ := input["url"].(string)
	if rawURL == "" {
		return tool.ErrorResult(fmt.Errorf("url is required for navigate action")), nil
	}

	if t.opts.guard != nil {
		if err := t.opts.guard.CheckURL(rawURL); err != nil {
			return tool.ErrorResult(err), nil
		}
	}

	if t.opts.browser == nil {
		return tool.ErrorResult(fmt.Errorf("browser backend not configured")), nil
	}

	if err := t.opts.browser.Navigate(ctx, rawURL); err != nil {
		return tool.ErrorResult(fmt.Errorf("navigate failed: %w", err)), nil
	}

	return tool.TextResult(fmt.Sprintf("Navigated to %s", rawURL)), nil
}

func (t *ComputerUseTool) handleScreenshot(ctx context.Context) (*tool.Result, error) {
	var screenshot []byte
	var err error

	if t.opts.browser != nil {
		screenshot, err = t.opts.browser.Screenshot(ctx)
	} else if t.opts.action != nil {
		screenshot, err = t.opts.action.Screenshot(ctx)
	} else {
		return tool.ErrorResult(fmt.Errorf("no backend configured for screenshots")), nil
	}

	if err != nil {
		return tool.ErrorResult(fmt.Errorf("screenshot failed: %w", err)), nil
	}

	parts := []schema.ContentPart{
		schema.ImagePart{Data: screenshot, MimeType: "image/png"},
	}

	// If analyzer is available, add description.
	if t.opts.analyzer != nil {
		desc, analyzeErr := t.opts.analyzer.Analyze(ctx, screenshot)
		if analyzeErr == nil {
			parts = append(parts, schema.TextPart{Text: desc})
		}
	}

	return &tool.Result{Content: parts}, nil
}

func (t *ComputerUseTool) handleClick(ctx context.Context, input map[string]any) (*tool.Result, error) {
	x, y := toInt(input["x"]), toInt(input["y"])

	if t.opts.browser != nil {
		if err := t.opts.browser.Click(ctx, x, y); err != nil {
			return tool.ErrorResult(fmt.Errorf("click failed: %w", err)), nil
		}
		return tool.TextResult(fmt.Sprintf("Clicked at (%d, %d)", x, y)), nil
	}

	if t.opts.action != nil {
		result, err := t.opts.action.Execute(ctx, ActionRequest{Type: ActionClick, X: x, Y: y})
		if err != nil {
			return tool.ErrorResult(fmt.Errorf("click failed: %w", err)), nil
		}
		return tool.TextResult(result.Description), nil
	}

	return tool.ErrorResult(fmt.Errorf("no backend configured for click")), nil
}

func (t *ComputerUseTool) handleType(ctx context.Context, input map[string]any) (*tool.Result, error) {
	text, _ := input["text"].(string)
	if text == "" {
		return tool.ErrorResult(fmt.Errorf("text is required for type action")), nil
	}

	if t.opts.browser != nil {
		if err := t.opts.browser.Type(ctx, text); err != nil {
			return tool.ErrorResult(fmt.Errorf("type failed: %w", err)), nil
		}
		// Do not echo typed text back to the LLM — it may contain
		// passwords, tokens, or other credentials.
		return tool.TextResult(fmt.Sprintf("Typed %d character(s)", len(text))), nil
	}

	if t.opts.action != nil {
		result, err := t.opts.action.Execute(ctx, ActionRequest{Type: ActionTypeText, Text: text})
		if err != nil {
			return tool.ErrorResult(fmt.Errorf("type failed: %w", err)), nil
		}
		return tool.TextResult(result.Description), nil
	}

	return tool.ErrorResult(fmt.Errorf("no backend configured for type")), nil
}

func (t *ComputerUseTool) handleScroll(ctx context.Context, input map[string]any) (*tool.Result, error) {
	x, y := toInt(input["x"]), toInt(input["y"])
	delta := toInt(input["scroll_delta"])

	if eb, ok := t.opts.browser.(ExtendedBrowser); ok {
		if err := eb.Scroll(ctx, x, y, delta); err != nil {
			return tool.ErrorResult(fmt.Errorf("scroll failed: %w", err)), nil
		}
		return tool.TextResult(fmt.Sprintf("Scrolled by %d at (%d, %d)", delta, x, y)), nil
	}

	if t.opts.action != nil {
		result, err := t.opts.action.Execute(ctx, ActionRequest{Type: ActionScroll, X: x, Y: y, ScrollDelta: delta})
		if err != nil {
			return tool.ErrorResult(fmt.Errorf("scroll failed: %w", err)), nil
		}
		return tool.TextResult(result.Description), nil
	}

	return tool.ErrorResult(fmt.Errorf("no backend configured for scroll")), nil
}

func (t *ComputerUseTool) handleKeyPress(ctx context.Context, input map[string]any) (*tool.Result, error) {
	key, _ := input["text"].(string)
	if key == "" {
		return tool.ErrorResult(fmt.Errorf("text (key name) is required for key_press action")), nil
	}

	if t.opts.action != nil {
		result, err := t.opts.action.Execute(ctx, ActionRequest{Type: ActionKeyPress, Text: key})
		if err != nil {
			return tool.ErrorResult(fmt.Errorf("key_press failed: %w", err)), nil
		}
		return tool.TextResult(result.Description), nil
	}

	return tool.ErrorResult(fmt.Errorf("no backend configured for key_press")), nil
}

// toInt converts a JSON number to int, handling both float64 and json.Number.
func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case int:
		return n
	default:
		return 0
	}
}
