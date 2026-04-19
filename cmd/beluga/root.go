package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// newRootCmd builds the beluga root command tree. Constructed per-call so
// tests can capture output and parse fresh args without state leakage.
func newRootCmd() *cobra.Command {
	var (
		logLevel string // persistent flag; S1 recognised but not wired to slog
		output   string // persistent flag; consumed by `providers` in S1
	)

	root := &cobra.Command{
		Use:           "beluga",
		Short:         "Beluga AI CLI — scaffold, run, and operate agents",
		Long:          "beluga is the official command-line tool for the Beluga AI framework.",
		SilenceUsage:  true, // RunE errors print once via the top-level handler
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "info",
		"log level (debug, info, warn, error) — written to stderr")
	root.PersistentFlags().StringVarP(&output, "output", "o", "",
		`output format for machine-readable commands (e.g. "json")`)

	root.AddCommand(
		newVersionCmd(),
		newInitCmd(),
		newDevCmd(),
		newTestCmd(),
		newDeployCmd(),
	)

	return root
}

// Execute is the entry point called by main(). It returns an exit code so
// tests can exercise error paths without os.Exit.
func Execute(stdout, stderr io.Writer) int {
	cmd := newRootCmd()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		// cobra's SilenceErrors means we own the stderr write.
		_, _ = fmt.Fprintf(stderr, "error: %s\n", err.Error())
		return 1
	}
	return 0
}
