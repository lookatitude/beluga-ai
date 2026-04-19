package main

import (
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
		Short:         "Generate deployment artifacts",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDeploy(target, config, output)
		},
	}
	cmd.Flags().StringVar(&target, "target", "docker", "deployment target (docker, compose, k8s)")
	cmd.Flags().StringVar(&config, "config", "config/agent.json", "agent config file")
	cmd.Flags().StringVar(&output, "output", ".", "output directory for generated artifacts")
	return cmd
}

// runDeploy executes the deploy workflow with pre-parsed flag values.
func runDeploy(target, config, output string) error {
	fmt.Printf("[stub] Would generate %s deployment artifacts\n", target)
	fmt.Printf("  Config: %s\n", config)
	fmt.Printf("  Output: %s\n", output)

	switch target {
	case "docker":
		fmt.Printf("[stub] would write Dockerfile to %s\n", output)
		fmt.Println("Build with: docker build -t my-agent .")
	case "compose":
		fmt.Printf("[stub] would write docker-compose.yml to %s\n", output)
		fmt.Println("Run with: docker compose up")
	case "k8s":
		fmt.Printf("[stub] would write k8s/deployment.yaml, k8s/service.yaml to %s\n", output)
		fmt.Println("Apply with: kubectl apply -f k8s/")
	default:
		return fmt.Errorf("unknown deployment target: %s (supported: docker, compose, k8s)", target)
	}

	fmt.Println("\nnote: artifact generation is not yet implemented; no files were written.")
	return nil
}
