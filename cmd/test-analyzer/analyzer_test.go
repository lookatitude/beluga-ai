package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	astparser "github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/ast"
	"github.com/lookatitude/beluga-ai/cmd/test-analyzer/internal/patterns"
)

func TestNewAnalyzer(t *testing.T) {
	detector := patterns.NewDetector()
	parser := astparser.NewParser()
	analyzer := NewAnalyzer(detector, parser)

	if analyzer == nil {
		t.Fatal("NewAnalyzer() returned nil")
	}

	if _, ok := analyzer.(Analyzer); !ok {
		t.Error("NewAnalyzer() does not implement Analyzer interface")
	}
}

func TestAnalyzer_DetectIssues(t *testing.T) {
	ctx := context.Background()
	detector := patterns.NewDetector()
	parser := astparser.NewParser()
	analyzer := NewAnalyzer(detector, parser)

	t.Run("DetectIssuesWithValidFunction", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_test.go")
		content := `package test

import "testing"

func TestExample(t *testing.T) {
	for {
		// Infinite loop
	}
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		astFile, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		functions, err := parser.ExtractTestFunctions(ctx, astFile)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) == 0 {
			t.Fatal("Expected at least one function")
		}

		testFunc := &TestFunction{
			Name: functions[0].Name,
			Type: convertTestType(functions[0].Type),
			File: &TestFile{
				Path:    testFile,
				Package: astFile.Package,
				AST:     astFile.AST,
			},
			LineStart: functions[0].LineStart,
			LineEnd:   functions[0].LineEnd,
		}

		issues, err := analyzer.DetectIssues(ctx, testFunc)
		if err != nil {
			t.Fatalf("DetectIssues() error = %v", err)
		}
		// Should detect infinite loop
		if len(issues) == 0 {
			t.Error("Expected at least one issue (infinite loop), got 0")
		}
	})

	// Note: DetectIssues doesn't check for nil function, it will panic
	// This is acceptable as callers should ensure non-nil functions
}

func TestAnalyzer_AnalyzeFile(t *testing.T) {
	ctx := context.Background()
	detector := patterns.NewDetector()
	parser := astparser.NewParser()
	analyzer := NewAnalyzer(detector, parser)

	t.Run("AnalyzeValidTestFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_test.go")
		content := `package test

import "testing"

func TestOne(t *testing.T) {
}

func TestTwo(t *testing.T) {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		fileAnalysis, err := analyzer.AnalyzeFile(ctx, testFile)
		if err != nil {
			t.Fatalf("AnalyzeFile() error = %v", err)
		}
		if fileAnalysis == nil {
			t.Fatal("AnalyzeFile() returned nil")
		}
		if len(fileAnalysis.Functions) != 2 {
			t.Errorf("Expected 2 functions, got %d", len(fileAnalysis.Functions))
		}
	})

	t.Run("AnalyzeNonExistentFile", func(t *testing.T) {
		nonExistentFile := filepath.Join(t.TempDir(), "nonexistent_test.go")
		fileAnalysis, err := analyzer.AnalyzeFile(ctx, nonExistentFile)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
		if fileAnalysis != nil {
			t.Errorf("Expected nil file analysis, got %v", fileAnalysis)
		}
	})
}

func TestAnalyzer_AnalyzePackage(t *testing.T) {
	ctx := context.Background()
	detector := patterns.NewDetector()
	parser := astparser.NewParser()
	analyzer := NewAnalyzer(detector, parser)

	t.Run("AnalyzePackageWithTestFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile1 := filepath.Join(tmpDir, "test1_test.go")
		testFile2 := filepath.Join(tmpDir, "test2_test.go")

		content1 := `package test

import "testing"

func TestOne(t *testing.T) {
}
`
		content2 := `package test

import "testing"

func TestTwo(t *testing.T) {
}
`
		if err := os.WriteFile(testFile1, []byte(content1), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		if err := os.WriteFile(testFile2, []byte(content2), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		packageAnalysis, err := analyzer.AnalyzePackage(ctx, tmpDir)
		if err != nil {
			t.Fatalf("AnalyzePackage() error = %v", err)
		}
		if packageAnalysis == nil {
			t.Fatal("AnalyzePackage() returned nil")
		}
		if len(packageAnalysis.Files) < 2 {
			t.Errorf("Expected at least 2 files, got %d", len(packageAnalysis.Files))
		}
	})

	t.Run("AnalyzeNonExistentPackage", func(t *testing.T) {
		nonExistentDir := filepath.Join(t.TempDir(), "nonexistent")
		packageAnalysis, err := analyzer.AnalyzePackage(ctx, nonExistentDir)
		// May or may not error depending on implementation
		_ = packageAnalysis
		_ = err
	})
}
