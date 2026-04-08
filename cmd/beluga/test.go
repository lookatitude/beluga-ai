package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

func cmdTest(args []string) error {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	verbose := fs.Bool("v", false, "verbose test output")
	race := fs.Bool("race", false, "enable race detector")
	pkg := fs.String("pkg", "./...", "packages to test")
	fs.Parse(args)

	goArgs := []string{"test"}
	if *verbose {
		goArgs = append(goArgs, "-v")
	}
	if *race {
		goArgs = append(goArgs, "-race")
	}
	goArgs = append(goArgs, *pkg)

	fmt.Printf("Running: go %v\n", goArgs)

	cmd := exec.Command("go", goArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
