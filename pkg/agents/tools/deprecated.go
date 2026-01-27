// Package tools provides backward-compatible shims for the tools packages.
// All types and functions in this file are deprecated and will be removed in v2.0.
// Please update your imports to use the new package locations.
//
// Migration guide:
//   - github.com/lookatitude/beluga-ai/pkg/agents/tools -> github.com/lookatitude/beluga-ai/pkg/tools
//   - github.com/lookatitude/beluga-ai/pkg/agents/tools/api -> github.com/lookatitude/beluga-ai/pkg/tools/api
//   - github.com/lookatitude/beluga-ai/pkg/agents/tools/mcp -> github.com/lookatitude/beluga-ai/pkg/tools/mcp
//   - github.com/lookatitude/beluga-ai/pkg/agents/tools/gofunc -> github.com/lookatitude/beluga-ai/pkg/tools/gofunc
//   - github.com/lookatitude/beluga-ai/pkg/agents/tools/shell -> github.com/lookatitude/beluga-ai/pkg/tools/shell
//   - github.com/lookatitude/beluga-ai/pkg/agents/tools/providers -> github.com/lookatitude/beluga-ai/pkg/tools/providers
package tools

// NOTE: This package still contains the original implementations for backward compatibility.
// The new pkg/tools package contains copies of the same code with updated import paths.
// Both packages will work identically until pkg/agents/tools is removed in v2.0.
//
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/tools instead.
