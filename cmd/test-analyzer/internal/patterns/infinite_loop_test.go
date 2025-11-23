package patterns

import (
	"context"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewInfiniteLoopDetector(t *testing.T) {
	detector := NewInfiniteLoopDetector()
	if detector == nil {
		t.Fatal("NewInfiniteLoopDetector() returned nil")
	}

	// Test that it implements the InfiniteLoopDetector interface
	if _, ok := detector.(InfiniteLoopDetector); !ok {
		t.Error("NewInfiniteLoopDetector() does not implement InfiniteLoopDetector interface")
	}
}

func TestInfiniteLoopDetector_Detect(t *testing.T) {
	ctx := context.Background()
	detector := NewInfiniteLoopDetector()
	fset := token.NewFileSet()

	t.Run("DetectBasicInfiniteLoop", func(t *testing.T) {
		src := `package test

func TestInfiniteLoop(t *testing.T) {
	for {
		// This will run forever
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestInfiniteLoop")
		if funcDecl == nil {
			t.Fatal("Could not find TestInfiniteLoop function")
		}

		testFunc := &TestFunction{
			Name:      "TestInfiniteLoop",
			Type:      "Unit",
			LineStart: 3,
			LineEnd:   7,
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
		if issue.Type != "InfiniteLoop" {
			t.Errorf("Expected issue type 'InfiniteLoop', got %q", issue.Type)
		}
		if issue.Severity != "Critical" {
			t.Errorf("Expected severity 'Critical', got %q", issue.Severity)
		}
		if !issue.Fixable {
			t.Error("Expected issue to be fixable")
		}
	})

	t.Run("DetectForTrueLoop", func(t *testing.T) {
		src := `package test

func TestForTrue(t *testing.T) {
	for true {
		// This will run forever
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestForTrue")
		if funcDecl == nil {
			t.Fatal("Could not find TestForTrue function")
		}

		testFunc := &TestFunction{
			Name:      "TestForTrue",
			Type:      "Unit",
			LineStart: 3,
			LineEnd:   7,
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
		if issue.Type != "InfiniteLoop" {
			t.Errorf("Expected issue type 'InfiniteLoop', got %q", issue.Type)
		}
	})

	t.Run("DetectInfiniteLoopWithBreak", func(t *testing.T) {
		src := `package test

func TestInfiniteWithBreak(t *testing.T) {
	for {
		if someCondition {
			break
		}
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestInfiniteWithBreak")
		if funcDecl == nil {
			t.Fatal("Could not find TestInfiniteWithBreak function")
		}

		testFunc := &TestFunction{
			Name:      "TestInfiniteWithBreak",
			Type:      "Unit",
			LineStart: 3,
			LineEnd:   9,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect issue because it has an exit condition (break)
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues (has exit condition), got %d", len(issues))
		}
	})

	t.Run("DetectInfiniteLoopWithReturn", func(t *testing.T) {
		src := `package test

func TestInfiniteWithReturn(t *testing.T) {
	for {
		if done {
			return
		}
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestInfiniteWithReturn")
		if funcDecl == nil {
			t.Fatal("Could not find TestInfiniteWithReturn function")
		}

		testFunc := &TestFunction{
			Name:      "TestInfiniteWithReturn",
			Type:      "Unit",
			LineStart: 3,
			LineEnd:   9,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect issue because it has an exit condition (return)
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues (has exit condition), got %d", len(issues))
		}
	})

	t.Run("DetectConcurrentTestRunnerPattern", func(t *testing.T) {
		src := `package test

import "time"

func TestConcurrentRunner(t *testing.T) {
	timer := time.NewTicker(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// Do something
			return
		}
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestConcurrentRunner")
		if funcDecl == nil {
			t.Fatal("Could not find TestConcurrentRunner function")
		}

		testFunc := &TestFunction{
			Name:      "TestConcurrentRunner",
			Type:      "Unit",
			LineStart: 5,
			LineEnd:   16,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("Expected 1 issue (ConcurrentTestRunner pattern), got %d", len(issues))
		}

		issue := issues[0]
		if issue.Type != "InfiniteLoop" {
			t.Errorf("Expected issue type 'InfiniteLoop', got %q", issue.Type)
		}
		if issue.Severity != "High" {
			t.Errorf("Expected severity 'High' for ConcurrentTestRunner, got %q", issue.Severity)
		}
		if issue.Context["pattern"] != "ConcurrentTestRunner" {
			t.Errorf("Expected pattern 'ConcurrentTestRunner', got %v", issue.Context["pattern"])
		}
	})

	t.Run("DetectConcurrentTestRunnerWithAssignment", func(t *testing.T) {
		src := `package test

import "time"

func TestConcurrentRunnerAssign(t *testing.T) {
	timer := time.NewTicker(100 * time.Millisecond)
	defer timer.Stop()

	for {
		select {
		case val := <-timer.C:
			_ = val
			return
		}
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestConcurrentRunnerAssign")
		if funcDecl == nil {
			t.Fatal("Could not find TestConcurrentRunnerAssign function")
		}

		testFunc := &TestFunction{
			Name:      "TestConcurrentRunnerAssign",
			Type:      "Unit",
			LineStart: 5,
			LineEnd:   16,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 1 {
			t.Fatalf("Expected 1 issue (ConcurrentTestRunner pattern), got %d", len(issues))
		}

		issue := issues[0]
		if issue.Context["pattern"] != "ConcurrentTestRunner" {
			t.Errorf("Expected pattern 'ConcurrentTestRunner', got %v", issue.Context["pattern"])
		}
	})

	t.Run("DetectNormalForLoop", func(t *testing.T) {
		src := `package test

func TestNormalLoop(t *testing.T) {
	for i := 0; i < 10; i++ {
		// Normal loop
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestNormalLoop")
		if funcDecl == nil {
			t.Fatal("Could not find TestNormalLoop function")
		}

		testFunc := &TestFunction{
			Name:      "TestNormalLoop",
			Type:      "Unit",
			LineStart: 3,
			LineEnd:   7,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect issue for normal for loop
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for normal loop, got %d", len(issues))
		}
	})

	t.Run("DetectMultipleInfiniteLoops", func(t *testing.T) {
		src := `package test

func TestMultipleLoops(t *testing.T) {
	for {
		// First infinite loop
	}
	
	for true {
		// Second infinite loop
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestMultipleLoops")
		if funcDecl == nil {
			t.Fatal("Could not find TestMultipleLoops function")
		}

		testFunc := &TestFunction{
			Name:      "TestMultipleLoops",
			Type:      "Unit",
			LineStart: 3,
			LineEnd:   12,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) != 2 {
			t.Fatalf("Expected 2 issues, got %d", len(issues))
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

	// Note: The infinite loop detector doesn't currently check for context cancellation
	// This test verifies normal detection still works
	t.Run("DetectWithCancelledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		src := `package test

func TestCancelled(t *testing.T) {
	for {
	}
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestCancelled")
		if funcDecl == nil {
			t.Fatal("Could not find TestCancelled function")
		}

		testFunc := &TestFunction{
			Name:      "TestCancelled",
			Type:      "Unit",
			LineStart: 3,
			LineEnd:   5,
			AST:       funcDecl,
		}

		// Note: Detector doesn't check context cancellation, so it will still process
		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should still detect the infinite loop even with cancelled context
		// (since detector doesn't check context)
		if len(issues) != 1 {
			t.Errorf("Expected 1 issue, got %d", len(issues))
		}
	})
}

// findTestFunctionDecl finds a test function declaration by name in an AST file.
func findTestFunctionDecl(astFile *ast.File, name string) *ast.FuncDecl {
	for _, decl := range astFile.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Name != nil && fn.Name.Name == name {
				return fn
			}
		}
	}
	return nil
}
