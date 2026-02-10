package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/protocol/mcp"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// ServerEntry describes a registered MCP server.
type ServerEntry struct {
	// Name is a human-readable name for this server.
	Name string
	// URL is the MCP server endpoint URL.
	URL string
	// Tags are optional labels for filtering servers.
	Tags []string
}

// DiscoveredTool wraps a tool with metadata about which MCP server provides it.
type DiscoveredTool struct {
	// Tool is the native tool.Tool instance.
	Tool tool.Tool
	// ServerName is the name of the MCP server that provides this tool.
	ServerName string
	// ServerURL is the URL of the MCP server.
	ServerURL string
}

// Registry manages MCP server discovery and tool aggregation. It maintains
// a list of known MCP servers and provides methods to discover and instantiate
// their tools as native tool.Tool instances. Safe for concurrent use.
type Registry struct {
	mu      sync.RWMutex
	servers []ServerEntry
	// clientFactory creates an MCP client for a given URL. Used for testing.
	clientFactory func(url string) MCPClientInterface
}

// MCPClientInterface abstracts the MCP client for testing.
type MCPClientInterface interface {
	Initialize(ctx context.Context) (*mcp.ServerCapabilities, error)
	ListTools(ctx context.Context) ([]mcp.ToolInfo, error)
	CallTool(ctx context.Context, name string, args map[string]any) (*mcp.ToolCallResult, error)
}

// New creates a new MCP Registry.
func New() *Registry {
	return &Registry{
		clientFactory: func(url string) MCPClientInterface {
			return mcp.NewClient(url)
		},
	}
}

// AddServer registers an MCP server with the registry.
func (r *Registry) AddServer(name, url string, tags ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.servers = append(r.servers, ServerEntry{
		Name: name,
		URL:  url,
		Tags: tags,
	})
}

// RemoveServer removes a server from the registry by name.
func (r *Registry) RemoveServer(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	filtered := r.servers[:0]
	for _, s := range r.servers {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}
	r.servers = filtered
}

// Servers returns a copy of all registered server entries.
func (r *Registry) Servers() []ServerEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ServerEntry, len(r.servers))
	copy(result, r.servers)
	return result
}

// ServersByTag returns servers that have at least one matching tag.
func (r *Registry) ServersByTag(tag string) []ServerEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []ServerEntry
	for _, s := range r.servers {
		for _, t := range s.Tags {
			if t == tag {
				result = append(result, s)
				break
			}
		}
	}
	return result
}

// DiscoverTools connects to all registered MCP servers, initializes them,
// and returns all available tools as native tool.Tool instances.
func (r *Registry) DiscoverTools(ctx context.Context) ([]DiscoveredTool, error) {
	r.mu.RLock()
	servers := make([]ServerEntry, len(r.servers))
	copy(servers, r.servers)
	r.mu.RUnlock()

	var allTools []DiscoveredTool
	var errs []error

	for _, srv := range servers {
		tools, err := r.discoverFromServer(ctx, srv)
		if err != nil {
			errs = append(errs, fmt.Errorf("server %q (%s): %w", srv.Name, srv.URL, err))
			continue
		}
		allTools = append(allTools, tools...)
	}

	if len(errs) > 0 && len(allTools) == 0 {
		return nil, fmt.Errorf("mcp/registry: all servers failed: %v", errs)
	}

	return allTools, nil
}

// DiscoverToolsFromServer connects to a specific server by name and returns
// its tools.
func (r *Registry) DiscoverToolsFromServer(ctx context.Context, serverName string) ([]DiscoveredTool, error) {
	r.mu.RLock()
	var srv *ServerEntry
	for _, s := range r.servers {
		if s.Name == serverName {
			cp := s
			srv = &cp
			break
		}
	}
	r.mu.RUnlock()

	if srv == nil {
		return nil, fmt.Errorf("mcp/registry: server %q not found", serverName)
	}

	return r.discoverFromServer(ctx, *srv)
}

// Tools is a convenience method that returns just the tool.Tool instances
// without server metadata.
func (r *Registry) Tools(ctx context.Context) ([]tool.Tool, error) {
	discovered, err := r.DiscoverTools(ctx)
	if err != nil {
		return nil, err
	}

	tools := make([]tool.Tool, len(discovered))
	for i, dt := range discovered {
		tools[i] = dt.Tool
	}
	return tools, nil
}

func (r *Registry) discoverFromServer(ctx context.Context, srv ServerEntry) ([]DiscoveredTool, error) {
	client := r.clientFactory(srv.URL)

	if _, err := client.Initialize(ctx); err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	infos, err := client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	tools := make([]DiscoveredTool, len(infos))
	for i, info := range infos {
		tools[i] = DiscoveredTool{
			Tool:       newRemoteTool(client, info),
			ServerName: srv.Name,
			ServerURL:  srv.URL,
		}
	}
	return tools, nil
}

// remoteTool wraps an MCP remote tool as a native tool.Tool.
type remoteTool struct {
	client MCPClientInterface
	info   mcp.ToolInfo
}

func newRemoteTool(client MCPClientInterface, info mcp.ToolInfo) *remoteTool {
	return &remoteTool{client: client, info: info}
}

func (t *remoteTool) Name() string              { return t.info.Name }
func (t *remoteTool) Description() string        { return t.info.Description }
func (t *remoteTool) InputSchema() map[string]any { return t.info.InputSchema }

func (t *remoteTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	result, err := t.client.CallTool(ctx, t.info.Name, input)
	if err != nil {
		return nil, fmt.Errorf("mcp/registry/execute: %w", err)
	}

	// Convert MCP content items to native ContentParts.
	var content []schema.ContentPart
	for _, item := range result.Content {
		if item.Type == "text" {
			content = append(content, schema.TextPart{Text: item.Text})
		}
	}

	return &tool.Result{
		Content: content,
		IsError: result.IsError,
	}, nil
}
