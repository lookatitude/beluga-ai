// Package shell provides a tool implementation for executing shell commands.
package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// ShellTool allows executing shell commands.
// WARNING: Executing arbitrary shell commands based on LLM output is inherently risky.
// Use with extreme caution and proper sandboxing.
type ShellTool struct {
	tools.BaseTool
	Def     tools.ToolDefinition // Store definition directly
	Timeout time.Duration        // Optional timeout for command execution
}

// DefaultInputSchema defines a simple schema expecting a "command" string.
const DefaultInputSchema = `{"type": "object", "properties": {"command": {"type": "string", "description": "The shell command to execute."}}, "required": ["command"]}`

// NewShellTool creates a new ShellTool.
func NewShellTool(timeout time.Duration) (*ShellTool, error) {
	var inputSchema map[string]any // Changed to map[string]any
	if err := json.Unmarshal([]byte(DefaultInputSchema), &inputSchema); err != nil {
		// This should not happen with a valid constant
		return nil, fmt.Errorf("internal error: failed to parse default schema: %w", err)
	}

	return &ShellTool{
		Def: tools.ToolDefinition{
			Name:        "shell",
			Description: "Executes a shell command. Input must be a JSON object with a \"command\" key.",
			InputSchema: inputSchema,
		},
		Timeout: timeout,
	}, nil
}

// Definition returns the tool's definition.
func (st *ShellTool) Definition() tools.ToolDefinition {
	return st.Def
}

// Description returns the tool's description.
func (st *ShellTool) Description() string {
	return st.Definition().Description
}

// Name returns the tool's name.
func (st *ShellTool) Name() string {
	return st.Definition().Name
}

// Execute runs the shell command.
// Corrected input type to any and return type to any
func (st *ShellTool) Execute(ctx context.Context, input any) (any, error) {
	var commandStr string
	// Handle input type: map[string]any or string
	switch v := input.(type) {
	case map[string]any:
		cmdVal, ok := v["command"].(string)
		if !ok || cmdVal == "" {
			return nil, fmt.Errorf("invalid input map: missing or empty \"command\" string")
		}
		commandStr = cmdVal
	case string:
		// Allow direct string input for convenience, assuming it's the command
		if v == "" {
			return nil, fmt.Errorf("invalid input string: command cannot be empty")
		}
		commandStr = v
	default:
		return nil, fmt.Errorf("invalid input type: expected map[string]any or string, got %T", input)
	}

	// Add timeout to context if specified
	execCtx := ctx
	if st.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, st.Timeout)
		defer cancel()
	}

	// Execute the command using sh -c
	// SECURITY WARNING: This executes the command string directly. Ensure proper sanitization
	// or sandboxing if the command string originates from untrusted input (like an LLM).
	cmd := exec.CommandContext(execCtx, "sh", "-c", commandStr)

	outputBytes, err := cmd.CombinedOutput() // Get both stdout and stderr
	output := string(outputBytes)

	if execCtx.Err() == context.DeadlineExceeded {
		// Return timeout as part of output string, not error
		return fmt.Sprintf("Command timed out after %s. Output:\n%s", st.Timeout, output), nil
	}

	if err != nil {
		// Return command error as part of output string, not error
		return fmt.Sprintf("Command failed with error: %v. Output:\n%s", err, output), nil
	}

	return strings.TrimSpace(output), nil
}

// Implement core.Runnable Invoke for ShellTool
func (st *ShellTool) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// Execute now takes any
	return st.Execute(ctx, input)
}

// Batch implementation
// Batch implements the tools.Tool interface
func (st *ShellTool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := st.Execute(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error processing batch item %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Run implements the core.Runnable Batch method with options
func (st *ShellTool) Run(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	return st.Batch(ctx, inputs) // Options are ignored for now
}

// Stream is not applicable for shell commands.
func (st *ShellTool) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	resultChan := make(chan any, 1)
	go func() {
		defer close(resultChan)
		output, err := st.Invoke(ctx, input, options...)
		if err != nil {
			resultChan <- err
		} else {
			resultChan <- output
		}
	}()
	return resultChan, nil
}

// Ensure implementation satisfies interfaces
// Make sure interfaces are correctly implemented
var _ tools.Tool = (*ShellTool)(nil)
// Define a custom interface that matches what we've implemented
type batcherWithOptions interface {
	Run(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
	Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error)
	Invoke(ctx context.Context, input any, options ...core.Option) (any, error)
}
var _ batcherWithOptions = (*ShellTool)(nil)
