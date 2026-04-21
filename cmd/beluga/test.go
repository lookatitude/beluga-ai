package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"

	"github.com/spf13/cobra"
)

// validPkgPattern restricts -pkg values to conservative Go package path
// patterns to prevent smuggling additional `go test` flags via the argument.
var validPkgPattern = regexp.MustCompile(`^[A-Za-z0-9_./\-]+(\.\.\.)?$`)

// execCommand is indirected so tests can stub command construction. The
// production implementation resolves the binary via an absolute path (so
// execution does not rely on PATH lookup at Run time) and wires stdout/stderr
// through to the caller.
var execCommand = func(stdout, stderr io.Writer, name string, args ...string) *exec.Cmd {
	// #nosec G204 -- name is always an absolute path resolved via exec.LookPath("go")
	// in runTest, and args are validated (verbose/race are bool flags, pkg is
	// checked against validPkgPattern). No shell is involved.
	c := exec.Command(name, args...) //nolint:gosec // G204: see nosec justification above
	c.Stdout = stdout
	c.Stderr = stderr
	return c
}

// lookPath is indirected so tests can stub binary resolution.
var lookPath = exec.LookPath

// newTestCmd returns the cobra subcommand for `beluga test`. Flag names are
// preserved from the pre-cobra CLI: --verbose (short -v), --race, --pkg.
func newTestCmd() *cobra.Command {
	var (
		verbose bool
		race    bool
		pkg     string
	)
	cmd := &cobra.Command{
		Use:           "test [flags]",
		Short:         "Run agent tests",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTest(cmd.OutOrStdout(), cmd.ErrOrStderr(), verbose, race, pkg)
		},
	}
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose test output")
	cmd.Flags().BoolVar(&race, "race", false, "enable race detector")
	cmd.Flags().StringVar(&pkg, "pkg", "./...", "packages to test")
	return cmd
}

// canonicalTestEnv is the fixed set of env vars `beluga test` injects
// into the `go test` child. They opt the scaffolded project into
// mock-provider routing and deterministic golden regen — the
// equivalent of manually setting BELUGA_ENV=test before invoking
// `go test`, but without relying on the user to remember it.
var canonicalTestEnv = []string{
	"BELUGA_ENV=test",
	"BELUGA_LLM_PROVIDER=mock",
	"OTEL_SDK_DISABLED=true",
}

// runTest executes the test workflow with pre-parsed flag values. stdout
// and stderr are the writers the child `go test` process inherits, and
// are also where the "Running: ..." banner is emitted — threaded from
// the cobra command so tests can capture output instead of bypassing
// cobra's writer plumbing via os.Stdout/os.Stderr directly.
func runTest(stdout, stderr io.Writer, verbose, race bool, pkg string) error {
	if !validPkgPattern.MatchString(pkg) {
		return fmt.Errorf("invalid package pattern: %q", pkg)
	}

	// Resolve `go` to an absolute path so execution does not depend on the
	// current PATH. This addresses SonarCloud go:S4036 ("Make sure the PATH
	// variable only contains fixed, unwriteable directories"): by locating
	// the binary once and passing the absolute path, a mutated PATH cannot
	// redirect us to an attacker-controlled `go` at exec time.
	goBin, err := lookPath("go")
	if err != nil {
		return fmt.Errorf("locate go toolchain: %w", err)
	}

	goArgs := []string{"test"}
	if verbose {
		goArgs = append(goArgs, "-v")
	}
	if race {
		goArgs = append(goArgs, "-race")
	}
	goArgs = append(goArgs, pkg)

	fmt.Fprintf(stdout, "Running: %s %v\n", goBin, goArgs)

	cmd := execCommand(stdout, stderr, goBin, goArgs...)
	cmd.Env = append(os.Environ(), canonicalTestEnv...)
	return cmd.Run()
}
