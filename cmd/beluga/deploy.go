package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// newDeployCmd returns the cobra subcommand for `beluga deploy`. Flag names
// are preserved from the pre-cobra CLI: --target, --config, --output.
//
// The local --output flag shadows the root-level persistent --output/-o flag;
// the local flag has no short alias so the root's -o is never ambiguous.
// This preserves the pre-cobra CLI behaviour where `deploy -output foo`
// meant the artifact output directory (AC5: no flag regression).
func newDeployCmd() *cobra.Command {
	var (
		target string
		config string
		output string
	)
	cmd := &cobra.Command{
		Use:           "deploy [flags]",
		Short:         "Generate deployment artifacts (not implemented yet; exits non-zero)",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDeploy(target)
		},
	}
	cmd.Flags().StringVar(&target, "target", "docker", "deployment target (docker, compose, k8s)")
	cmd.Flags().StringVar(&config, "config", "config/agent.json", "agent config file")
	cmd.Flags().StringVar(&output, "output", ".", "output directory for generated artifacts")
	return cmd
}

// errDeployNotImplemented is returned for every valid target: artifact
// generation is not implemented yet, so the command must not report success.
var errDeployNotImplemented = errors.New("deploy: artifact generation is not implemented yet; no files were written " +
	"(use the scaffolded Dockerfile, or deploy.GenerateDockerfile / deploy.GenerateCompose from Go)")

// runDeploy validates the flags and reports that artifact generation is not
// implemented. It exits non-zero so scripts and CI do not mistake the
// placeholder for a successful deploy.
func runDeploy(target string) error {
	switch target {
	case "docker", "compose", "k8s":
	default:
		return fmt.Errorf("unknown deployment target: %s (supported: docker, compose, k8s)", target)
	}
	return fmt.Errorf("%s target: %w", target, errDeployNotImplemented)
}
