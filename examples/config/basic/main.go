package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/config"
)

func main() {
	fmt.Println("🔄 Beluga AI Config Package Usage Example")
	fmt.Println("==========================================")

	// Example 1: Create Config Loader
	fmt.Println("\n📋 Example 1: Creating Config Loader")
	options := config.DefaultLoaderOptions()
	loader, err := config.NewLoader(options)
	if err != nil {
		log.Fatalf("Failed to create config loader: %v", err)
	}
	fmt.Println("✅ Config loader created successfully")

	// Example 2: Load from Environment Variables
	fmt.Println("\n📋 Example 2: Loading from Environment Variables")
	os.Setenv("BELUGA_APP_NAME", "beluga-example")
	os.Setenv("BELUGA_APP_PORT", "8080")
	os.Setenv("BELUGA_APP_DEBUG", "true")

	_, err = config.LoadFromEnv("BELUGA")
	if err != nil {
		log.Printf("⚠️  Failed to load from env: %v", err)
	} else {
		fmt.Println("✅ Configuration loaded from environment")
		fmt.Printf("   Config loaded successfully\n")
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
		yamlConfig, err := config.LoadFromFile(tmpFile)
		if err != nil {
			log.Printf("⚠️  Failed to load from YAML: %v", err)
		} else {
			fmt.Println("✅ Configuration loaded from YAML file")
			fmt.Printf("   Config loaded successfully\n")
		}
		_ = yamlConfig
	}

	// Example 4: Load Default Config
	fmt.Println("\n📋 Example 4: Loading Default Config")
	cfg, err := loader.LoadConfig()
	if err != nil {
		log.Printf("⚠️  Failed to load config: %v", err)
	} else {
		fmt.Println("✅ Configuration loaded successfully")
		_ = cfg
	}

	// Example 5: Validate Configuration
	fmt.Println("\n📋 Example 5: Validating Configuration")
	if cfg != nil {
		err := config.ValidateConfig(cfg)
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
