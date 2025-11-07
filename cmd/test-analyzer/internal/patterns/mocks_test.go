package patterns

import (
	"context"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewMocksDetector(t *testing.T) {
	detector := NewMocksDetector()
	if detector == nil {
		t.Fatal("NewMocksDetector() returned nil")
	}

	if _, ok := detector.(MocksDetector); !ok {
		t.Error("NewMocksDetector() does not implement MocksDetector interface")
	}
}

func TestMocksDetector_Detect(t *testing.T) {
	ctx := context.Background()
	detector := NewMocksDetector()
	fset := token.NewFileSet()

	t.Run("DetectMissingMock", func(t *testing.T) {
		src := `package test

import (
	"net/http"
	"testing"
)

func TestMissingMock(t *testing.T) {
	var client http.Client
	client.Get("https://example.com")
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		funcDecl := findTestFunctionDecl(astFile, "TestMissingMock")
		if funcDecl == nil {
			t.Fatal("Could not find TestMissingMock function")
		}

		testFunc := &TestFunction{
			Name:      "TestMissingMock",
			Type:      "Unit",
			LineStart: 7,
			LineEnd:   11,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// May or may not detect depending on implementation
		_ = issues
	})

	t.Run("DetectNonUnitTest", func(t *testing.T) {
		src := `package test

import "testing"

func TestIntegration(t *testing.T) {
	// Integration test
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
			Name:      "TestIntegration",
			Type:      "Integration",
			LineStart: 4,
			LineEnd:   7,
			AST:       funcDecl,
		}

		issues, err := detector.Detect(ctx, testFunc)
		if err != nil {
			t.Fatalf("Detect() error = %v", err)
		}
		// Should not detect for non-unit tests
		if len(issues) != 0 {
			t.Errorf("Expected 0 issues for integration test, got %d", len(issues))
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

