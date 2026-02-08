package tool

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// MCPServerInfo describes an MCP server and the tools it provides.
type MCPServerInfo struct {
	// Name is the human-readable name of the MCP server.
	Name string

	// URL is the server's endpoint URL.
	URL string

	// Tools lists the tool definitions exposed by this server.
	Tools []schema.ToolDefinition

	// Transport is the transport protocol (e.g., "streamable-http").
	Transport string
}

// MCPRegistry provides discovery of MCP servers. Implementations may search
// a local directory, a remote registry service, or a static configuration.
type MCPRegistry interface {
	// Search finds MCP servers matching the given query string.
	Search(ctx context.Context, query string) ([]MCPServerInfo, error)

	// Discover returns all known MCP servers.
	Discover(ctx context.Context) ([]MCPServerInfo, error)
}

// StaticMCPRegistry is an MCPRegistry backed by a static list of servers.
type StaticMCPRegistry struct {
	servers []MCPServerInfo
}

// NewStaticMCPRegistry creates an MCPRegistry from a fixed list of servers.
func NewStaticMCPRegistry(servers ...MCPServerInfo) *StaticMCPRegistry {
	return &StaticMCPRegistry{servers: servers}
}

// Search filters servers whose name contains the query substring.
func (r *StaticMCPRegistry) Search(_ context.Context, query string) ([]MCPServerInfo, error) {
	var matched []MCPServerInfo
	for _, s := range r.servers {
		if containsCI(s.Name, query) {
			matched = append(matched, s)
		}
	}
	return matched, nil
}

// Discover returns all servers in the registry.
func (r *StaticMCPRegistry) Discover(_ context.Context) ([]MCPServerInfo, error) {
	result := make([]MCPServerInfo, len(r.servers))
	copy(result, r.servers)
	return result, nil
}

// containsCI performs a case-insensitive substring match.
func containsCI(s, substr string) bool {
	sLower := toLower(s)
	subLower := toLower(substr)
	return len(subLower) == 0 || contains(sLower, subLower)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		} else {
			b[i] = c
		}
	}
	return string(b)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
