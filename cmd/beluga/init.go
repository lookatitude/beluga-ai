package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// newInitCmd returns the cobra subcommand for `beluga init`. Flag names are
// preserved from the pre-cobra CLI: --name, --dir.
func newInitCmd() *cobra.Command {
	var (
		name string
		dir  string
	)
	cmd := &cobra.Command{
		Use:           "init [flags]",
		Short:         "Initialize a new Beluga AI project",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(name, dir)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "project name (default: current directory name)")
	cmd.Flags().StringVar(&dir, "dir", ".", "project directory")
	return cmd
}

// runInit executes the init workflow with pre-parsed flag values. Extracted
// from the RunE body so tests can exercise the behaviour directly without a
// cobra command tree.
func runInit(name, dir string) error {
	// Resolve to an absolute, cleaned path and require it to be rooted under
	// the current working directory. This defeats both relative (`../..`) and
	// absolute (`/tmp/../etc/passwd`) traversal attempts.
	cleanDir, err := filepath.Abs(filepath.Clean(dir))
	if err != nil {
		return fmt.Errorf("resolve directory: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	base, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	rel, err := filepath.Rel(base, cleanDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path traversal not allowed: %q", dir)
	}

	if name == "" {
		name = filepath.Base(cleanDir)
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
`, name)

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

	"github.com/lookatitude/beluga-ai/v2/agent"
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
`, name)

	if err := os.WriteFile(mainPath, []byte(mainContent), 0600); err != nil {
		return fmt.Errorf("write main.go: %w", err)
	}

	fmt.Printf("Initialized Beluga AI project %q in %s\n", name, cleanDir)
	fmt.Println("  agents/       - agent definitions")
	fmt.Println("  tools/        - custom tools")
	fmt.Println("  config/       - configuration files")
	fmt.Println("  main.go       - entry point")

	return nil
}
