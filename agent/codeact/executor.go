package codeact

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// CodeExecutor executes code actions and returns results. Implementations
// must respect context cancellation and enforce timeouts.
type CodeExecutor interface {
	// Execute runs the given code action and returns the result.
	Execute(ctx context.Context, action CodeAction) (CodeResult, error)
}

// Compile-time interface checks.
var (
	_ CodeExecutor = (*NoopExecutor)(nil)
	_ CodeExecutor = (*ProcessExecutor)(nil)
)

// NoopExecutor returns the code itself as output without executing it.
// Useful for testing and dry-run scenarios.
type NoopExecutor struct{}

// NewNoopExecutor creates a new NoopExecutor.
func NewNoopExecutor() *NoopExecutor {
	return &NoopExecutor{}
}

// Execute returns the code as the output without running it.
func (e *NoopExecutor) Execute(_ context.Context, action CodeAction) (CodeResult, error) {
	return CodeResult{
		Output:   action.Code,
		ExitCode: 0,
		Duration: 0,
	}, nil
}

// ProcessExecutor runs code via os/exec with timeout and resource limits.
// It maps language names to interpreter commands.
//
// Security note: ProcessExecutor should only be used in sandboxed environments.
// Never pass untrusted user input directly as code without proper sandboxing.
type ProcessExecutor struct {
	// interpreters maps language name to command (e.g., "python" -> "python3").
	interpreters map[string]string
	// defaultTimeout is the fallback timeout when CodeAction.Timeout is zero.
	defaultTimeout time.Duration
}

// ProcessExecutorOption configures a ProcessExecutor.
type ProcessExecutorOption func(*ProcessExecutor)

// WithInterpreter registers a command for a language name.
func WithInterpreter(language, command string) ProcessExecutorOption {
	return func(e *ProcessExecutor) {
		e.interpreters[language] = command
	}
}

// WithDefaultTimeout sets the default execution timeout.
func WithDefaultTimeout(d time.Duration) ProcessExecutorOption {
	return func(e *ProcessExecutor) {
		if d > 0 {
			e.defaultTimeout = d
		}
	}
}

// NewProcessExecutor creates a new ProcessExecutor with the given options.
// Default interpreters: python -> python3, javascript -> node.
func NewProcessExecutor(opts ...ProcessExecutorOption) *ProcessExecutor {
	e := &ProcessExecutor{
		interpreters: map[string]string{
			"python":     "python3",
			"javascript": "node",
		},
		defaultTimeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Execute runs code by writing it to stdin of the appropriate interpreter.
// It enforces timeouts via context and returns stdout/stderr.
func (e *ProcessExecutor) Execute(ctx context.Context, action CodeAction) (CodeResult, error) {
	interpreter, ok := e.interpreters[action.Language]
	if !ok {
		return CodeResult{}, core.NewError(
			"codeact.execute",
			core.ErrInvalidInput,
			fmt.Sprintf("unsupported language %q (supported: %v)", action.Language, e.supportedLanguages()),
			nil,
		)
	}

	if action.Code == "" {
		return CodeResult{}, core.NewError(
			"codeact.execute",
			core.ErrInvalidInput,
			"empty code",
			nil,
		)
	}

	timeout := action.Timeout
	if timeout == 0 {
		timeout = e.defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	// Run through interpreter with code on stdin.
	// Security: code comes from the LLM, not directly from user input.
	// The ProcessExecutor should only be used in sandboxed environments.
	// #nosec G204 -- interpreter is from a whitelisted map controlled by the
	// application, not user input. ProcessExecutor is meant for sandboxed use.
	cmd := exec.CommandContext(execCtx, interpreter) //nolint:gosec // sandboxed execution
	cmd.Stdin = bytes.NewReader([]byte(action.Code))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	result := CodeResult{
		Output:   stdout.String(),
		Error:    stderr.String(),
		Duration: duration,
	}

	if err != nil {
		// Check context first: when exec.CommandContext kills a process due
		// to a deadline, cmd.Run() returns *exec.ExitError rather than the
		// context error, so we must branch on ctx state before the type check.
		if execCtx.Err() != nil {
			return result, core.NewError(
				"codeact.execute",
				core.ErrTimeout,
				fmt.Sprintf("code execution timed out after %v", timeout),
				execCtx.Err(),
			)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, core.NewError(
			"codeact.execute",
			core.ErrToolFailed,
			"code execution failed",
			err,
		)
	}

	return result, nil
}

// supportedLanguages returns a sorted list of registered language names.
func (e *ProcessExecutor) supportedLanguages() []string {
	langs := make([]string, 0, len(e.interpreters))
	for lang := range e.interpreters {
		langs = append(langs, lang)
	}
	return langs
}
