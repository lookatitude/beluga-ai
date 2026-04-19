package main

import (
	"flag"
	"fmt"

	"github.com/spf13/cobra"
)

// newDeployCmd is a T2 adapter that delegates to cmdDeploy. T3 replaces this
// with a native cobra RunE that uses pflag directly.
func newDeployCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "deploy [flags]",
		Short:              "Generate deployment artifacts",
		SilenceUsage:       true,
		SilenceErrors:      true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmdDeploy(args)
		},
	}
}

func cmdDeploy(args []string) error {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	target := fs.String("target", "docker", "deployment target (docker, compose, k8s)")
	config := fs.String("config", "config/agent.json", "agent config file")
	output := fs.String("output", ".", "output directory for generated artifacts")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	fmt.Printf("[stub] Would generate %s deployment artifacts\n", *target)
	fmt.Printf("  Config: %s\n", *config)
	fmt.Printf("  Output: %s\n", *output)

	switch *target {
	case "docker":
		fmt.Printf("[stub] would write Dockerfile to %s\n", *output)
		fmt.Println("Build with: docker build -t my-agent .")
	case "compose":
		fmt.Printf("[stub] would write docker-compose.yml to %s\n", *output)
		fmt.Println("Run with: docker compose up")
	case "k8s":
		fmt.Printf("[stub] would write k8s/deployment.yaml, k8s/service.yaml to %s\n", *output)
		fmt.Println("Apply with: kubectl apply -f k8s/")
	default:
		return fmt.Errorf("unknown deployment target: %s (supported: docker, compose, k8s)", *target)
	}

	fmt.Println("\nnote: artifact generation is not yet implemented; no files were written.")
	return nil
}
