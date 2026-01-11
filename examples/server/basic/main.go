package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/server"
)

func main() {
	fmt.Println("🌐 Beluga AI - Server Package Example")
	fmt.Println("=====================================")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n⚠️  Shutdown signal received...")
		cancel()
	}()

	// Step 1: Create REST server configuration
	fmt.Println("\n📋 Step 1: Creating REST server configuration...")
	restConfig := server.DefaultRESTConfig()
	restConfig.Port = 8080
	restConfig.Host = "localhost"
	fmt.Println("✅ REST configuration created")

	// Step 2: Create REST server
	fmt.Println("\n📋 Step 2: Creating REST server...")
	restServer, err := server.NewRESTServer(
		server.WithRESTConfig(restConfig),
	)
	if err != nil {
		log.Fatalf("Failed to create REST server: %v", err)
	}
	fmt.Println("✅ REST server created")

	// Step 3: Create MCP server configuration
	fmt.Println("\n📋 Step 3: Creating MCP server configuration...")
	mcpConfig := server.DefaultMCPConfig()
	mcpConfig.Port = 8081
	mcpConfig.Host = "localhost"
	fmt.Println("✅ MCP configuration created")

	// Step 4: Create MCP server
	fmt.Println("\n📋 Step 4: Creating MCP server...")
	mcpServer, err := server.NewMCPServer(
		server.WithMCPConfig(mcpConfig),
	)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}
	fmt.Println("✅ MCP server created")

	// Note: In a real application, you would:
	// - Register handlers for REST server
	// - Register tools and resources for MCP server
	// - Start the servers
	// - Handle requests

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Register HTTP handlers for REST endpoints")
	fmt.Println("- Register tools and resources for MCP server")
	fmt.Println("- Start servers and handle requests")
	fmt.Println("- Configure middleware and authentication")
	fmt.Println("\nNote: This example demonstrates server creation only.")
	fmt.Println("      Start servers in a separate goroutine for production use.")
}
