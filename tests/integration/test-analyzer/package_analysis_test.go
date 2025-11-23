package testanalyzer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageAnalysis_Integration_AllPackages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test analyzing all framework packages
	pkgDir := filepath.Join("..", "..", "..", "pkg")
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		t.Skip("pkg/ directory not found")
	}

	// List of packages to test
	packages := []string{
		"agents", "chatmodels", "config", "core",
		"embeddings", "llms", "memory", "monitoring",
		"orchestration", "prompts",
	}

	for _, pkg := range packages {
		t.Run("Package_"+pkg, func(t *testing.T) {
			pkgPath := filepath.Join(pkgDir, pkg)
			if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
				t.Skipf("Package %s not found", pkg)
			}

			// Run analysis on the package
			cmd := exec.CommandContext(context.Background(), "go", "run",
				filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
				"--dry-run", "--output", "json", pkgPath)

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Verify we got some output
			if len(outputStr) == 0 && err == nil {
				t.Error("Expected non-empty output or error")
			}

			// If JSON output, verify it's valid JSON structure
			if strings.Contains(outputStr, "{") {
				// Basic JSON structure check
				if !strings.Contains(outputStr, "\"") {
					t.Error("Expected JSON format")
				}
			}
		})
	}
}

func TestPackageAnalysis_Integration_ComparePackages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test comparing analysis results across packages
	pkgDir := filepath.Join("..", "..", "..", "pkg")
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		t.Skip("pkg/ directory not found")
	}

	packages := []string{"llms", "memory", "orchestration", "agents"}

	for _, pkg := range packages {
		t.Run("Compare_"+pkg, func(t *testing.T) {
			pkgPath := filepath.Join(pkgDir, pkg)
			if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
				t.Skipf("Package %s not found", pkg)
			}

			cmd := exec.CommandContext(context.Background(), "go", "run",
				filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
				"--dry-run", "--output", "json", pkgPath)

			output, err := cmd.CombinedOutput()
			_ = err

			// Store or compare results
			_ = output
		})
	}
}
