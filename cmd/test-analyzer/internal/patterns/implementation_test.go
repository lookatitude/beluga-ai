package patterns

import (
	"context"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewImplementationDetector(t *testing.T) {
	detector := NewImplementationDetector()
	if detector == nil {
		t.Fatal("NewImplementationDetector() returned nil")
	}

	if _, ok := detector.(ImplementationDetector); !ok {
		t.Error("NewImplementationDetector() does not implement ImplementationDetector interface")
	}
}

func TestImplementationDetector_Detect(t *testing.T) {
	ctx := context.Background()
	detector := NewImplementationDetector()
	fset := token.NewFileSet()

	t.Run("DetectActualImplementationUsage", func(t *testing.T) {
		src := `package test

import (
	"net/http"
	"testing"
)

func TestActualImplementation(t *testing.T) {
	client := http.Client{}
	client.Get("https://example.com")
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestActualImplementation")
		if funcDecl == nil {
			t.Fatal("Could not find TestActualImplementation function")
		}

		testFunc := &TestFunction{
			Name:      "TestActualImplementation",
			Type:      "Unit",
			LineStart: 7,
			LineEnd:   11,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		if len(issues) >= 1 {
			issue := issues[0]
			if issue.Type != "ActualImplementationUsage" {
				t.Errorf("Expected issue type 'ActualImplementationUsage', got %q", issue.Type)
			}
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

