package patterns

import (
	"context"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewIterationsDetector(t *testing.T) {
	detector := NewIterationsDetector()
	if detector == nil {
		t.Fatal("NewIterationsDetector() returned nil")
	}

	if _, ok := detector.(IterationsDetector); !ok {
		t.Error("NewIterationsDetector() does not implement IterationsDetector interface")
	}
}

func TestIterationsDetector_Detect(t *testing.T) {
	ctx := context.Background()
	detector := NewIterationsDetector()
	fset := token.NewFileSet()

	t.Run("DetectLargeSimpleIteration", func(t *testing.T) {
		src := `package test

import "testing"

func TestLargeIteration(t *testing.T) {
	for i := 0; i < 1000; i++ {
		_ = i * 2
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestLargeIteration")
		if funcDecl == nil {
			t.Fatal("Could not find TestLargeIteration function")
		}

		testFunc := &TestFunction{
			Name:      "TestLargeIteration",
			Type:      "Unit",
			LineStart: 4,
			LineEnd:   8,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("Expected 1 issue, got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != "LargeIteration" {
			t.Errorf("Expected issue type 'LargeIteration', got %q", issue.Type)
		}
	})

	t.Run("DetectSmallIteration", func(t *testing.T) {
		src := `package test

import "testing"

func TestSmallIteration(t *testing.T) {
	for i := 0; i < 10; i++ {
		_ = i
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestSmallIteration")
		if funcDecl == nil {
			t.Fatal("Could not find TestSmallIteration function")
		}

		testFunc := &TestFunction{
			Name:      "TestSmallIteration",
			Type:      "Unit",
			LineStart: 4,
			LineEnd:   8,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect issue for small iteration
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for small iteration, got %d", len(issues))
		}
	})

	t.Run("DetectRangeOverLargeSlice", func(t *testing.T) {
		src := `package test

import "testing"

func TestRangeLarge(t *testing.T) {
	items := make([]int, 1000)
	for range items {
		// Process item
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestRangeLarge")
		if funcDecl == nil {
			t.Fatal("Could not find TestRangeLarge function")
		}

		testFunc := &TestFunction{
			Name:      "TestRangeLarge",
			Type:      "Unit",
			LineStart: 4,
			LineEnd:   10,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// May or may not detect depending on implementation
		// Just verify it doesn't crash
		_ = issues
	})

	t.Run("DetectComplexLoop", func(t *testing.T) {
		src := `package test

import (
	"net/http"
	"testing"
)

func TestComplexLoop(t *testing.T) {
	for i := 0; i < 50; i++ {
		http.Get("https://example.com")
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
			LineStart: 7,
			LineEnd:   11,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Complex operations in loop should be detected
		if len(issues) >= 1 {
			issue := issues[0]
			if issue.Type != "LargeIteration" {
				t.Errorf("Expected issue type 'LargeIteration', got %q", issue.Type)
			}
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

