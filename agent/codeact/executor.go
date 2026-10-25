package codeact

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// opExecute is the operation name used in error codes emitted by code executors.
const opExecute = "codeact.execute"

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

// ProcessExecutor runs code via os/exec with a timeout. It maps language
// names to interpreter commands.
//
// It enforces no memory, CPU or network limits and provides no isolation:
// the code runs with the privileges of the parent process. Each run gets a
// fresh temporary working directory and a minimal environment (PATH, HOME
// and TMPDIR pointing at that directory, a UTF-8 locale), so the parent's
// environment variables such as API keys are not passed on.
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
			opExecute,
			core.ErrInvalidInput,
			fmt.Sprintf("unsupported language %q (supported: %v)", action.Language, e.supportedLanguages()),
			nil,
		)
	}

	if action.Code == "" {
		return CodeResult{}, core.NewError(
			opExecute,
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
	workDir, err := os.MkdirTemp("", "beluga-codeact-*")
	if err != nil {
		return CodeResult{}, core.NewError(opExecute, core.ErrToolFailed, "failed to create working directory", err)
	}
	defer os.RemoveAll(workDir)

	// The ProcessExecutor should only be used in sandboxed environments.
	// The interpreter is from a whitelisted map controlled by the
	// application, not user input.
	cmd := exec.CommandContext(execCtx, interpreter) // #nosec G204 -- interpreter comes from the application-controlled map
	cmd.Stdin = bytes.NewReader([]byte(action.Code))
	cmd.Dir = workDir
	cmd.Env = minimalEnv(workDir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
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
				opExecute,
				core.CodeOf(execCtx.Err(), core.ErrTimeout),
				fmt.Sprintf("code execution timed out after %v", timeout),
				execCtx.Err(),
			)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, core.NewError(
			opExecute,
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
	sort.Strings(langs)
	return langs
}

// minimalEnv is the environment given to interpreter processes: PATH so the
// interpreter can be found, HOME/TMPDIR inside the run's directory, and a
// UTF-8 locale. On Windows SYSTEMROOT is kept as well.
func minimalEnv(dir string) []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + dir,
		"TMPDIR=" + dir,
		"LANG=C.UTF-8",
	}
	if root := os.Getenv("SYSTEMROOT"); root != "" {
		env = append(env, "SYSTEMROOT="+root)
	}
	return env
}
