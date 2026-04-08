// Command beluga provides CLI tools for managing Beluga AI projects.
// Subcommands: init, dev, test, deploy.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "init":
		err = cmdInit(os.Args[2:])
	case "dev":
		err = cmdDev(os.Args[2:])
	case "test":
		err = cmdTest(os.Args[2:])
	case "deploy":
		err = cmdDeploy(os.Args[2:])
	case "version":
		fmt.Println("beluga v0.1.0")
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: beluga <command> [options]

Commands:
  init     Initialize a new Beluga AI project
  dev      Start development server
  test     Run agent tests
  deploy   Generate deployment artifacts
  version  Print version information
  help     Show this help message

Run 'beluga <command> -h' for command-specific help.`)
}
