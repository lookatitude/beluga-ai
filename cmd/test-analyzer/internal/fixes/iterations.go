package fixes

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// GenerateIterationFix generates the complete fix for reducing iterations.
// It reads the file, finds the loop, and replaces the iteration count.
func GenerateIterationFix(ctx context.Context, filePath string, oldCount, newCount int, lineStart, lineEnd int) (oldCode, newCode string, err error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("reading file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Extract the actual code at the specified line range
	if lineStart > 0 && lineStart <= len(lines) {
		// Get the line(s) to replace
		endLine := lineEnd
		if endLine < lineStart {
			endLine = lineStart
		}
		if endLine > len(lines) {
			endLine = len(lines)
		}

		// Extract old code
		oldCodeLines := lines[lineStart-1 : endLine]
		oldCode = strings.Join(oldCodeLines, "\n")

		// Replace iteration count in the code
		newCode = oldCode
		// Try multiple replacement strategies
		newCode = strings.Replace(newCode, fmt.Sprintf("%d", oldCount), fmt.Sprintf("%d", newCount), -1)
		// Also try with spaces around
		newCode = strings.Replace(newCode, fmt.Sprintf(" %d ", oldCount), fmt.Sprintf(" %d ", newCount), -1)
		newCode = strings.Replace(newCode, fmt.Sprintf("< %d", oldCount), fmt.Sprintf("< %d", newCount), -1)
		newCode = strings.Replace(newCode, fmt.Sprintf("<= %d", oldCount), fmt.Sprintf("<= %d", newCount), -1)

		return oldCode, newCode, nil
	}

	return "", "", fmt.Errorf("could not extract code from line %d", lineStart)
}
