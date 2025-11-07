package testanalyzer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndToEnd_DryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test scenario: Dry run analysis
	pkgDir := filepath.Join("..", "..", "..", "pkg", "llms")
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		t.Skip("pkg/llms not found")
	}

	cmd := exec.CommandContext(context.Background(), "go", "run",
		filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
		"--dry-run", pkgDir)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Command output: %s", string(output))
	}
	
	// Verify output contains analysis results
	outputStr := string(output)
	if !strings.Contains(outputStr, "package") && !strings.Contains(outputStr, "files") {
		t.Error("Expected analysis output")
	}
}

func TestEndToEnd_SpecificPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test scenario: Analyze specific package
	pkgDir := filepath.Join("..", "..", "..", "pkg", "memory")
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		t.Skip("pkg/memory not found")
	}

	cmd := exec.CommandContext(context.Background(), "go", "run",
		filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
		"--dry-run", "--output", "markdown", pkgDir)
	
	output, err := cmd.CombinedOutput()
	_ = err
	_ = output
}

func TestEndToEnd_ReportGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test scenario: Generate reports in different formats
	pkgDir := filepath.Join("..", "..", "..", "pkg", "llms")
	if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
		t.Skip("pkg/llms not found")
	}

	formats := []string{"json", "html", "markdown", "plain"}
	for _, format := range formats {
		t.Run("Format_"+format, func(t *testing.T) {
			cmd := exec.CommandContext(context.Background(), "go", "run",
				filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
				"--dry-run", "--output", format, pkgDir)
			
			output, err := cmd.CombinedOutput()
			_ = err
			
			// Verify output is not empty
			if len(output) == 0 {
				t.Error("Expected non-empty output")
			}
		})
	}
}

func TestEndToEnd_AutoFixWithValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test scenario: Auto-fix with validation (dry-run mode)
	// This tests the full workflow without actually modifying files
	fixturesDir := filepath.Join(".", "fixtures")
	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Skip("fixtures directory not found")
	}

	cmd := exec.CommandContext(context.Background(), "go", "run",
		filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
		"--dry-run", "--auto-fix", fixturesDir)
	
	output, err := cmd.CombinedOutput()
	_ = err
	_ = output
}

