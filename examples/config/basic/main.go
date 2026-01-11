package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
	fmt.Println("🔄 Beluga AI Config Package Usage Example")
	fmt.Println("==========================================")

	ctx := context.Background()

	// Example 1: Create Config Manager
	fmt.Println("\n📋 Example 1: Creating Config Manager")
	manager, err := config.NewManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create config manager: %v", err)
	}
	fmt.Println("✅ Config manager created successfully")

	// Example 2: Load from Environment Variables
	fmt.Println("\n📋 Example 2: Loading from Environment Variables")
	os.Setenv("APP_NAME", "beluga-example")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("APP_DEBUG", "true")

	envConfig, err := manager.LoadFromEnv()
	if err != nil {
		log.Printf("⚠️  Failed to load from env: %v", err)
	} else {
		fmt.Println("✅ Configuration loaded from environment")
		fmt.Printf("   App Name: %s\n", envConfig.GetString("app_name"))
		fmt.Printf("   Port: %s\n", envConfig.GetString("app_port"))
	}

	// Example 3: Load from YAML File
	fmt.Println("\n📋 Example 3: Loading from YAML File")
	// Create a sample config file
	yamlContent := `
app:
  name: beluga-example
  port: 8080
  debug: true
database:
  host: localhost
  port: 5432
`
	tmpFile := "/tmp/beluga-config.yaml"
	err = os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	if err != nil {
		log.Printf("⚠️  Failed to create temp file: %v", err)
	} else {
		defer os.Remove(tmpFile)
		yamlConfig, err := manager.LoadFromFile(tmpFile)
		if err != nil {
			log.Printf("⚠️  Failed to load from YAML: %v", err)
		} else {
			fmt.Println("✅ Configuration loaded from YAML file")
			fmt.Printf("   App Name: %s\n", yamlConfig.GetString("app.name"))
			fmt.Printf("   Database Host: %s\n", yamlConfig.GetString("database.host"))
		}
	}

	// Example 4: Get Configuration Value
	fmt.Println("\n📋 Example 4: Getting Configuration Values")
	if envConfig != nil {
		appName := envConfig.GetString("app_name")
		port := envConfig.GetString("app_port")
		debug := envConfig.GetBool("app_debug", false)
		fmt.Printf("✅ Retrieved values:\n")
		fmt.Printf("   App Name: %s\n", appName)
		fmt.Printf("   Port: %s\n", port)
		fmt.Printf("   Debug: %t\n", debug)
	}

	// Example 5: Set Configuration Value
	fmt.Println("\n📋 Example 5: Setting Configuration Values")
	if envConfig != nil {
		envConfig.Set("app_version", "1.0.0")
		version := envConfig.GetString("app_version")
		fmt.Printf("✅ Set and retrieved version: %s\n", version)
	}

	// Example 6: Validate Configuration
	fmt.Println("\n📋 Example 6: Validating Configuration")
	if envConfig != nil {
		err := manager.Validate(envConfig)
		if err != nil {
			log.Printf("⚠️  Validation failed: %v", err)
		} else {
			fmt.Println("✅ Configuration validation passed")
		}
	}

	fmt.Println("\n✨ All examples completed successfully!")
	fmt.Println("\nFor more examples, see:")
	fmt.Println("  - examples/llm-usage/config_loader.go - Advanced config loading")
	fmt.Println("  - Package documentation: pkg/config/README.md")
}
