// Command beluga provides CLI tools for managing Beluga AI projects.
// Subcommands: init, dev, test, deploy.
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes the CLI with the given args and output sinks and returns an
// exit code. It exists so tests can exercise dispatch and error paths without
// calling os.Exit.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		printUsage(stdout)
		return 1
	}

	var err error
	switch args[0] {
	case "init":
		err = cmdInit(args[1:])
	case "dev":
		err = cmdDev(args[1:])
	case "test":
		err = cmdTest(args[1:])
	case "deploy":
		err = cmdDeploy(args[1:])
	case "version":
		fmt.Fprintln(stdout, "beluga v0.1.0")
	case "help", "-h", "--help":
		printUsage(stdout)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}

	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `Usage: beluga <command> [options]

Commands:
  init     Initialize a new Beluga AI project
  dev      Start development server
  test     Run agent tests
  deploy   Generate deployment artifacts
  version  Print version information
  help     Show this help message

Run 'beluga <command> -h' for command-specific help.`)
}
