package patterns

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"time"
)

// SleepDetector detects sleep delays.
type SleepDetector interface {
	Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error)
}

// sleepDetector implements SleepDetector.
type sleepDetector struct {
	threshold time.Duration
}

// NewSleepDetector creates a new SleepDetector.
func NewSleepDetector() SleepDetector {
	return &sleepDetector{
		threshold: 100 * time.Millisecond, // 100ms default
	}
}

// Detect implements SleepDetector.Detect.
func (d *sleepDetector) Detect(ctx context.Context, function *TestFunction) ([]PerformanceIssue, error) {
	var issues []PerformanceIssue

	// Get function AST
	funcDecl := getFuncDecl(function)
	if funcDecl == nil || funcDecl.Body == nil {
		return issues, nil
	}

	// Find all time.Sleep calls and accumulate duration
	totalSleep := d.accumulateSleepDuration(funcDecl.Body)

	if totalSleep > d.threshold {
		severity := "Medium"
		if totalSleep > 1*time.Second {
			severity = "High"
		}

		issue := PerformanceIssue{
			Type:        "SleepDelay",
			Severity:    severity,
			Location: Location{
				File:      getFilePath(function),
				Function:  function.Name,
				LineStart: function.LineStart,
				LineEnd:   function.LineEnd,
			},
			Description: fmt.Sprintf("Test function %s contains %v of sleep delays (threshold: %v)", 
				function.Name, totalSleep, d.threshold),
			Context: map[string]interface{}{
				"total_sleep": totalSleep.String(),
				"threshold":   d.threshold.String(),
				"function":    function.Name,
			},
			Fixable: true,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// accumulateSleepDuration finds all time.Sleep calls and sums their durations.
func (d *sleepDetector) accumulateSleepDuration(body *ast.BlockStmt) time.Duration {
	var total time.Duration

	if body == nil {
		return total
	}

	// Find all function calls
	calls := findCallExprs(body)

	for _, call := range calls {
		if isTimeSleepCall(call) {
			if len(call.Args) > 0 {
				duration := d.extractDuration(call.Args[0])
				total += duration
			}
		}
	}

	return total
}

// extractDuration attempts to extract a duration from an AST expression.
func (d *sleepDetector) extractDuration(expr ast.Expr) time.Duration {
	// Handle basic literals: time.Sleep(100 * time.Millisecond)
	if basicLit, ok := expr.(*ast.BasicLit); ok {
		// Try to parse as duration string like "100ms"
		if basicLit.Kind == token.STRING {
			if dur, err := time.ParseDuration(basicLit.Value); err == nil {
				return dur
			}
		}
		// Try to parse as number (assume nanoseconds)
		if basicLit.Kind == token.INT {
			if val, err := strconv.ParseInt(basicLit.Value, 10, 64); err == nil {
				return time.Duration(val)
			}
		}
	}

	// Handle binary expressions: 100 * time.Millisecond
	if binary, ok := expr.(*ast.BinaryExpr); ok && binary.Op == token.MUL {
		// Try to extract multiplier and unit
		if basicLit, ok := binary.X.(*ast.BasicLit); ok {
			if sel, ok := binary.Y.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "time" {
					multiplier, _ := strconv.ParseInt(basicLit.Value, 10, 64)
					unit := d.getTimeUnit(sel.Sel.Name)
					return time.Duration(multiplier) * unit
				}
			}
		}
	}

	// Handle selector expressions: time.Second, time.Minute, etc.
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "time" {
			unit := d.getTimeUnit(sel.Sel.Name)
			return unit
		}
	}

	return 0
}

// getTimeUnit converts a time unit name to duration.
func (d *sleepDetector) getTimeUnit(name string) time.Duration {
	switch name {
	case "Nanosecond":
		return time.Nanosecond
	case "Microsecond":
		return time.Microsecond
	case "Millisecond":
		return time.Millisecond
	case "Second":
		return time.Second
	case "Minute":
		return time.Minute
	case "Hour":
		return time.Hour
	default:
		return 0
	}
}

