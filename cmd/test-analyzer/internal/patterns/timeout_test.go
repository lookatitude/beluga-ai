package patterns

import (
	"context"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewTimeoutDetector(t *testing.T) {
	detector := NewTimeoutDetector()
	if detector == nil {
		t.Fatal("NewTimeoutDetector() returned nil")
	}

	if _, ok := detector.(TimeoutDetector); !ok {
		t.Error("NewTimeoutDetector() does not implement TimeoutDetector interface")
	}
}

func TestTimeoutDetector_Detect(t *testing.T) {
	ctx := context.Background()
	detector := NewTimeoutDetector()
	fset := token.NewFileSet()

	t.Run("DetectMissingTimeout", func(t *testing.T) {
		src := `package test

import "testing"

func TestNoTimeout(t *testing.T) {
	// No timeout mechanism
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestNoTimeout")
		if funcDecl == nil {
			t.Fatal("Could not find TestNoTimeout function")
		}

		testFunc := &TestFunction{
			Name:       "TestNoTimeout",
			Type:       "Unit",
			LineStart:  4,
			LineEnd:    7,
			HasTimeout: false,
			AST:        funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != "MissingTimeout" {
			t.Errorf("Expected issue type 'MissingTimeout', got %q", issue.Type)
		}
		if issue.Severity != "High" {
			t.Errorf("Expected severity 'High' for unit test, got %q", issue.Severity)
		}
	})

	t.Run("DetectWithContextWithTimeout", func(t *testing.T) {
		src := `package test

import (
	"context"
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestWithTimeout")
		if funcDecl == nil {
			t.Fatal("Could not find TestWithTimeout function")
		}

		testFunc := &TestFunction{
			Name:       "TestWithTimeout",
			Type:       "Unit",
			LineStart:  8,
			LineEnd:    12,
			HasTimeout: false,
			AST:        funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect issue because context.WithTimeout is present
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues (has timeout), got %d", len(issues))
		}
	})

	t.Run("DetectWithTimeAfterInSelect", func(t *testing.T) {
		src := `package test

import (
	"testing"
	"time"
)

func TestWithTimeAfter(t *testing.T) {
	select {
	case <-time.After(5 * time.Second):
		return
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestWithTimeAfter")
		if funcDecl == nil {
			t.Fatal("Could not find TestWithTimeAfter function")
		}

		testFunc := &TestFunction{
			Name:       "TestWithTimeAfter",
			Type:       "Unit",
			LineStart:  7,
			LineEnd:    12,
			HasTimeout: false,
			AST:        funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect issue because time.After is present in select
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues (has timeout), got %d", len(issues))
		}
	})

	t.Run("DetectIntegrationTestMissingTimeout", func(t *testing.T) {
		src := `package test

import "testing"

func TestIntegration(t *testing.T) {
	// Integration test without timeout
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestIntegration")
		if funcDecl == nil {
			t.Fatal("Could not find TestIntegration function")
		}

		testFunc := &TestFunction{
			Name:       "TestIntegration",
			Type:       "Integration",
			LineStart:  4,
			LineEnd:    7,
			HasTimeout: false,
			AST:        funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Severity != "Medium" {
			t.Errorf("Expected severity 'Medium' for integration test, got %q", issue.Severity)
		}
	})

	t.Run("DetectLoadTestMissingTimeout", func(t *testing.T) {
		src := `package test

import "testing"

func BenchmarkExample(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "BenchmarkExample")
		if funcDecl == nil {
			t.Fatal("Could not find BenchmarkExample function")
		}

		testFunc := &TestFunction{
			Name:       "BenchmarkExample",
			Type:       "Load",
			LineStart:  4,
			LineEnd:    7,
			HasTimeout: false,
			AST:        funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Severity != "Low" {
			t.Errorf("Expected severity 'Low' for load test, got %q", issue.Severity)
		}
	})

	t.Run("DetectWithHasTimeoutFlag", func(t *testing.T) {
		src := `package test

import "testing"

func TestWithFlag(t *testing.T) {
	// Test with timeout flag set
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestWithFlag")
		if funcDecl == nil {
			t.Fatal("Could not find TestWithFlag function")
		}

		testFunc := &TestFunction{
			Name:       "TestWithFlag",
			Type:       "Unit",
			LineStart:  4,
			LineEnd:    7,
			HasTimeout: true, // Flag is set
			AST:        funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect issue because HasTimeout flag is true
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues (has timeout flag), got %d", len(issues))
		}
	})

	t.Run("DetectWithNilAST", func(t *testing.T) {
		testFunc := &TestFunction{
			Name:       "TestNilAST",
			Type:       "Unit",
			LineStart:  1,
			LineEnd:    5,
			HasTimeout: false,
			AST:        nil,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for nil AST, got %d", len(issues))
		}
	})

	t.Run("DetectComplexTimeoutPattern", func(t *testing.T) {
		src := `package test

import (
	"context"
	"testing"
	"time"
)

func TestComplexTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
		// Do something
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestComplexTimeout")
		if funcDecl == nil {
			t.Fatal("Could not find TestComplexTimeout function")
		}

		testFunc := &TestFunction{
			Name:       "TestComplexTimeout",
			Type:       "Unit",
			LineStart:  8,
			LineEnd:    20,
			HasTimeout: false,
			AST:        funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect issue because timeout mechanisms are present
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues (has timeout), got %d", len(issues))
		}
	})
}
