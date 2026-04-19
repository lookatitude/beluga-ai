// Command beluga provides CLI tools for managing Beluga AI projects.
// Subcommands: init, dev, test, deploy.
package main

import (
	"os"

	// Side-effect import: triggers init() registration for the curated set of
	// providers shipped with the beluga CLI. See cmd/beluga/providers.
	_ "github.com/lookatitude/beluga-ai/v2/cmd/beluga/providers"
)

func main() { os.Exit(Execute(os.Stdout, os.Stderr)) }
