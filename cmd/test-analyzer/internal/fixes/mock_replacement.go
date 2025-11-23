package fixes

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// GenerateMockReplacementFix generates the complete fix for replacing with mock.
// It reads the file, finds constructor calls, and replaces them with mock constructors.
func GenerateMockReplacementFix(ctx context.Context, filePath, componentName, mockName string, lineStart, lineEnd int) (oldCode, newCode string, err error) {
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

		// Replace constructor name with mock constructor
		newCode = oldCode
		// Try various patterns
		if strings.Contains(oldCode, "New"+componentName) {
			newCode = strings.Replace(newCode, "New"+componentName, "NewAdvancedMock"+componentName, 1)
		} else if strings.Contains(oldCode, "new"+strings.ToLower(componentName)) {
			newCode = strings.Replace(newCode, "new"+strings.ToLower(componentName), "NewAdvancedMock"+componentName, 1)
		} else if strings.Contains(oldCode, componentName) {
			// Try to find and replace any constructor pattern
			// This is a simplified version - would need more sophisticated pattern matching
			newCode = strings.Replace(newCode, componentName+"(", "AdvancedMock"+componentName+"(", 1)
		}

		return oldCode, newCode, nil
	}

	return "", "", fmt.Errorf("could not extract code from line %d", lineStart)
}
