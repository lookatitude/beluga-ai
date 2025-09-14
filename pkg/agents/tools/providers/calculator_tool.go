package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

// CalculatorTool is a tool that can perform basic arithmetic calculations.
// It expects a map input with a key "expression" containing a string
// representing a simple arithmetic expression (e.g., "2 + 2", "10 * 5 / 2").
type CalculatorTool struct {
	tools.BaseTool
}

// NewCalculatorTool creates a new CalculatorTool.
func NewCalculatorTool(cfg iface.ToolConfig) (*CalculatorTool, error) {
	inputSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"expression": map[string]interface{}{
				"type":        "string",
				"description": "The arithmetic expression to evaluate. Example: \"10 + 5 * (3 - 1)\"",
			},
		},
		"required": []string{"expression"},
	}

	tool := &CalculatorTool{
		BaseTool: tools.BaseTool{},
	}
	tool.SetName(cfg.Name)
	tool.SetDescription(cfg.Description)
	tool.SetInputSchema(inputSchema)
	return tool, nil
}

// Execute evaluates the arithmetic expression from the input.
func (ct *CalculatorTool) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input must be a map[string]interface{}, got %T", input)
	}
	expression, ok := inputMap["expression"].(string)
	if !ok {
		inputBytes, err := json.Marshal(inputMap)
		if err != nil {
			return nil, fmt.Errorf("invalid input format for CalculatorTool: expected a map with a string field 'expression', but got something unmarshalable: %v", inputMap)
		}
		return nil, fmt.Errorf("invalid input format for CalculatorTool: expected a map with a string field 'expression', but got %s", string(inputBytes))
	}

	// Basic security: allow only numbers, operators, parentheses, and spaces.
	validPattern := regexp.MustCompile(`^[0-9\t .+\-*/%()]+$`)
	if !validPattern.MatchString(expression) {
		return nil, fmt.Errorf("invalid characters in expression: %s", expression)
	}

	// Simplified evaluation for expressions like "A op B"
	parts := regexp.MustCompile(`^\s*([0-9\.]+)\s*([+\-*/])\s*([0-9\.]+)\s*$`).FindStringSubmatch(expression)
	if len(parts) == 4 {
		val1, err1 := strconv.ParseFloat(parts[1], 64)
		operator := parts[2]
		val2, err2 := strconv.ParseFloat(parts[3], 64)

		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("invalid numbers in expression: %s. Error1: %v, Error2: %v", expression, err1, err2)
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
				return nil, fmt.Errorf("division by zero in expression: %s", expression)
			}
			result = val1 / val2
		default:
			return nil, fmt.Errorf("unsupported operator %s in expression: %s", operator, expression)
		}
		return strconv.FormatFloat(result, 'f', -1, 64), nil
	}

	return nil, fmt.Errorf("calculator tool can only evaluate simple 'number operator number' expressions (e.g., '2 + 3', '10.5 * 2'). For complex expressions, a more robust parser is needed. Input: %s", expression)
}

// Ensure CalculatorTool implements the Tool interface.
var _ tools.Tool = (*CalculatorTool)(nil)
