package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

// validPkgPattern restricts -pkg values to conservative Go package path
// patterns to prevent smuggling additional `go test` flags via the argument.
var validPkgPattern = regexp.MustCompile(`^[A-Za-z0-9_./\-]+(\.\.\.)?$`)

func cmdTest(args []string) error {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	verbose := fs.Bool("v", false, "verbose test output")
	race := fs.Bool("race", false, "enable race detector")
	pkg := fs.String("pkg", "./...", "packages to test")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	if !validPkgPattern.MatchString(*pkg) {
		return fmt.Errorf("invalid package pattern: %q", *pkg)
	}

	goArgs := []string{"test"}
	if *verbose {
		goArgs = append(goArgs, "-v")
	}
	if *race {
		goArgs = append(goArgs, "-race")
	}
	goArgs = append(goArgs, *pkg)

	fmt.Printf("Running: go %v\n", goArgs)

	//nolint:gosec // G204: package pattern is validated against validPkgPattern above.
	cmd := exec.Command("go", goArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
