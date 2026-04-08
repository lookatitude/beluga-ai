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
package computeruse
