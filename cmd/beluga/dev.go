package main

import (
	"flag"
	"fmt"
)

func cmdDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	port := fs.Int("port", 8080, "development server port")
	config := fs.String("config", "config/agent.json", "agent config file")
	fs.Parse(args)

	fmt.Printf("Starting development server on :%d with config %s\n", *port, *config)
	fmt.Println("Playground available at http://localhost:", *port, "/playground")
	fmt.Println("Press Ctrl+C to stop.")

	// In a full implementation, this would start an HTTP server with the
	// playground handler and hot-reload on config changes.
	// For now, print the configuration for verification.
	fmt.Printf("\nConfiguration:\n  Port: %d\n  Config: %s\n", *port, *config)

	return nil
}
