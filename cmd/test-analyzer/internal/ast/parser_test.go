package ast

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	parser := NewParser()
	if parser == nil {
		t.Fatal("NewParser() returned nil")
	}

	// Test that it implements the Parser interface
	if _, ok := parser.(Parser); !ok {
		t.Error("NewParser() does not implement Parser interface")
	}
}

func TestParseFile(t *testing.T) {
	ctx := context.Background()
	parser := NewParser()

	t.Run("ParseValidTestFile", func(t *testing.T) {
		// Create a temporary test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_test.go")
		content := `package test

import "testing"

func TestExample(t *testing.T) {
	// Test code
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}
		if file == nil {
			t.Fatal("ParseFile() returned nil")
		}

		if file.Package != "test" {
			t.Errorf("Expected package 'test', got %q", file.Package)
		}
		if file.AST == nil {
			t.Error("Expected AST to be set, got nil")
		}
		if !strings.HasSuffix(file.Path, "test_test.go") {
			t.Errorf("Expected path to end with 'test_test.go', got %q", file.Path)
		}
		if file.HasIntegrationSuffix {
			t.Error("Expected HasIntegrationSuffix to be false for regular test file")
		}
	})

	t.Run("ParseIntegrationTestFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_integration_test.go")
		content := `package test

import "testing"

func TestIntegration(t *testing.T) {
	// Integration test code
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}
		if file == nil {
			t.Fatal("ParseFile() returned nil")
		}

		if !file.HasIntegrationSuffix {
			t.Error("Expected HasIntegrationSuffix to be true for integration test file")
		}
	})

	t.Run("ParseFileWithBenchmark", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "bench_test.go")
		content := `package test

import "testing"

func BenchmarkExample(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Benchmark code
	}
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}
		if file == nil {
			t.Fatal("ParseFile() returned nil")
		}

		if file.Package != "test" {
			t.Errorf("Expected package 'test', got %q", file.Package)
		}
	})

	t.Run("ParseFileWithMultipleFunctions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "multi_test.go")
		content := `package test

import "testing"

func TestOne(t *testing.T) {
}

func TestTwo(t *testing.T) {
}

func BenchmarkOne(b *testing.B) {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}
		if file == nil {
			t.Fatal("ParseFile() returned nil")
		}

		if file.Package != "test" {
			t.Errorf("Expected package 'test', got %q", file.Package)
		}
	})

	t.Run("ParseFileWithFuzz", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "fuzz_test.go")
		content := `package test

import "testing"

func FuzzExample(f *testing.F) {
	f.Add("test")
	f.Fuzz(func(t *testing.T, s string) {
		// Fuzz test code
	})
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}
		if file == nil {
			t.Fatal("ParseFile() returned nil")
		}

		if file.Package != "test" {
			t.Errorf("Expected package 'test', got %q", file.Package)
		}
	})

	t.Run("ParseFileWithPackageName", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "custom_test.go")
		content := `package custompkg

import "testing"

func TestExample(t *testing.T) {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}
		if file == nil {
			t.Fatal("ParseFile() returned nil")
		}

		if file.Package != "custompkg" {
			t.Errorf("Expected package 'custompkg', got %q", file.Package)
		}
	})

	t.Run("ParseFileNotFound", func(t *testing.T) {
		nonExistentFile := filepath.Join(t.TempDir(), "nonexistent_test.go")
		file, err := parser.ParseFile(ctx, nonExistentFile)
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
		if file != nil {
			t.Errorf("Expected nil file, got %v", file)
		}
		if !strings.Contains(err.Error(), "reading file") {
			t.Errorf("Expected error to mention 'reading file', got %q", err.Error())
		}
	})

	t.Run("ParseFileInvalidGo", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "invalid_test.go")
		content := `package test

invalid syntax here
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err == nil {
			t.Error("Expected error for invalid Go syntax, got nil")
		}
		if file != nil {
			t.Errorf("Expected nil file, got %v", file)
		}
		if !strings.Contains(err.Error(), "parsing file") {
			t.Errorf("Expected error to mention 'parsing file', got %q", err.Error())
		}
	})

	t.Run("ParseFileWithContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_test.go")
		content := `package test

func TestExample(t *testing.T) {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err == nil {
			t.Error("Expected error for cancelled context, got nil")
		}
		if file != nil {
			t.Errorf("Expected nil file, got %v", file)
		}
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})
}

