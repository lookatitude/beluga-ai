package fixes

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// GenerateLoopExitFix generates the complete fix for adding loop exit.
// It reads the file, finds the infinite loop, and adds exit conditions.
func GenerateLoopExitFix(ctx context.Context, filePath string, lineStart, lineEnd int) (oldCode, newCode string, err error) {
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

		// Check if it's an infinite loop
		if strings.Contains(oldCode, "for {") || strings.Contains(oldCode, "for true {") {
			// Add exit condition - replace the loop opening with a select-based exit
			newCode = `for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			return
		default:
` + strings.TrimPrefix(oldCode, "for {") + `
		}
	}`
			// If the old code doesn't start with "for {", try simpler replacement
			if !strings.HasPrefix(oldCode, "for {") {
				newCode = strings.Replace(oldCode, "for {", `for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			return
		default:
			// Loop body
		}`, 1)
			}
			return oldCode, newCode, nil
		}
	}

	return "", "", fmt.Errorf("infinite loop not found at line %d", lineStart)
}
