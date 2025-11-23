package fixes

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// GenerateTimeoutFix generates the complete fix for adding a timeout.
// It reads the file and adds context.WithTimeout at the start of the function body.
func GenerateTimeoutFix(ctx context.Context, filePath, functionName string, lineStart, lineEnd int, timeoutDuration string) (oldCode, newCode string, err error) {
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

		// Generate the timeout code
		timeoutCode := fmt.Sprintf("\tctx, cancel := context.WithTimeout(context.Background(), %s)\n\tdefer cancel()", timeoutDuration)

		// Generate new code: add timeout code before the existing code
		newCode = timeoutCode + "\n" + oldCode
		return oldCode, newCode, nil
	}

	return "", "", fmt.Errorf("could not extract code from line %d", lineStart)
}
