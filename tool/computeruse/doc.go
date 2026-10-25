// Package computeruse provides computer use and browser automation
// capabilities for AI agents.
//
// It defines interfaces for computer actions (screenshot, click, type, scroll)
// and browser backends, with safety guards for URL allowlisting and action
// rate limiting. A ScreenAnalyzer uses multimodal LLMs to describe
// screenshots, enabling vision-based agent interaction.
//
// Key types:
//   - ComputerAction: Interface for screen interactions
//   - BrowserBackend: Interface for browser automation
//   - ScreenAnalyzer: Multimodal LLM-based screenshot analysis
//   - ComputerUseTool: Implements tool.Tool for agent integration
//   - SafetyGuard: URL allowlisting and action rate limiting
//
// # Layering note
//
// ScreenAnalyzer depends on package llm. The package dependency map lists tool/ as
// depending only on core, schema and o11y, so this sideways layer-3 import is
// a known exception. It is kept to avoid breaking the exported API and is
// scheduled to move out of tool/ in the next major version.
package computeruse
