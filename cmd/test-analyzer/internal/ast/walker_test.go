package ast

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"testing"
)

func TestNewWalker(t *testing.T) {
	fset := token.NewFileSet()
	walker := NewWalker(fset)

	if walker == nil {
		t.Fatal("NewWalker() returned nil")
	}

	// Test that it implements the Walker interface
	if _, ok := walker.(Walker); !ok {
		t.Error("NewWalker() does not implement Walker interface")
	}

	// Test that it implements ast.Visitor
	if _, ok := walker.(ast.Visitor); !ok {
		t.Error("NewWalker() does not implement ast.Visitor interface")
	}
}

func TestWalker_Functions(t *testing.T) {
	fset := token.NewFileSet()
	walker := NewWalker(fset)

	// Initially should return empty slice
	functions := walker.Functions()
	if functions == nil {
		t.Error("Functions() should not return nil, should return empty slice")
	}
	if len(functions) != 0 {
		t.Errorf("Expected 0 functions initially, got %d", len(functions))
	}
}

func TestWalker_Visit(t *testing.T) {
	fset := token.NewFileSet()

	t.Run("VisitNilNode", func(t *testing.T) {
		walker := NewWalker(fset)
		result := walker.Visit(nil)
		if result != nil {
			t.Error("Expected nil visitor for nil node")
		}
	})

	t.Run("VisitNonFunctionNode", func(t *testing.T) {
		walker := NewWalker(fset)
		ident := &ast.Ident{Name: "test"}
		result := walker.Visit(ident)
		if result == nil {
			t.Error("Expected non-nil visitor for non-function node")
		}
		if len(walker.Functions()) != 0 {
			t.Error("Expected no functions collected for non-function node")
		}
	})

	t.Run("VisitTestFunction", func(t *testing.T) {
		walker := NewWalker(fset)
		testFunc := &ast.FuncDecl{
			Name: &ast.Ident{Name: "TestExample"},
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
			},
		}

		result := walker.Visit(testFunc)
		if result == nil {
			t.Error("Expected non-nil visitor for function node")
		}

		functions := walker.Functions()
		if len(functions) != 1 {
			t.Fatalf("Expected 1 function, got %d", len(functions))
		}
		if functions[0].Name.Name != "TestExample" {
			t.Errorf("Expected function name 'TestExample', got %q", functions[0].Name.Name)
		}
	})

	t.Run("VisitBenchmarkFunction", func(t *testing.T) {
		walker := NewWalker(fset)
		benchFunc := &ast.FuncDecl{
			Name: &ast.Ident{Name: "BenchmarkExample"},
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
			},
		}

		walker.Visit(benchFunc)
		functions := walker.Functions()
		if len(functions) != 1 {
			t.Fatalf("Expected 1 function, got %d", len(functions))
		}
		if functions[0].Name.Name != "BenchmarkExample" {
			t.Errorf("Expected function name 'BenchmarkExample', got %q", functions[0].Name.Name)
		}
	})

	t.Run("VisitFuzzFunction", func(t *testing.T) {
		walker := NewWalker(fset)
		fuzzFunc := &ast.FuncDecl{
			Name: &ast.Ident{Name: "FuzzExample"},
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
			},
		}

		walker.Visit(fuzzFunc)
		functions := walker.Functions()
		if len(functions) != 1 {
			t.Fatalf("Expected 1 function, got %d", len(functions))
		}
		if functions[0].Name.Name != "FuzzExample" {
			t.Errorf("Expected function name 'FuzzExample', got %q", functions[0].Name.Name)
		}
	})

	t.Run("VisitNonTestFunction", func(t *testing.T) {
		walker := NewWalker(fset)
		regularFunc := &ast.FuncDecl{
			Name: &ast.Ident{Name: "regularFunction"},
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
			},
		}

		walker.Visit(regularFunc)
		functions := walker.Functions()
		if len(functions) != 0 {
			t.Errorf("Expected 0 functions, got %d", len(functions))
		}
	})

	t.Run("VisitFunctionWithNilName", func(t *testing.T) {
		walker := NewWalker(fset)
		funcWithNilName := &ast.FuncDecl{
			Name: nil,
			Type: &ast.FuncType{
				Params: &ast.FieldList{},
			},
		}

		walker.Visit(funcWithNilName)
		functions := walker.Functions()
		if len(functions) != 0 {
			t.Errorf("Expected 0 functions for function with nil name, got %d", len(functions))
		}
	})

	t.Run("VisitMultipleFunctions", func(t *testing.T) {
		walker := NewWalker(fset)

		testFunc1 := &ast.FuncDecl{
			Name: &ast.Ident{Name: "TestOne"},
			Type: &ast.FuncType{Params: &ast.FieldList{}},
		}
		testFunc2 := &ast.FuncDecl{
			Name: &ast.Ident{Name: "TestTwo"},
			Type: &ast.FuncType{Params: &ast.FieldList{}},
		}
		benchFunc := &ast.FuncDecl{
			Name: &ast.Ident{Name: "BenchmarkOne"},
			Type: &ast.FuncType{Params: &ast.FieldList{}},
		}

		walker.Visit(testFunc1)
		walker.Visit(testFunc2)
		walker.Visit(benchFunc)

		functions := walker.Functions()
		if len(functions) != 3 {
			t.Fatalf("Expected 3 functions, got %d", len(functions))
		}

		names := make(map[string]bool)
		for _, fn := range functions {
			names[fn.Name.Name] = true
		}

		expectedNames := []string{"TestOne", "TestTwo", "BenchmarkOne"}
		for _, name := range expectedNames {
			if !names[name] {
				t.Errorf("Expected function %q not found", name)
			}
		}
	})
}

