package codeact

import (
	"context"
	"time"
)

// CodeAction describes a code block to be executed.
type CodeAction struct {
	// Language is the programming language of the code (e.g., "python", "go", "javascript").
	Language string
	// Code is the source code to execute.
	Code string
	// Timeout is the maximum duration allowed for execution. Zero means use default.
	Timeout time.Duration
}

// CodeResult holds the outcome of executing a CodeAction.
type CodeResult struct {
	// Output is the stdout content from execution.
	Output string
	// Error is the stderr content or error message from execution, if any.
	Error string
	// ExitCode is the process exit code. Zero indicates success.
	ExitCode int
	// Duration is how long the execution took.
	Duration time.Duration
}

// Success reports whether the code executed successfully, defined by the
// POSIX convention of a zero exit code. Stderr output is ignored because
// well-behaved programs (including Python deprecation warnings) routinely
// write to stderr while exiting with code 0.
func (r CodeResult) Success() bool {
	return r.ExitCode == 0
}

// CodeActHooks provides optional callback functions for code execution lifecycle.
// All fields are optional; nil hooks are skipped.
type CodeActHooks struct {
	// BeforeExec is called before code is executed. Returning an error cancels execution.
	BeforeExec func(ctx context.Context, action CodeAction) error
	// AfterExec is called after code execution completes.
	AfterExec func(ctx context.Context, action CodeAction, result CodeResult) error
	// OnError is called when code execution fails. The returned error replaces the original.
	OnError func(ctx context.Context, action CodeAction, err error) error
}

// ComposeCodeActHooks merges multiple CodeActHooks into a single value.
// Callbacks are called in the order provided. The first error short-circuits.
func ComposeCodeActHooks(hooks ...CodeActHooks) CodeActHooks {
	return CodeActHooks{
		BeforeExec: composeBeforeExec(hooks),
		AfterExec:  composeAfterExec(hooks),
		OnError:    composeOnError(hooks),
	}
}

func composeBeforeExec(hooks []CodeActHooks) func(context.Context, CodeAction) error {
	return func(ctx context.Context, action CodeAction) error {
		for _, h := range hooks {
			if h.BeforeExec == nil {
				continue
			}
			if err := h.BeforeExec(ctx, action); err != nil {
				return err
			}
		}
		return nil
	}
}

func composeAfterExec(hooks []CodeActHooks) func(context.Context, CodeAction, CodeResult) error {
	return func(ctx context.Context, action CodeAction, result CodeResult) error {
		for _, h := range hooks {
			if h.AfterExec == nil {
				continue
			}
			if err := h.AfterExec(ctx, action, result); err != nil {
				return err
			}
		}
		return nil
	}
}

func composeOnError(hooks []CodeActHooks) func(context.Context, CodeAction, error) error {
	return func(ctx context.Context, action CodeAction, err error) error {
		for _, h := range hooks {
			if h.OnError == nil {
				continue
			}
			err = h.OnError(ctx, action, err)
			if err == nil {
				return nil
			}
		}
		return err
	}
}
