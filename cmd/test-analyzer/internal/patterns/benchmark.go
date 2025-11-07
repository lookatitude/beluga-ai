package patterns

import (
	"context"
	"fmt"
	"go/ast"
	"strings"
)

// BenchmarkDetector detects benchmark helper usage in regular tests.
type BenchmarkDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// benchmarkDetector implements BenchmarkDetector.
type benchmarkDetector struct{}

// NewBenchmarkDetector creates a new BenchmarkDetector.
func NewBenchmarkDetector() BenchmarkDetector {
	return &benchmarkDetector{}
}

// Detect implements BenchmarkDetector.Detect.
func (d *benchmarkDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	var issues []PerformanceIssue

	// Only check non-benchmark tests
	if function.Type == "Load" || strings.HasPrefix(function.Name, "Benchmark") {
		return issues, nil
	}

	// Get function AST
	funcDecl := getFuncDecl(function)
	if funcDecl == nil || funcDecl.Body == nil {
		return issues, nil
	}

	// Find benchmark helper calls
	benchmarkHelpers := d.findBenchmarkHelpers(funcDecl.Body)

	if len(benchmarkHelpers) > 0 {
		issue := PerformanceIssue{
			Type:     "BenchmarkHelperUsage",
			Severity: "Low",
			Location: Location{
				File:      getFilePath(function),
				Function:  function.Name,
				LineStart: function.LineStart,
				LineEnd:   function.LineEnd,
			},
			Description: fmt.Sprintf("Test function %s uses benchmark helpers: %s", 
				function.Name, strings.Join(benchmarkHelpers, ", ")),
			Context: map[string]interface{}{
				"helpers":      benchmarkHelpers,
				"helper_count": len(benchmarkHelpers),
				"test_type":    function.Type,
			},
			Fixable: false, // Usually intentional
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// findBenchmarkHelpers finds benchmark helper function calls.
func (d *benchmarkDetector) findBenchmarkHelpers(body *ast.BlockStmt) []string {
	var helpers []string

	if body == nil {
		return helpers
	}

	// Find all function calls
	calls := findCallExprs(body)

	for _, call := range calls {
		helper := d.isBenchmarkHelper(call)
		if helper != "" {
			helpers = append(helpers, helper)
		}
	}

	return helpers
}

// isBenchmarkHelper checks if a call is a benchmark helper.
func (d *benchmarkDetector) isBenchmarkHelper(call *ast.CallExpr) string {
	// Check for common benchmark helper patterns
	// b.ResetTimer(), b.StopTimer(), b.StartTimer(), etc.
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		// Check if it's called on 'b' (benchmark parameter)
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "b" {
			method := sel.Sel.Name
			benchmarkMethods := []string{
				"ResetTimer", "StopTimer", "StartTimer",
				"ReportAllocs", "ReportMetric", "SetBytes",
			}
			for _, bm := range benchmarkMethods {
				if method == bm {
					return fmt.Sprintf("b.%s", method)
				}
			}
		}
	}

	// Check for standalone benchmark helper functions
	if ident, ok := call.Fun.(*ast.Ident); ok {
		name := ident.Name
		benchmarkFuncs := []string{
			"Benchmark", "RunBenchmarks", "benchmarkHelper",
		}
		for _, bf := range benchmarkFuncs {
			if strings.Contains(strings.ToLower(name), strings.ToLower(bf)) {
				return name
			}
		}
	}

	return ""
}

