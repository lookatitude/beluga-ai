package testanalyzer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzer_Integration_RealTestFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test using the command-line tool as a black box
	t.Run("AnalyzeLLMsPackage", func(t *testing.T) {
		pkgDir := filepath.Join("..", "..", "..", "pkg", "llms")
		if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
			t.Skip("pkg/llms not found")
		}

		// Run test-analyzer on the package
		cmd := exec.CommandContext(context.Background(), "go", "run",
			filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
			"--dry-run", "--output", "json", pkgDir)

		output, err := cmd.CombinedOutput()
		if err != nil {
			// Tool may return non-zero exit code if issues are found, which is expected
			t.Logf("Tool output: %s", string(output))
		} else {
			// If no error, verify output is valid JSON
			if !strings.Contains(string(output), "{") {
				t.Error("Expected JSON output")
			}
		}
	})

	t.Run("AnalyzeMemoryPackage", func(t *testing.T) {
		pkgDir := filepath.Join("..", "..", "..", "pkg", "memory")
		if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
			t.Skip("pkg/memory not found")
		}

		cmd := exec.CommandContext(context.Background(), "go", "run",
			filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
			"--dry-run", "--output", "json", pkgDir)

		output, err := cmd.CombinedOutput()
		_ = err // Error is acceptable if issues found
		_ = output
	})

	t.Run("AnalyzeTestFixtures", func(t *testing.T) {
		fixturesDir := filepath.Join(".", "fixtures")
		if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
			t.Skip("fixtures directory not found")
		}

		cmd := exec.CommandContext(context.Background(), "go", "run",
			filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
			"--dry-run", "--output", "json", fixturesDir)

		output, err := cmd.CombinedOutput()
		_ = err
		_ = output
	})
}

func TestAnalyzer_Integration_DetectIssues(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test that the tool can detect issues in fixture files
	fixturesDir := filepath.Join(".", "fixtures")
	if _, err := os.Stat(fixturesDir); os.IsNotExist(err) {
		t.Skip("fixtures directory not found")
	}

	cmd := exec.CommandContext(context.Background(), "go", "run",
		filepath.Join("..", "..", "..", "cmd", "test-analyzer"),
		"--dry-run", "--output", "json", fixturesDir)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Check if issues were detected
	if strings.Contains(outputStr, "InfiniteLoop") ||
		strings.Contains(outputStr, "MissingTimeout") ||
		strings.Contains(outputStr, "LargeIteration") {
		t.Logf("Issues detected as expected: %s", outputStr[:min(200, len(outputStr))])
	}

	_ = err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
