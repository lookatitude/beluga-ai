package sandbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

// Compile-time interface check.
var _ tool.Tool = (*SandboxTool)(nil)

// SandboxTool wraps a Sandbox as a tool.Tool, making it callable by agents.
// The tool accepts JSON input with "code" and "language" fields and returns
// the execution result as structured JSON text.
type SandboxTool struct {
	sandbox Sandbox
	timeout time.Duration
}

// SandboxToolOption configures a SandboxTool.
type SandboxToolOption func(*SandboxTool)

// WithToolTimeout sets the default execution timeout for the sandbox tool.
// This can be overridden by the SandboxConfig timeout if the sandbox
// provider supports it. Default is 30 seconds.
func WithToolTimeout(d time.Duration) SandboxToolOption {
	return func(t *SandboxTool) { t.timeout = d }
}

// NewSandboxTool creates a new SandboxTool wrapping the given Sandbox.
func NewSandboxTool(sb Sandbox, opts ...SandboxToolOption) *SandboxTool {
	t := &SandboxTool{
		sandbox: sb,
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *SandboxTool) Name() string { return "code_sandbox" }

// Description returns a human-readable description for LLM tool selection.
func (t *SandboxTool) Description() string {
	return "Execute code in a sandboxed environment. Supports python, javascript, bash, sh, ruby, and go."
}

// InputSchema returns the JSON Schema for the tool's input parameters.
func (t *SandboxTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "The source code to execute",
			},
			"language": map[string]any{
				"type":        "string",
				"description": "Programming language (python, javascript, bash, sh, ruby, go)",
				"enum":        []string{"python", "javascript", "bash", "sh", "ruby", "go"},
			},
		},
		"required": []string{"code", "language"},
	}
}

// Execute runs code in the sandbox. Input must contain "code" (string) and
// "language" (string) fields. Returns a tool.Result with the execution output
// as JSON text.
func (t *SandboxTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	code, ok := input["code"].(string)
	if !ok || code == "" {
		return nil, core.NewError("sandbox.tool", core.ErrInvalidInput, "input must contain a non-empty 'code' string", nil)
	}

	language, ok := input["language"].(string)
	if !ok || language == "" {
		return nil, core.NewError("sandbox.tool", core.ErrInvalidInput, "input must contain a non-empty 'language' string", nil)
	}

	cfg := SandboxConfig{
		Language: language,
		Timeout:  t.timeout,
	}

	result, err := t.sandbox.Execute(ctx, code, cfg)
	if err != nil {
		return tool.ErrorResult(core.Errorf(core.ErrToolFailed, "sandbox execution failed: %w", err)), nil
	}

	// Marshal the result as JSON for structured output.
	output := map[string]any{
		"output":    result.Output,
		"error":     result.Error,
		"exit_code": result.ExitCode,
		"duration":  result.Duration.String(),
	}
	data, marshalErr := json.Marshal(output)
	if marshalErr != nil {
		return tool.ErrorResult(core.Errorf(core.ErrInvalidInput, "failed to marshal result: %w", marshalErr)), nil
	}

	r := tool.TextResult(string(data))
	if result.ExitCode != 0 {
		r.IsError = true
	}
	return r, nil
}
