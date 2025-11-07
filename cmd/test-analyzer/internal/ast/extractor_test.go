package ast

import (
	"context"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewExtractor(t *testing.T) {
	extractor := NewExtractor()
	if extractor == nil {
		t.Fatal("NewExtractor() returned nil")
	}

	// Test that it implements the Extractor interface
	if _, ok := extractor.(Extractor); !ok {
		t.Error("NewExtractor() does not implement Extractor interface")
	}
}

func TestExtractor_ExtractTestFunctions(t *testing.T) {
	ctx := context.Background()
	extractor := NewExtractor()
	fset := token.NewFileSet()

	t.Run("ExtractSingleTestFunction", func(t *testing.T) {
		src := `package test

import "testing"

func TestExample(t *testing.T) {
	// Test code
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 1 {
			t.Fatalf("Expected 1 function, got %d", len(functions))
		}

		fn := functions[0]
		if fn.Name != "TestExample" {
			t.Errorf("Expected function name 'TestExample', got %q", fn.Name)
		}
		if fn.Type != "Unit" {
			t.Errorf("Expected type 'Unit', got %q", fn.Type)
		}
		if fn.LineStart <= 0 {
			t.Errorf("Expected LineStart > 0, got %d", fn.LineStart)
		}
		if fn.LineEnd <= 0 {
			t.Errorf("Expected LineEnd > 0, got %d", fn.LineEnd)
		}
	})

	t.Run("ExtractMultipleTestFunctions", func(t *testing.T) {
		src := `package test

import "testing"

func TestOne(t *testing.T) {
}

func TestTwo(t *testing.T) {
}

func TestThree(t *testing.T) {
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 3 {
			t.Fatalf("Expected 3 functions, got %d", len(functions))
		}

		names := make(map[string]bool)
		for _, fn := range functions {
			names[fn.Name] = true
			if fn.Type != "Unit" {
				t.Errorf("Expected type 'Unit' for %s, got %q", fn.Name, fn.Type)
			}
		}

		expectedNames := []string{"TestOne", "TestTwo", "TestThree"}
		for _, name := range expectedNames {
			if !names[name] {
				t.Errorf("Expected function %q not found", name)
			}
		}
	})

	t.Run("ExtractBenchmarkFunction", func(t *testing.T) {
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

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 1 {
			t.Fatalf("Expected 1 function, got %d", len(functions))
		}

		fn := functions[0]
		if fn.Name != "BenchmarkExample" {
			t.Errorf("Expected function name 'BenchmarkExample', got %q", fn.Name)
		}
		if fn.Type != "Load" {
			t.Errorf("Expected type 'Load', got %q", fn.Type)
		}
	})

	t.Run("ExtractFuzzFunction", func(t *testing.T) {
		src := `package test

import "testing"

func FuzzExample(f *testing.F) {
	f.Add("test")
	f.Fuzz(func(t *testing.T, s string) {
	})
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 1 {
			t.Fatalf("Expected 1 function, got %d", len(functions))
		}

		fn := functions[0]
		if fn.Name != "FuzzExample" {
			t.Errorf("Expected function name 'FuzzExample', got %q", fn.Name)
		}
		if fn.Type != "Load" {
			t.Errorf("Expected type 'Load', got %q", fn.Type)
		}
	})

	t.Run("ExtractMixedFunctionTypes", func(t *testing.T) {
		src := `package test

import "testing"

func TestUnit(t *testing.T) {
}

func BenchmarkLoad(b *testing.B) {
}

func FuzzFuzz(f *testing.F) {
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 3 {
			t.Fatalf("Expected 3 functions, got %d", len(functions))
		}

		typeMap := make(map[string]string)
		for _, fn := range functions {
			typeMap[fn.Name] = fn.Type
		}

		if typeMap["TestUnit"] != "Unit" {
			t.Errorf("Expected TestUnit to be 'Unit', got %q", typeMap["TestUnit"])
		}
		if typeMap["BenchmarkLoad"] != "Load" {
			t.Errorf("Expected BenchmarkLoad to be 'Load', got %q", typeMap["BenchmarkLoad"])
		}
		if typeMap["FuzzFuzz"] != "Load" {
			t.Errorf("Expected FuzzFuzz to be 'Load', got %q", typeMap["FuzzFuzz"])
		}
	})

	t.Run("ExtractFromFileWithNoTestFunctions", func(t *testing.T) {
		src := `package test

import "testing"

func regularFunction() {
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 0 {
			t.Fatalf("Expected 0 functions, got %d", len(functions))
		}
	})

	t.Run("ExtractFromNilAST", func(t *testing.T) {
		functions, err := extractor.ExtractTestFunctions(ctx, nil, fset)
		if err == nil {
			t.Error("Expected error for nil AST, got nil")
		}
		if functions != nil {
			t.Errorf("Expected nil functions, got %v", functions)
		}
	})

	t.Run("ExtractWithContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		src := `package test

import "testing"

func TestExample(t *testing.T) {
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err == nil {
			t.Error("Expected error for cancelled context, got nil")
		}
		if functions != nil {
			t.Errorf("Expected nil functions, got %v", functions)
		}
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})

	t.Run("ExtractFunctionsWithLineNumbers", func(t *testing.T) {
		src := `package test

import "testing"

func TestFirst(t *testing.T) {
	// Line 5
	// Line 6
}

func TestSecond(t *testing.T) {
	// Line 10
	// Line 11
	// Line 12
}
`
		astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 2 {
			t.Fatalf("Expected 2 functions, got %d", len(functions))
		}

		// Verify line numbers are set correctly
		for _, fn := range functions {
			if fn.LineStart <= 0 {
				t.Errorf("Expected LineStart > 0 for %s, got %d", fn.Name, fn.LineStart)
			}
			if fn.LineEnd <= 0 {
				t.Errorf("Expected LineEnd > 0 for %s, got %d", fn.Name, fn.LineEnd)
			}
			if fn.LineEnd < fn.LineStart {
				t.Errorf("Expected LineEnd >= LineStart for %s, got LineEnd=%d, LineStart=%d",
					fn.Name, fn.LineEnd, fn.LineStart)
			}
		}
	})
}

func TestDetermineFunctionType(t *testing.T) {
	// Test determineFunctionType indirectly through ExtractTestFunctions
	// since it's a private function

	tests := []struct {
		name        string
		funcName    string
		expectedType string
	}{
		{"TestFunction", "TestExample", "Unit"},
		{"BenchmarkFunction", "BenchmarkExample", "Load"},
		{"FuzzFunction", "FuzzExample", "Load"},
		{"TestWithSuffix", "TestExampleWithSuffix", "Unit"},
		{"BenchmarkWithSuffix", "BenchmarkExampleWithSuffix", "Load"},
		{"FuzzWithSuffix", "FuzzExampleWithSuffix", "Load"},
	}

	ctx := context.Background()
	extractor := NewExtractor()
	fset := token.NewFileSet()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := `package test

import "testing"

func ` + tt.funcName + `(t *testing.T) {
}
`
			astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
			if err != nil {
				t.Fatalf("ExtractTestFunctions() error = %v", err)
			}
			if len(functions) != 1 {
				t.Fatalf("Expected 1 function, got %d", len(functions))
			}

			if functions[0].Type != tt.expectedType {
				t.Errorf("Expected type %q, got %q", tt.expectedType, functions[0].Type)
			}
		})
	}
}

func TestExtractor_EdgeCases(t *testing.T) {
	ctx := context.Background()
	extractor := NewExtractor()
	fset := token.NewFileSet()

	t.Run("ExtractFromEmptyFile", func(t *testing.T) {
		src := `package test`
		astFile, err := goparser.ParseFile(fset, "empty.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 0 {
			t.Errorf("Expected 0 functions in empty file, got %d", len(functions))
		}
	})

	t.Run("ExtractFromFileWithOnlyImports", func(t *testing.T) {
		src := `package test

import "testing"
`
		astFile, err := goparser.ParseFile(fset, "imports.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 0 {
			t.Errorf("Expected 0 functions in file with only imports, got %d", len(functions))
		}
	})

	t.Run("ExtractFromFileWithVariedTestNames", func(t *testing.T) {
		src := `package test

import "testing"

func Test(t *testing.T) {
}

func TestA(t *testing.T) {
}

func Test123(t *testing.T) {
}

func Test_With_Underscores(t *testing.T) {
}
`
		astFile, err := goparser.ParseFile(fset, "varied.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 4 {
			t.Fatalf("Expected 4 test functions, got %d", len(functions))
		}

		names := make(map[string]bool)
		for _, fn := range functions {
			names[fn.Name] = true
		}

		expectedNames := []string{"Test", "TestA", "Test123", "Test_With_Underscores"}
		for _, name := range expectedNames {
			if !names[name] {
				t.Errorf("Expected function %q not found", name)
			}
		}
	})

	t.Run("ExtractFromFileWithNestedFunctions", func(t *testing.T) {
		src := `package test

import "testing"

func TestOuter(t *testing.T) {
	helper := func() {
		// Nested function
	}
	helper()
}
`
		astFile, err := goparser.ParseFile(fset, "nested.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 1 {
			t.Fatalf("Expected 1 test function, got %d", len(functions))
		}

		if functions[0].Name != "TestOuter" {
			t.Errorf("Expected function name 'TestOuter', got %q", functions[0].Name)
		}
	})

	t.Run("ExtractFromFileWithMethodReceivers", func(t *testing.T) {
		src := `package test

import "testing"

type TestSuite struct{}

func (ts *TestSuite) TestMethod(t *testing.T) {
	// Method with receiver
}

func TestFunction(t *testing.T) {
	// Regular test function
}
`
		astFile, err := goparser.ParseFile(fset, "methods.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		functions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) < 1 {
			t.Fatalf("Expected at least 1 test function, got %d", len(functions))
		}

		names := make(map[string]bool)
		for _, fn := range functions {
			names[fn.Name] = true
		}

		// TestFunction should definitely be included
		if !names["TestFunction"] {
			t.Error("Expected TestFunction not found")
		}
	})
}

func TestExtractor_ConsistencyWithParser(t *testing.T) {
	// Test that Extractor extracts functions correctly
	ctx := context.Background()
	extractor := NewExtractor()
	fset := token.NewFileSet()

	src := `package test

import "testing"

func TestOne(t *testing.T) {
}

func TestTwo(t *testing.T) {
}

func BenchmarkOne(b *testing.B) {
}
`

	astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	// Extract with extractor
	extractedFunctions, err := extractor.ExtractTestFunctions(ctx, astFile, fset)
	if err != nil {
		t.Fatalf("ExtractTestFunctions() error = %v", err)
	}

	if len(extractedFunctions) != 3 {
		t.Fatalf("Expected 3 functions, got %d", len(extractedFunctions))
	}

	names := make(map[string]bool)
	for _, fn := range extractedFunctions {
		names[fn.Name] = true
	}

	expectedNames := []string{"TestOne", "TestTwo", "BenchmarkOne"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("Expected function %q not found", name)
		}
	}
}

