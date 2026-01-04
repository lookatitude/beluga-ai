package code

import (
	"context"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CodeChange represents a code modification.
type CodeChange struct {
	File        string
	LineStart   int
	LineEnd     int
	OldCode     string
	NewCode     string
	Description string
}

// CodeModifier is the interface for safely modifying Go source code.
type CodeModifier interface {
	// CreateBackup creates a backup copy of a file.
	CreateBackup(ctx context.Context, filePath string) (string, error)

	// ApplyCodeChange applies a code change to a file.
	ApplyCodeChange(ctx context.Context, change *CodeChange) error

	// FormatCode formats Go code using go/format.
	FormatCode(ctx context.Context, code string) (string, error)
}

// modifier implements the CodeModifier interface.
type modifier struct{}

// NewModifier creates a new CodeModifier instance.
func NewModifier() CodeModifier {
	return &modifier{}
}

// CreateBackup implements CodeModifier.CreateBackup.
func (m *modifier) CreateBackup(ctx context.Context, filePath string) (string, error) {
	// Read original file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	// Create backup filename with timestamp
	backupDir := filepath.Join(filepath.Dir(filePath), ".test-analyzer-backups")
	if err := os.MkdirAll(backupDir, 0750); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(backupDir, filepath.Base(filePath)+"."+timestamp+".bak")

	// Write backup
	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return "", fmt.Errorf("writing backup: %w", err)
	}

	return backupPath, nil
}

// ApplyCodeChange implements CodeModifier.ApplyCodeChange.
func (m *modifier) ApplyCodeChange(ctx context.Context, change *CodeChange) error {
	// Check if file exists
	_, err := os.Stat(change.File)
	fileExists := err == nil

	// If file doesn't exist and OldCode is empty, this is a new file creation
	if !fileExists && change.OldCode == "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(change.File)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		// Format and write new file
		formatted, err := m.FormatCode(ctx, change.NewCode)
		if err != nil {
			// If formatting fails, use unformatted code
			formatted = change.NewCode
		}

		// Add package declaration if missing
		if !strings.Contains(formatted, "package ") {
			// Extract package name from directory or use default
			packageName := filepath.Base(dir)
			formatted = fmt.Sprintf("package %s\n\n%s", packageName, formatted)
		}

		if err := os.WriteFile(change.File, []byte(formatted), 0600); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		return nil
	}

	// File exists or we're modifying existing file
	content, err := os.ReadFile(change.File)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")

	// Validate line numbers
	if change.LineStart < 1 || change.LineStart > len(lines) {
		return fmt.Errorf("invalid line start: %d", change.LineStart)
	}
	if change.LineEnd < change.LineStart || change.LineEnd > len(lines) {
		return fmt.Errorf("invalid line end: %d", change.LineEnd)
	}

	// Replace lines
	newLines := make([]string, 0, len(lines))
	newLines = append(newLines, lines[:change.LineStart-1]...)

	// Add new code
	newCodeLines := strings.Split(change.NewCode, "\n")
	newLines = append(newLines, newCodeLines...)

	// Add remaining lines
	if change.LineEnd < len(lines) {
		newLines = append(newLines, lines[change.LineEnd:]...)
	}

	// Join and format
	newContent := strings.Join(newLines, "\n")
	formatted, err := m.FormatCode(ctx, newContent)
	if err != nil {
		// If formatting fails, use unformatted code
		formatted = newContent
	}

	// Write file
	if err := os.WriteFile(change.File, []byte(formatted), 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

// FormatCode implements CodeModifier.FormatCode.
func (m *modifier) FormatCode(ctx context.Context, code string) (string, error) {
	formatted, err := format.Source([]byte(code))
	if err != nil {
		return "", fmt.Errorf("formatting code: %w", err)
	}
	return string(formatted), nil
}
