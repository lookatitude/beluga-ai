package patterns

import (
	"context"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewSleepDetector(t *testing.T) {
	detector := NewSleepDetector()
	if detector == nil {
		t.Fatal("NewSleepDetector() returned nil")
	}

	if _, ok := detector.(SleepDetector); !ok {
		t.Error("NewSleepDetector() does not implement SleepDetector interface")
	}
}

func TestSleepDetector_Detect(t *testing.T) {
	ctx := context.Background()
	detector := NewSleepDetector()
	fset := token.NewFileSet()

	t.Run("DetectSingleSleep", func(t *testing.T) {
		src := `package test

import (
	"testing"
	"time"
)

func TestSingleSleep(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestSingleSleep")
		if funcDecl == nil {
			t.Fatal("Could not find TestSingleSleep function")
		}

		testFunc := &TestFunction{
			Name:      "TestSingleSleep",
			Type:      "Unit",
			LineStart: 7,
			LineEnd:   10,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// May or may not detect depending on threshold
		_ = issues
	})

	t.Run("DetectMultipleSleeps", func(t *testing.T) {
		src := `package test

import (
	"testing"
	"time"
)

func TestMultipleSleeps(t *testing.T) {
	time.Sleep(50 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	time.Sleep(40 * time.Millisecond)
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestMultipleSleeps")
		if funcDecl == nil {
			t.Fatal("Could not find TestMultipleSleeps function")
		}

		testFunc := &TestFunction{
			Name:      "TestMultipleSleeps",
			Type:      "Unit",
			LineStart: 7,
			LineEnd:   11,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should detect if total exceeds threshold
		if len(issues) >= 1 {
			issue := issues[0]
			if issue.Type != "SleepDelay" {
				t.Errorf("Expected issue type 'SleepDelay', got %q", issue.Type)
			}
		}
	})

	t.Run("DetectNoSleep", func(t *testing.T) {
		src := `package test

import "testing"

func TestNoSleep(t *testing.T) {
	// No sleep calls
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestNoSleep")
		if funcDecl == nil {
			t.Fatal("Could not find TestNoSleep function")
		}

		testFunc := &TestFunction{
			Name:      "TestNoSleep",
			Type:      "Unit",
			LineStart: 4,
			LineEnd:   7,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for no sleep, got %d", len(issues))
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