func TestIsTestFunction(t *testing.T) {
	fset := token.NewFileSet()
	walker := NewWalker(fset).(*walker)

	tests := []struct {
		name     string
		funcName string
		expected bool
	}{
		{"TestFunction", "TestExample", true},
		{"TestFunctionWithSuffix", "TestExampleWithSuffix", true},
		{"BenchmarkFunction", "BenchmarkExample", true},
		{"BenchmarkFunctionWithSuffix", "BenchmarkExampleWithSuffix", true},
		{"FuzzFunction", "FuzzExample", true},
		{"FuzzFunctionWithSuffix", "FuzzExampleWithSuffix", true},
		{"RegularFunction", "regularFunction", false},
		{"PrivateFunction", "privateFunction", false},
		{"InitFunction", "init", false},
		{"MainFunction", "main", false},
		{"EmptyName", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var funcDecl *ast.FuncDecl
			if tt.funcName != "" {
				funcDecl = &ast.FuncDecl{
					Name: &ast.Ident{Name: tt.funcName},
					Type: &ast.FuncType{Params: &ast.FieldList{}},
				}
			} else {
				funcDecl = &ast.FuncDecl{
					Name: nil,
					Type: &ast.FuncType{Params: &ast.FieldList{}},
				}
			}

			result := walker.isTestFunction(funcDecl)
			if result != tt.expected {
				t.Errorf("isTestFunction(%q) = %v, want %v", tt.funcName, result, tt.expected)
			}
		})
	}

	// Note: isTestFunction doesn't check for nil function pointer,
	// it only checks for nil Name. This is acceptable as the caller
	// (Visit method) ensures non-nil function declarations.
}

func TestWalker_WithRealAST(t *testing.T) {
	fset := token.NewFileSet()
	src := `package test

import "testing"

func TestExample(t *testing.T) {
	// Test code
}

func BenchmarkExample(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}

func FuzzExample(f *testing.F) {
	f.Add("test")
}

func regularFunction() {
	// Not a test function
}
`

	astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	walker := NewWalker(fset)
	ast.Walk(walker, astFile)

	functions := walker.Functions()
	if len(functions) != 3 {
		t.Fatalf("Expected 3 test functions, got %d", len(functions))
	}

	names := make(map[string]bool)
	for _, fn := range functions {
		names[fn.Name.Name] = true
	}

	expectedNames := []string{"TestExample", "BenchmarkExample", "FuzzExample"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("Expected function %q not found", name)
		}
	}

	// Verify regularFunction is not included
	if names["regularFunction"] {
		t.Error("regularFunction should not be included in test functions")
	}
}

