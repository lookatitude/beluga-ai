package patterns

import (
	"context"
	"fmt"
	"go/ast"
)

// TimeoutDetector detects missing timeout mechanisms.
type TimeoutDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// timeoutDetector implements TimeoutDetector.
type timeoutDetector struct{}

// NewTimeoutDetector creates a new TimeoutDetector.
func NewTimeoutDetector() TimeoutDetector {
	return &timeoutDetector{}
}

// Detect implements TimeoutDetector.Detect.
func (d *timeoutDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	var issues []PerformanceIssue

	// Get function AST
	funcDecl := getFuncDecl(function)
	if funcDecl == nil || funcDecl.Body == nil {
		return issues, nil
	}

	// Check if function has timeout mechanism in AST
	hasTimeout := d.hasTimeoutMechanism(funcDecl)

	// Check if function has timeout flag (from metadata)
	if !function.HasTimeout && !hasTimeout {
		// Determine severity based on test type
		severity := "High"
		if function.Type == "Integration" {
			severity = "Medium"
		} else if function.Type == "Load" {
			severity = "Low" // Benchmarks might not need timeouts
		}

		issue := PerformanceIssue{
			Type:        "MissingTimeout",
			Severity:    severity,
			Location: Location{
				File:      getFilePath(function),
				Function:  function.Name,
				LineStart: function.LineStart,
				LineEnd:   function.LineEnd,
			},
			Description: fmt.Sprintf("Test function %s is missing a timeout mechanism", function.Name),
			Context: map[string]interface{}{
				"test_type": function.Type,
				"function":  function.Name,
			},
			Fixable: true,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// hasTimeoutMechanism checks if the function has a timeout mechanism.
func (d *timeoutDetector) hasTimeoutMechanism(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Body == nil {
		return false
	}

	// Find all function calls
	calls := findCallExprs(funcDecl.Body)

	// Check for context.WithTimeout
	for _, call := range calls {
		if isContextWithTimeout(call) {
			return true
		}
		// Also check for time.After in select statements
		if isTimeAfter(call) {
			return true
		}
	}

	// Check for select statements with time.After (common timeout pattern)
	selects := findSelectStmts(funcDecl.Body)
	for _, sel := range selects {
		if d.hasTimeoutInSelect(sel) {
			return true
		}
	}

	return false
}

// hasTimeoutInSelect checks if a select statement has a timeout case.
func (d *timeoutDetector) hasTimeoutInSelect(sel *ast.SelectStmt) bool {
	if sel.Body == nil {
		return false
	}

	for _, stmt := range sel.Body.List {
		if comm, ok := stmt.(*ast.CommClause); ok {
			// Check if this case uses time.After
			if comm.Comm != nil {
				if exprStmt, ok := comm.Comm.(*ast.ExprStmt); ok {
					if call, ok := exprStmt.X.(*ast.CallExpr); ok {
						if isTimeAfter(call) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

