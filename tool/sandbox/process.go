package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// opSandboxExecute is the core.Error operation name for sandbox execution failures.
const opSandboxExecute = "sandbox.execute"

// Compile-time interface check.
var _ Sandbox = (*ProcessSandbox)(nil)

// processOptions holds configuration for ProcessSandbox.
type processOptions struct {
	workDir string
	env     []string
	hooks   Hooks
}

// ProcessOption configures a ProcessSandbox.
type ProcessOption func(*processOptions)

// WithWorkDir sets the base working directory for process execution.
// A unique temporary subdirectory is created under this path for each
// execution. If empty, os.TempDir() is used.
func WithWorkDir(dir string) ProcessOption {
	return func(o *processOptions) { o.workDir = dir }
}

// WithEnv sets the environment variables for the executed process.
// Each entry is in the form "KEY=VALUE". If nil, the process inherits
// a minimal environment.
func WithEnv(env []string) ProcessOption {
	return func(o *processOptions) { o.env = env }
}

// WithProcessHooks sets lifecycle hooks on the ProcessSandbox. Hooks fire
// around each Execute call — BeforeExecute, AfterExecute, OnTimeout and
// OnError — and are composable via ComposeHooks.
func WithProcessHooks(h Hooks) ProcessOption {
	return func(o *processOptions) { o.hooks = h }
}

// ProcessSandbox executes code as a child process in a temporary directory.
//
// WARNING: This implementation is for DEVELOPMENT AND TESTING ONLY. It does NOT
// provide kernel-level isolation (no seccomp, no namespaces, no cgroups). The
// executed code runs with the same privileges as the parent process. Do NOT use
// this in production with untrusted code. For production use, register a
// container-based sandbox provider.
type ProcessSandbox struct {
	opts processOptions
}

// NewProcessSandbox creates a new ProcessSandbox with the given options.
func NewProcessSandbox(opts ...ProcessOption) *ProcessSandbox {
	o := processOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return &ProcessSandbox{opts: o}
}

