package tools

import (
	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
)

// Registry is an alias for iface.ToolRegistry for backward compatibility.
type Registry = iface.ToolRegistry

// InMemoryToolRegistry is an alias for iface.InMemoryToolRegistry for backward compatibility.
type InMemoryToolRegistry = iface.InMemoryToolRegistry

// NewInMemoryToolRegistry is an alias for iface.NewInMemoryToolRegistry for backward compatibility.
var NewInMemoryToolRegistry = iface.NewInMemoryToolRegistry
