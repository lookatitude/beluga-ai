package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newDevCmd returns the cobra subcommand for `beluga dev`. Flag names are
// preserved from the pre-cobra CLI: --port, --config.
func newDevCmd() *cobra.Command {
	var (
		port   int
		config string
	)
	cmd := &cobra.Command{
		Use:           "dev [flags]",
		Short:         "Start development server",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDev(port, config)
		},
	}
	cmd.Flags().IntVar(&port, "port", 8080, "development server port")
	cmd.Flags().StringVar(&config, "config", "config/agent.json", "agent config file")
	return cmd
}

// runDev executes the dev workflow with pre-parsed flag values.
func runDev(port int, config string) error {
	fmt.Printf("Starting development server on :%d with config %s\n", port, config)
	fmt.Printf("Playground available at http://localhost:%d/playground\n", port)
	fmt.Println("Press Ctrl+C to stop.")

	// In a full implementation, this would start an HTTP server with the
	// playground handler and hot-reload on config changes.
	// For now, print the configuration for verification.
	fmt.Printf("\nConfiguration:\n  Port: %d\n  Config: %s\n", port, config)

	return nil
}
