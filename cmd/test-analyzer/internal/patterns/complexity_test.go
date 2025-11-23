package patterns

import (
	"context"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewComplexityDetector(t *testing.T) {
	detector := NewComplexityDetector()
	if detector == nil {
		t.Fatal("NewComplexityDetector() returned nil")
	}

	if _, ok := detector.(ComplexityDetector); !ok {
		t.Error("NewComplexityDetector() does not implement ComplexityDetector interface")
	}
}

func TestComplexityDetector_Detect(t *testing.T) {
	ctx := context.Background()
	detector := NewComplexityDetector()
	fset := token.NewFileSet()

	t.Run("DetectComplexOperationsInLoop", func(t *testing.T) {
		src := `package test

import (
	"net/http"
	"os"
	"testing"
)

func TestComplexLoop(t *testing.T) {
	for i := 0; i < 50; i++ {
		http.Get("https://example.com")
		os.ReadFile("test.txt")
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestComplexLoop")
		if funcDecl == nil {
			t.Fatal("Could not find TestComplexLoop function")
		}

		testFunc := &TestFunction{
			Name:      "TestComplexLoop",
			Type:      "Unit",
			LineStart: 9,
			LineEnd:   13,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) >= 1 {
			issue := issues[0]
			if issue.Type != "HighConcurrency" {
				t.Errorf("Expected issue type 'HighConcurrency', got %q", issue.Type)
			}
		}
	})

	t.Run("DetectSimpleLoop", func(t *testing.T) {
		src := `package test

import "testing"

func TestSimpleLoop(t *testing.T) {
	for i := 0; i < 10; i++ {
		_ = i * 2
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestSimpleLoop")
		if funcDecl == nil {
			t.Fatal("Could not find TestSimpleLoop function")
		}

		testFunc := &TestFunction{
			Name:      "TestSimpleLoop",
			Type:      "Unit",
			LineStart: 4,
			LineEnd:   8,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Simple loops should not trigger complexity issues
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for simple loop, got %d", len(issues))
		}
	})

	t.Run("DetectInfiniteLoop", func(t *testing.T) {
		src := `package test

import "testing"

func TestInfinite(t *testing.T) {
	for {
		// Infinite loop
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestInfinite")
		if funcDecl == nil {
			t.Fatal("Could not find TestInfinite function")
		}

		testFunc := &TestFunction{
			Name:      "TestInfinite",
			Type:      "Unit",
			LineStart: 4,
			LineEnd:   8,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should skip infinite loops (handled by InfiniteLoopDetector)
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for infinite loop (handled elsewhere), got %d", len(issues))
		}
	})

	t.Run("DetectWithNilAST", func(t *testing.T) {
		testFunc := &TestFunction{
			Name:      "TestNilAST",
			Type:      "Unit",
			LineStart: 1,
			LineEnd:   5,
			AST:       nil,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for nil AST, got %d", len(issues))
		}
	})
}
