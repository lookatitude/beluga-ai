package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	name := fs.String("name", "", "project name (default: current directory name)")
	dir := fs.String("dir", ".", "project directory")
	fs.Parse(args)

	cleanDir := filepath.Clean(*dir)
	if strings.Contains(cleanDir, "..") {
		return fmt.Errorf("path traversal not allowed: %q", *dir)
	}

	if *name == "" {
		absDir, err := filepath.Abs(cleanDir)
		if err != nil {
			return fmt.Errorf("resolve directory: %w", err)
		}
		*name = filepath.Base(absDir)
	}

	// Create project structure.
	dirs := []string{
		filepath.Join(cleanDir, "agents"),
		filepath.Join(cleanDir, "tools"),
		filepath.Join(cleanDir, "config"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0750); err != nil {
			return fmt.Errorf("create directory %s: %w", d, err)
		}
	}

	// Write config file.
	configPath := filepath.Join(cleanDir, "config", "agent.json")
	configContent := fmt.Sprintf(`{
  "id": "%s-agent",
  "persona": {
    "role": "Assistant",
    "goal": "Help users with their tasks"
  },
  "model": {
    "provider": "openai",
    "model": "gpt-4o"
  }
}
`, *name)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	// Write main.go.
	mainPath := filepath.Join(cleanDir, "main.go")
	mainContent := fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/agent"
)

func main() {
	a := agent.New("%s-agent",
		agent.WithPersona(agent.Persona{Role: "Assistant"}),
	)

	result, err := a.Invoke(context.Background(), "Hello!")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
`, *name)

	if err := os.WriteFile(mainPath, []byte(mainContent), 0600); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	fmt.Printf("Initialized Beluga AI project %q in %s\n", *name, cleanDir)
	fmt.Println("  agents/       — agent definitions")
	fmt.Println("  tools/        — custom tools")
	fmt.Println("  config/       — configuration files")
	fmt.Println("  main.go       — entry point")

	return nil
}
