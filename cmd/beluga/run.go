package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/devloop"
	"github.com/spf13/cobra"
)

// devloopRun is indirected so unit tests can stub the supervisor without
// actually invoking the Go toolchain. Production code uses devloop.Run.
var devloopRun = devloop.Run

// devloopExitCode is indirected for symmetry with devloopRun.
var devloopExitCode = devloop.ExitCode

// newRunCmd returns the cobra subcommand for `beluga run`. It builds the
// scaffolded project at --project-root (default cwd) and execs the
// resulting binary once. Anything after `--` is forwarded to the child
// binary as argv.
func newRunCmd() *cobra.Command {
	var projectRoot string
	cmd := &cobra.Command{
		Use:   "run [flags] [-- args...]",
		Short: "Build and run the scaffolded project once",
		Long: "Builds the Go module at --project-root and execs the resulting " +
			"binary. The child inherits the current environment plus any KEY=value " +
			"entries in <root>/.env. Exits with the child's exit code.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			childArgs := argsAfterDash(cmd, args)
			return runScaffoldedProject(cmd.Context(), projectRoot, childArgs, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringVar(&projectRoot, "project-root", ".",
		"scaffolded project root (directory with go.mod + .beluga/project.yaml)")
	return cmd
}

// argsAfterDash returns argv entries after the `--` separator, or nil
// when cobra did not observe one. Cobra's ArgsLenAtDash returns -1 if
// no dash was present.
func argsAfterDash(cmd *cobra.Command, args []string) []string {
	idx := cmd.ArgsLenAtDash()
	if idx < 0 || idx > len(args) {
		return nil
	}
	return args[idx:]
}

func runScaffoldedProject(ctx context.Context, projectRoot string, childArgs []string, stdout, stderr io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}
	root, err := filepath.Abs(projectRoot)
	if err != nil {
		return fmt.Errorf("resolve project-root: %w", err)
	}

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err = devloopRun(ctx, devloop.Config{
		ProjectRoot: root,
		Watch:       false,
		Stdout:      stdout,
		Stderr:      stderr,
		ChildArgs:   childArgs,
	})
	if code := devloopExitCode(err); code != 0 {
		return &runExitError{code: code, err: err}
	}
	return nil
}

// runExitError carries the child's exit code up to the cobra entry
// point. The top-level Execute() uses it to mirror the child's exit
// code rather than always returning 1.
type runExitError struct {
	code int
	err  error
}

func (e *runExitError) Error() string {
	if e.err == nil {
		return fmt.Sprintf("child exited with code %d", e.code)
	}
	return e.err.Error()
}

func (e *runExitError) ExitCode() int { return e.code }
