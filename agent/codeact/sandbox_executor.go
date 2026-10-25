package codeact

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/tool/sandbox"
)

// Compile-time interface check.
var _ CodeExecutor = (*SandboxExecutor)(nil)

// SandboxExecutor adapts a sandbox.Sandbox to the CodeExecutor interface, so
// CodeActAgent can run code on any registered sandbox provider. sandbox is
// the framework's common code-execution backend; prefer it over
// ProcessExecutor so that isolation, resource limits and network policy are
// handled in one place.
type SandboxExecutor struct {
	sb  sandbox.Sandbox
	cfg sandbox.SandboxConfig
}

// NewSandboxExecutor returns a CodeExecutor that runs each CodeAction on sb.
// base supplies the defaults (network policy, resources, ...) for every run;
// the action's Language and a non-zero Timeout override base's.
func NewSandboxExecutor(sb sandbox.Sandbox, base sandbox.SandboxConfig) *SandboxExecutor {
	return &SandboxExecutor{sb: sb, cfg: base}
}

// Execute runs action on the sandbox and converts the result.
func (e *SandboxExecutor) Execute(ctx context.Context, action CodeAction) (CodeResult, error) {
	cfg := e.cfg
	if action.Language != "" {
		cfg.Language = action.Language
	}
	if action.Timeout > 0 {
		cfg.Timeout = action.Timeout
	}
	res, err := e.sb.Execute(ctx, action.Code, cfg)
	return CodeResult{
		Output:   res.Output,
		Error:    res.Error,
		ExitCode: res.ExitCode,
		Duration: res.Duration,
	}, err
}
