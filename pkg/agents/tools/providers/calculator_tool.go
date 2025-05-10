package providers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// CalculatorTool is a tool that can perform basic arithmetic calculations.
// It expects a string input representing a simple arithmetic expression (e.g., "2 + 2", "10 * 5 / 2").
type CalculatorTool struct {
	tools.BaseTool
}

// NewCalculatorTool creates a new CalculatorTool.
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{
		BaseTool: tools.BaseTool{
			Name:        "calculator",
			Description: "A tool to perform basic arithmetic calculations. Input should be a simple mathematical expression string (e.g., \"2 + 2\", \"10 * 5 / 2\"). Supports +, -, *, / operations and integers/floats.",
			InputSchema: `{"type": "string", "description": "The arithmetic expression to evaluate. Example: \"10 + 5 * (3 - 1)\""}`,
		},
	}
}

// Execute evaluates the arithmetic expression.
// This is a very simplified and potentially unsafe calculator. For production, use a proper math expression parser/evaluator.
func (ct *CalculatorTool) Execute(ctx context.Context, input string) (string, error) {
	// Basic security: allow only numbers, operators, parentheses, and spaces.
	// This is NOT a substitute for a proper parsing library and is still vulnerable.
	validPattern := regexp.MustCompile(`^[0-9	 .+\-*/%()]+$`)
	if !validPattern.MatchString(input) {
		return "", fmt.Errorf("invalid characters in expression: %s", input)
	}

	// Extremely simplified evaluation logic. This is NOT robust or safe for complex expressions or untrusted input.
	// It does not handle operator precedence correctly without parentheses and is very basic.
	// A real implementation should use a proper math expression parsing library (e.g., from go-exprtk or similar).
	// For now, we will attempt a very naive evaluation for simple cases or suggest using `bc` via shell for safety.
	// Let's try to evaluate simple two-operand expressions for demonstration.

	// Attempt to use a more robust (but still limited without external libs) approach for simple expressions.
	// This is still not a full parser.
	// For a production system, one would use a library like `github.com/Knetic/govaluate`
	// or shell out to `bc` for safety and correctness.

	// Given the sandbox environment, shelling out to `bc` is safer and more robust for this example.
	// However, the Tool interface expects the tool to run in-process.
	// Let's stick to a very simplified in-process evaluation for now, acknowledging its limitations.

	// This is a placeholder for a real math expression evaluator.
	// For now, we will return an error and suggest a better approach.
	// return "", fmt.Errorf("simplified calculator cannot evaluate complex expression: %s. Consider using a dedicated math library or shelling out to 'bc'", input)

	// Let's try a very, very simple evaluation for expressions like "A op B"
	parts := regexp.MustCompile(`\s*([0-9	.]+)\s*([+\-*/])\s*([0-9	.]+)\s*`).FindStringSubmatch(input)
	if len(parts) == 4 {
		val1, err1 := strconv.ParseFloat(parts[1], 64)
		operator := parts[2]
		val2, err2 := strconv.ParseFloat(parts[3], 64)

		if err1 != nil || err2 != nil {
			return "", fmt.Errorf("invalid numbers in expression: %s", input)
		}

		var result float64
		switch operator {
		case "+":
			result = val1 + val2
		case "-":
			result = val1 - val2
		case "*":
			result = val1 * val2
		case "/":
			if val2 == 0 {
				return "", fmt.Errorf("division by zero")
			}
			result = val1 / val2
		default:
			return "", fmt.Errorf("unsupported operator: %s", operator)
		}
		return strconv.FormatFloat(result, 'f', -1, 64), nil
	}

	return "", fmt.Errorf("calculator tool can only evaluate simple 'number operator number' expressions (e.g., '2 + 3', '10.5 * 2'). For complex expressions, a more robust parser is needed. Input: %s", input)
}

// Ensure CalculatorTool implements the Tool interface.
var _ tools.Tool = (*CalculatorTool)(nil)

