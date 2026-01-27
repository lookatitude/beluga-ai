// Package providers provides convenience functions for creating tool providers.
// It re-exports functionality from specific provider packages for backward compatibility.
package providers

import (
	"github.com/lookatitude/beluga-ai/pkg/tools"
	"github.com/lookatitude/beluga-ai/pkg/tools/providers/calculator"
	"github.com/lookatitude/beluga-ai/pkg/tools/providers/echo"
)

// NewCalculatorTool creates a new CalculatorTool with the given configuration.
// This is a convenience function that wraps calculator.NewCalculatorToolWithConfig.
func NewCalculatorTool(cfg tools.ToolConfig) (*calculator.CalculatorTool, error) {
	return calculator.NewCalculatorToolWithConfig(cfg)
}

// NewEchoTool creates a new EchoTool with the given configuration.
// This is a convenience function that wraps echo.NewEchoToolWithConfig.
func NewEchoTool(cfg tools.ToolConfig) (*echo.EchoTool, error) {
	return echo.NewEchoToolWithConfig(cfg)
}
