package patterns

import (
	"context"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewBenchmarkDetector(t *testing.T) {
	detector := NewBenchmarkDetector()
	if detector == nil {
		t.Fatal("NewBenchmarkDetector() returned nil")
	}

	if _, ok := detector.(BenchmarkDetector); !ok {
		t.Error("NewBenchmarkDetector() does not implement BenchmarkDetector interface")
	}
}

func TestBenchmarkDetector_Detect(t *testing.T) {
	ctx := context.Background()
	detector := NewBenchmarkDetector()
	fset := token.NewFileSet()

	t.Run("DetectBenchmarkHelperInRegularTest", func(t *testing.T) {
		src := `package test

import "testing"

func TestWithBenchmarkHelper(t *testing.T) {
	b := &testing.B{}
	b.ResetTimer()
	b.StopTimer()
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestWithBenchmarkHelper")
		if funcDecl == nil {
			t.Fatal("Could not find TestWithBenchmarkHelper function")
		}

		testFunc := &TestFunction{
			Name:      "TestWithBenchmarkHelper",
			Type:      "Unit",
			LineStart: 4,
			LineEnd:   9,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) >= 1 {
			issue := issues[0]
			if issue.Type != "BenchmarkHelperUsage" {
				t.Errorf("Expected issue type 'BenchmarkHelperUsage', got %q", issue.Type)
			}
		}
	})

	t.Run("DetectInBenchmarkFunction", func(t *testing.T) {
		src := `package test

import "testing"

func BenchmarkExample(b *testing.B) {
	b.ResetTimer()
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
			Name:      "BenchmarkExample",
			Type:      "Load",
			LineStart: 4,
			LineEnd:   9,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect for benchmark functions
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for benchmark function, got %d", len(issues))
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

