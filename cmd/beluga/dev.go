package main

import (
	"flag"
	"fmt"

	"github.com/spf13/cobra"
)

// newDevCmd is a T2 adapter that delegates to cmdDev. T3 replaces this with a
// native cobra RunE that uses pflag directly.
func newDevCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "dev [flags]",
		Short:              "Start development server",
		SilenceUsage:       true,
		SilenceErrors:      true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdDev(args)
		},
	}
}

func cmdDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	port := fs.Int("port", 8080, "development server port")
	config := fs.String("config", "config/agent.json", "agent config file")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	fmt.Printf("Starting development server on :%d with config %s\n", *port, *config)
	fmt.Printf("Playground available at http://localhost:%d/playground\n", *port)
	fmt.Println("Press Ctrl+C to stop.")

	// In a full implementation, this would start an HTTP server with the
	// playground handler and hot-reload on config changes.
	// For now, print the configuration for verification.
	fmt.Printf("\nConfiguration:\n  Port: %d\n  Config: %s\n", *port, *config)

	return nil
}
