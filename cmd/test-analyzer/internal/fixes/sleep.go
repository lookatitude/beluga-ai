package fixes

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// GenerateSleepFix generates the complete fix for optimizing sleep.
// It reads the file, finds time.Sleep calls, and reduces the duration.
func GenerateSleepFix(ctx context.Context, filePath string, oldDuration, newDuration time.Duration, lineStart, lineEnd int) (oldCode, newCode string, err error) {
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
		oldCodeLines := lines[lineStart-1:endLine]
		oldCode = strings.Join(oldCodeLines, "\n")

		// Replace sleep duration
		newCode = oldCode
		if strings.Contains(oldCode, "time.Sleep") {
			// Try multiple replacement strategies
			// Replace milliseconds
			if oldDuration.Milliseconds() > 0 {
				oldMs := int(oldDuration.Milliseconds())
				newMs := int(newDuration.Milliseconds())
				newCode = strings.Replace(newCode, fmt.Sprintf("%d*time.Millisecond", oldMs), fmt.Sprintf("%d*time.Millisecond", newMs), -1)
				newCode = strings.Replace(newCode, fmt.Sprintf("%d * time.Millisecond", oldMs), fmt.Sprintf("%d * time.Millisecond", newMs), -1)
			}
			// Replace seconds
			if oldDuration.Seconds() > 0 {
				oldSec := int(oldDuration.Seconds())
				newMs := int(newDuration.Milliseconds())
				newCode = strings.Replace(newCode, fmt.Sprintf("%d*time.Second", oldSec), fmt.Sprintf("%d*time.Millisecond", newMs), -1)
				newCode = strings.Replace(newCode, fmt.Sprintf("%d * time.Second", oldSec), fmt.Sprintf("%d * time.Millisecond", newMs), -1)
			}
			// Replace duration literals
			oldDurStr := formatDuration(oldDuration)
			newDurStr := formatDuration(newDuration)
			newCode = strings.Replace(newCode, oldDurStr, newDurStr, -1)
		}

		return oldCode, newCode, nil
	}

	return "", "", fmt.Errorf("could not extract code from line %d", lineStart)
}

// formatDuration formats a duration for code replacement
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%d*time.Millisecond", int(d.Milliseconds()))
	}
	return fmt.Sprintf("%d*time.Second", int(d.Seconds()))
}
