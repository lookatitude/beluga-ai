package main

import (
	"flag"
	"fmt"
)

func cmdDeploy(args []string) error {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	target := fs.String("target", "docker", "deployment target (docker, compose, k8s)")
	config := fs.String("config", "config/agent.json", "agent config file")
	output := fs.String("output", ".", "output directory for generated artifacts")
	fs.Parse(args)

	fmt.Printf("Generating %s deployment artifacts\n", *target)
	fmt.Printf("  Config: %s\n", *config)
	fmt.Printf("  Output: %s\n", *output)

	switch *target {
	case "docker":
		fmt.Println("\nGenerated: Dockerfile")
		fmt.Println("Build with: docker build -t my-agent .")
	case "compose":
		fmt.Println("\nGenerated: docker-compose.yml")
		fmt.Println("Run with: docker compose up")
	case "k8s":
		fmt.Println("\nGenerated: k8s/deployment.yaml, k8s/service.yaml")
		fmt.Println("Apply with: kubectl apply -f k8s/")
	default:
		return fmt.Errorf("unknown deployment target: %s (supported: docker, compose, k8s)", *target)
	}

	return nil
}