func TestWalker_WithNestedFunctions(t *testing.T) {
	fset := token.NewFileSet()
	src := `package test

import "testing"

func TestOuter(t *testing.T) {
	helper := func() {
		// Nested function
	}
	helper()
}

func TestInner(t *testing.T) {
	// Another test
}
`

	astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	walker := NewWalker(fset)
	ast.Walk(walker, astFile)

	functions := walker.Functions()
	if len(functions) != 2 {
		t.Fatalf("Expected 2 test functions, got %d", len(functions))
	}

	names := make(map[string]bool)
	for _, fn := range functions {
		names[fn.Name.Name] = true
	}

	if !names["TestOuter"] {
		t.Error("Expected TestOuter function not found")
	}
	if !names["TestInner"] {
		t.Error("Expected TestInner function not found")
	}
}

func TestWalker_WithMethodReceivers(t *testing.T) {
	fset := token.NewFileSet()
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

	astFile, err := goparser.ParseFile(fset, "test.go", src, goparser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	walker := NewWalker(fset)
	ast.Walk(walker, astFile)

	functions := walker.Functions()
	// Methods with receivers starting with Test should be included
	// Regular test functions should be included
	if len(functions) < 1 {
		t.Fatalf("Expected at least 1 test function, got %d", len(functions))
	}

	names := make(map[string]bool)
	for _, fn := range functions {
		names[fn.Name.Name] = true
	}

	// TestFunction should definitely be included
	if !names["TestFunction"] {
		t.Error("Expected TestFunction not found")
	}
}

func TestWalker_EdgeCases(t *testing.T) {
	fset := token.NewFileSet()

	t.Run("VisitEmptyFile", func(t *testing.T) {
		src := `package test`
		astFile, err := goparser.ParseFile(fset, "empty.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		walker := NewWalker(fset)
		ast.Walk(walker, astFile)

		functions := walker.Functions()
		if len(functions) != 0 {
			t.Errorf("Expected 0 functions in empty file, got %d", len(functions))
		}
	})

	t.Run("VisitFileWithOnlyImports", func(t *testing.T) {
		src := `package test

import "testing"
`
		astFile, err := goparser.ParseFile(fset, "imports.go", src, goparser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse source: %v", err)
		}

		walker := NewWalker(fset)
		ast.Walk(walker, astFile)

		functions := walker.Functions()
		if len(functions) != 0 {
			t.Errorf("Expected 0 functions in file with only imports, got %d", len(functions))
		}
	})

	t.Run("VisitFileWithVariedTestNames", func(t *testing.T) {
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

		walker := NewWalker(fset)
		ast.Walk(walker, astFile)

		functions := walker.Functions()
		if len(functions) != 4 {
			t.Fatalf("Expected 4 test functions, got %d", len(functions))
		}

		names := make(map[string]bool)
		for _, fn := range functions {
			names[fn.Name.Name] = true
		}

		expectedNames := []string{"Test", "TestA", "Test123", "Test_With_Underscores"}
		for _, name := range expectedNames {
			if !names[name] {
				t.Errorf("Expected function %q not found", name)
			}
		}
	})
}

func TestWalker_StringPrefixMatching(t *testing.T) {
	fset := token.NewFileSet()
	walker := NewWalker(fset).(*walker)

	// Test that prefix matching works correctly
	testCases := []struct {
		name     string
		expected bool
	}{
		{"Test", true},
		{"TestA", true},
		{"Test123", true},
		{"Test_", true},
		{"TestExample", true},
		{"Benchmark", true},
		{"BenchmarkA", true},
		{"Benchmark123", true},
		{"Fuzz", true},
		{"FuzzA", true},
		{"Fuzz123", true},
		{"test", false}, // lowercase
		{"TEST", false}, // uppercase
		{"NotTest", false},
		{"TestHelper", true}, // starts with Test, so it's a test function
		{"HelperTest", false}, // doesn't start with Test
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			funcDecl := &ast.FuncDecl{
				Name: &ast.Ident{Name: tc.name},
				Type: &ast.FuncType{Params: &ast.FieldList{}},
			}

			result := walker.isTestFunction(funcDecl)
			if result != tc.expected {
				t.Errorf("isTestFunction(%q) = %v, want %v", tc.name, result, tc.expected)
			}
		})
	}
}

