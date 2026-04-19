package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"

	"github.com/spf13/cobra"
)

// newTestCmd is a T2 adapter that delegates to cmdTest. T3 replaces this with
// a native cobra RunE that uses pflag directly.
func newTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "test [flags]",
		Short:              "Run agent tests",
		SilenceUsage:       true,
		SilenceErrors:      true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdTest(args)
		},
	}
}

// validPkgPattern restricts -pkg values to conservative Go package path
// patterns to prevent smuggling additional `go test` flags via the argument.
var validPkgPattern = regexp.MustCompile(`^[A-Za-z0-9_./\-]+(\.\.\.)?$`)

// execCommand is indirected so tests can stub command construction. The
// production implementation resolves the binary via an absolute path (so
// execution does not rely on PATH lookup at Run time) and wires stdout/stderr
// through to the caller.
var execCommand = func(stdout, stderr io.Writer, name string, args ...string) *exec.Cmd {
	// #nosec G204 -- name is always an absolute path resolved via exec.LookPath("go")
	// in cmdTest, and args are validated (verbose/race are bool flags, pkg is
	// checked against validPkgPattern). No shell is involved.
	c := exec.Command(name, args...) //nolint:gosec // G204: see nosec justification above
	c.Stdout = stdout
	c.Stderr = stderr
	return c
}

// lookPath is indirected so tests can stub binary resolution.
var lookPath = exec.LookPath

func cmdTest(args []string) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	verbose := fs.Bool("v", false, "verbose test output")
	race := fs.Bool("race", false, "enable race detector")
	pkg := fs.String("pkg", "./...", "packages to test")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if !validPkgPattern.MatchString(*pkg) {
		return fmt.Errorf("invalid package pattern: %q", *pkg)
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
	if *verbose {
		goArgs = append(goArgs, "-v")
	}
	if *race {
		goArgs = append(goArgs, "-race")
	}
	goArgs = append(goArgs, *pkg)

	fmt.Printf("Running: %s %v\n", goBin, goArgs)

	cmd := execCommand(os.Stdout, os.Stderr, goBin, goArgs...)
	return cmd.Run()
}