func TestExtractTestFunctions(t *testing.T) {
	ctx := context.Background()
	parser := NewParser()

	t.Run("ExtractSingleTestFunction", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_test.go")
		content := `package test

import "testing"

func TestExample(t *testing.T) {
	// Test code
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		functions, err := parser.ExtractTestFunctions(ctx, file)
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
		if fn.File == nil {
			t.Error("Expected File to be set, got nil")
		}
		if fn.LineStart <= 0 {
			t.Errorf("Expected LineStart > 0, got %d", fn.LineStart)
		}
		if fn.LineEnd <= 0 {
			t.Errorf("Expected LineEnd > 0, got %d", fn.LineEnd)
		}
		if fn.LineEnd < fn.LineStart {
			t.Errorf("Expected LineEnd >= LineStart, got LineEnd=%d, LineStart=%d", fn.LineEnd, fn.LineStart)
		}
	})

	t.Run("ExtractMultipleTestFunctions", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "multi_test.go")
		content := `package test

import "testing"

func TestOne(t *testing.T) {
}

func TestTwo(t *testing.T) {
}

func TestThree(t *testing.T) {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		functions, err := parser.ExtractTestFunctions(ctx, file)
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
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "bench_test.go")
		content := `package test

import "testing"

func BenchmarkExample(b *testing.B) {
	for i := 0; i < b.N; i++ {
	}
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		functions, err := parser.ExtractTestFunctions(ctx, file)
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
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "fuzz_test.go")
		content := `package test

import "testing"

func FuzzExample(f *testing.F) {
	f.Add("test")
	f.Fuzz(func(t *testing.T, s string) {
	})
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		functions, err := parser.ExtractTestFunctions(ctx, file)
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

	t.Run("ExtractIntegrationTestFunction", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_integration_test.go")
		content := `package test

import "testing"

func TestIntegration(t *testing.T) {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		functions, err := parser.ExtractTestFunctions(ctx, file)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 1 {
			t.Fatalf("Expected 1 function, got %d", len(functions))
		}

		fn := functions[0]
		if fn.Type != "Integration" {
			t.Errorf("Expected type 'Integration', got %q", fn.Type)
		}
	})

	t.Run("ExtractMixedFunctionTypes", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "mixed_test.go")
		content := `package test

import "testing"

func TestUnit(t *testing.T) {
}

func BenchmarkLoad(b *testing.B) {
}

func FuzzFuzz(f *testing.F) {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		functions, err := parser.ExtractTestFunctions(ctx, file)
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
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "notest_test.go")
		content := `package test

import "testing"

func regularFunction() {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		functions, err := parser.ExtractTestFunctions(ctx, file)
		if err != nil {
			t.Fatalf("ExtractTestFunctions() error = %v", err)
		}
		if len(functions) != 0 {
			t.Fatalf("Expected 0 functions, got %d", len(functions))
		}
	})

	t.Run("ExtractFromNilAST", func(t *testing.T) {
		file := &TestFile{
			Path:    "test.go",
			Package: "test",
			AST:     nil,
		}

		functions, err := parser.ExtractTestFunctions(ctx, file)
		if err == nil {
			t.Error("Expected error for nil AST, got nil")
		}
		if functions != nil {
			t.Errorf("Expected nil functions, got %v", functions)
		}
		if !strings.Contains(err.Error(), "AST is nil") {
			t.Errorf("Expected error to mention 'AST is nil', got %q", err.Error())
		}
	})

	t.Run("ExtractWithContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test_test.go")
		content := `package test

import "testing"

func TestExample(t *testing.T) {
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		file, err := parser.ParseFile(ctx, testFile)
		if err != nil {
			t.Fatalf("ParseFile() error = %v", err)
		}

		// Cancel context before extraction
		cancel()

		functions, err := parser.ExtractTestFunctions(ctx, file)
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
}

func TestDetermineTestType(t *testing.T) {
	// Test determineTestType indirectly through ExtractTestFunctions
	// since it's a private function

	tests := []struct {
		name                 string
		fileName             string
		funcName             string
		expectedType          string
	}{
		{"TestFunction", "test_test.go", "TestExample", "Unit"},
		{"BenchmarkFunction", "bench_test.go", "BenchmarkExample", "Load"},
		{"FuzzFunction", "fuzz_test.go", "FuzzExample", "Load"},
		{"IntegrationTest", "test_integration_test.go", "TestIntegration", "Integration"},
		{"TestInIntegrationFile", "test_integration_test.go", "TestExample", "Integration"},
	}

	ctx := context.Background()
	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, tt.fileName)
			content := `package test

import "testing"

func ` + tt.funcName + `(t *testing.T) {
}
`
			if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			file, err := parser.ParseFile(ctx, testFile)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}

			functions, err := parser.ExtractTestFunctions(ctx, file)
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