// Execute runs code as a child process. It creates a temporary directory,
// writes the code to a file, and executes it with the appropriate interpreter
// for the configured language. Timeout is enforced via context.
func (s *ProcessSandbox) Execute(ctx context.Context, code string, cfg SandboxConfig) (ExecutionResult, error) {
	lang, err := s.preflight(ctx, code, cfg)
	if err != nil {
		return ExecutionResult{}, err
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	maxOutput := cfg.Resources.MaxOutputBytes
	if maxOutput == 0 {
		maxOutput = defaultMaxOutputBytes
	}

	// Create isolated temp directory.
	baseDir := s.opts.workDir
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	tmpDir, err := os.MkdirTemp(baseDir, "beluga-sandbox-*")
	if err != nil {
		return ExecutionResult{}, core.NewError(opSandboxExecute, core.ErrToolFailed, "failed to create temp directory", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write code to file.
	filename := filepath.Join(tmpDir, lang.filename)
	if err := os.WriteFile(filename, []byte(code), 0600); err != nil {
		return ExecutionResult{}, core.NewError(opSandboxExecute, core.ErrToolFailed, "failed to write code file", err)
	}

	// Build command with timeout context.
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Copy lang.args so appending never writes into the shared
	// supportedLanguages backing array.
	args := make([]string, 0, len(lang.args)+1)
	args = append(args, lang.args...)
	args = append(args, filename)
	// The interpreter is selected from a fixed allowlist (supportedLanguages),
	// args are fixed per language, and filename is a server-generated path
	// inside a freshly created temp directory. The user-supplied code is
	// written to the file and never passed as a command argument.
	cmd := exec.CommandContext(execCtx, lang.interpreter, args...) //#nosec G204 -- interpreter from static allowlist, filename is server-generated
	cmd.Dir = tmpDir

	if len(s.opts.env) > 0 {
		cmd.Env = s.opts.env
	}

	var stdout, stderr bytes.Buffer
	stdout.Grow(4096)
	stderr.Grow(4096)
	cmd.Stdout = &limitedWriter{buf: &stdout, max: maxOutput}
	cmd.Stderr = &limitedWriter{buf: &stderr, max: maxOutput}

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)

	result := ExecutionResult{
		Output:   stdout.String(),
		Error:    stderr.String(),
		Duration: duration,
	}

	if runErr != nil {
		return s.handleRunError(ctx, execCtx, runErr, result, code, cfg, timeout)
	}

	s.fireAfterExecute(ctx, result, nil)
	return result, nil
}

// preflight validates the request (non-empty code, enforceable network
// policy), runs the BeforeExecute hook and resolves the language spec. Errors
// from the hook and language resolution pass through the OnError hook.
func (s *ProcessSandbox) preflight(ctx context.Context, code string, cfg SandboxConfig) (languageSpec, error) {
	if code == "" {
		return languageSpec{}, core.NewError(opSandboxExecute, core.ErrInvalidInput, "code must not be empty", nil)
	}

	// ProcessSandbox cannot enforce network isolation. Reject any non-
	// unrestricted network policy to prevent silent misconfiguration.
	if cfg.NetworkPolicy != "" && cfg.NetworkPolicy != NetworkUnrestricted {
		return languageSpec{}, core.NewError(
			opSandboxExecute,
			core.ErrInvalidInput,
			fmt.Sprintf("ProcessSandbox cannot enforce NetworkPolicy %q; use a container-based provider", cfg.NetworkPolicy),
			nil,
		)
	}

	if h := s.opts.hooks.BeforeExecute; h != nil {
		if err := h(ctx, code, cfg); err != nil {
			return languageSpec{}, s.applyOnError(ctx, err)
		}
	}

	lang, err := resolveLanguage(cfg.Language)
	if err != nil {
		return languageSpec{}, s.applyOnError(ctx, err)
	}
	return lang, nil
}

// handleRunError maps a failed cmd.Run into the sandbox result. Timeouts and
// caller cancellation become errors (after the OnTimeout/OnError hooks); a
// non-zero exit is reported through result.ExitCode with a nil error, since
// the code ran but failed. AfterExecute fires in every case.
func (s *ProcessSandbox) handleRunError(ctx, execCtx context.Context, runErr error, result ExecutionResult, code string, cfg SandboxConfig, timeout time.Duration) (ExecutionResult, error) {
	if execCtx.Err() == context.DeadlineExceeded {
		result.ExitCode = -1
		if th := s.opts.hooks.OnTimeout; th != nil {
			th(ctx, code, cfg)
		}
		timeoutErr := s.applyOnError(ctx, core.NewError(opSandboxExecute, core.ErrTimeout,
			fmt.Sprintf("execution timed out after %s", timeout), execCtx.Err()))
		s.fireAfterExecute(ctx, result, timeoutErr)
		return result, timeoutErr
	}
	if ctx.Err() != nil {
		result.ExitCode = -1
		cancelErr := s.applyOnError(ctx, core.NewError(opSandboxExecute, core.CodeOf(ctx.Err(), core.ErrTimeout), "execution cancelled", ctx.Err()))
		s.fireAfterExecute(ctx, result, cancelErr)
		return result, cancelErr
	}
	// Extract exit code if available.
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	} else {
		result.ExitCode = -1
	}
	// Non-zero exit is not a Go error — it means the code ran but failed.
	s.fireAfterExecute(ctx, result, nil)
	return result, nil
}

// applyOnError passes err through the OnError hook when one is set.
func (s *ProcessSandbox) applyOnError(ctx context.Context, err error) error {
	if oh := s.opts.hooks.OnError; oh != nil {
		return oh(ctx, err)
	}
	return err
}

// fireAfterExecute invokes the AfterExecute hook when one is set.
func (s *ProcessSandbox) fireAfterExecute(ctx context.Context, result ExecutionResult, err error) {
	if ah := s.opts.hooks.AfterExecute; ah != nil {
		ah(ctx, result, err)
	}
}

// Close is a no-op for ProcessSandbox since each execution creates and
// cleans up its own temporary directory.
func (s *ProcessSandbox) Close(_ context.Context) error {
	return nil
}

// languageSpec maps a language name to its interpreter and file extension.
type languageSpec struct {
	interpreter string
	args        []string
	filename    string
}

// supportedLanguages is the set of languages ProcessSandbox can execute.
var supportedLanguages = map[string]languageSpec{
	"python":     {interpreter: "python3", filename: "code.py"},
	"javascript": {interpreter: "node", filename: "code.js"},
	"bash":       {interpreter: "bash", filename: "code.sh"},
	"sh":         {interpreter: "sh", filename: "code.sh"},
	"ruby":       {interpreter: "ruby", filename: "code.rb"},
	"go": {
		interpreter: "go",
		args:        []string{"run"},
		filename:    "main.go",
	},
}

// resolveLanguage returns the language spec for the given name or an error.
func resolveLanguage(name string) (languageSpec, error) {
	if name == "" {
		return languageSpec{}, core.NewError(opSandboxExecute, core.ErrInvalidInput, "language must not be empty", nil)
	}
	lang, ok := supportedLanguages[name]
	if !ok {
		return languageSpec{}, core.NewError(opSandboxExecute, core.ErrInvalidInput,
			fmt.Sprintf("unsupported language %q", name), nil)
	}
	return lang, nil
}

// limitedWriter wraps a bytes.Buffer and stops writing after max bytes.
type limitedWriter struct {
	buf *bytes.Buffer
	max int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	remaining := w.max - w.buf.Len()
	if remaining <= 0 {
		return len(p), nil // Discard silently — still report full write to avoid cmd error.
	}
	if len(p) > remaining {
		p = p[:remaining]
	}
	return w.buf.Write(p)
}
