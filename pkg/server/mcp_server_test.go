// Package server provides comprehensive tests for MCP server functionality.
// These tests focus on MCP protocol handling, tool and resource management.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// MCP Server Tests

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestMCPServerToolRegistration(t *testing.T) {
	logger := newMockLogger()
	server, err := NewMCPServer(
		WithLogger(logger),
		WithMCPConfig(MCPConfig{
			Config:     Config{Host: "localhost", Port: 0},
			ServerName: "test-mcp-server",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	mcpServer := server.(MCPServer)

	// Test registering tools
	tool1 := newMockMCPTool("calculator", "Performs calculations")
	tool2 := newMockMCPTool("text-analyzer", "Analyzes text")

	err = mcpServer.RegisterTool(tool1)
	if err != nil {
		t.Errorf("Failed to register tool1: %v", err)
	}

	err = mcpServer.RegisterTool(tool2)
	if err != nil {
		t.Errorf("Failed to register tool2: %v", err)
	}

	// Test registering duplicate tool (should fail)
	err = mcpServer.RegisterTool(tool1)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}

	// Test listing tools
	tools, err := mcpServer.ListTools(context.Background())
	if err != nil {
		t.Errorf("Failed to list tools: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestMCPServerResourceRegistration(t *testing.T) {
	logger := newMockLogger()
	server, err := NewMCPServer(
		WithLogger(logger),
		WithMCPConfig(MCPConfig{
			Config:     Config{Host: "localhost", Port: 0},
			ServerName: "test-mcp-server",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	mcpServer := server.(MCPServer)

	// Test registering resources
	resource1 := newMockMCPResource("file://config.json", "config", "Configuration file", "application/json", "{}")
	resource2 := newMockMCPResource("text://readme", "readme", "Project documentation", "text/plain", "# README")

	err = mcpServer.RegisterResource(resource1)
	if err != nil {
		t.Errorf("Failed to register resource1: %v", err)
	}

	err = mcpServer.RegisterResource(resource2)
	if err != nil {
		t.Errorf("Failed to register resource2: %v", err)
	}

	// Test registering duplicate resource (should fail)
	err = mcpServer.RegisterResource(resource1)
	if err == nil {
		t.Error("Expected error when registering duplicate resource")
	}

	// Test listing resources
	resources, err := mcpServer.ListResources(context.Background())
	if err != nil {
		t.Errorf("Failed to list resources: %v", err)
	}
	if len(resources) != 2 {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}
}

func TestMCPServerToolExecution(t *testing.T) {
	logger := newMockLogger()
	server, err := NewMCPServer(
		WithLogger(logger),
		WithMCPConfig(MCPConfig{
			Config:     Config{Host: "localhost", Port: 0},
			ServerName: "test-mcp-server",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	mcpServer := server.(MCPServer)

	// Register a tool
	tool := newMockMCPTool("test-tool", "A test tool")
	err = mcpServer.RegisterTool(tool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Test successful tool execution
	input := map[string]interface{}{
		"input": "test data",
		"value": 42,
	}

	result, err := mcpServer.CallTool(context.Background(), "test-tool", input)
	if err != nil {
		t.Errorf("Failed to execute tool: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Test tool execution with error
	tool.shouldError = true
	_, err = mcpServer.CallTool(context.Background(), "test-tool", input)
	if err == nil {
		t.Error("Expected error from tool execution")
	}

	// Test calling non-existent tool
	_, err = mcpServer.CallTool(context.Background(), "non-existent-tool", input)
	if err == nil {
		t.Error("Expected error when calling non-existent tool")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestMCPServerResourceReading(t *testing.T) {
	logger := newMockLogger()
	server, err := NewMCPServer(
		WithLogger(logger),
		WithMCPConfig(MCPConfig{
			Config:     Config{Host: "localhost", Port: 0},
			ServerName: "test-mcp-server",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	mcpServer := server.(MCPServer)

	// Register a resource
	resource := newMockMCPResource("test://resource", "test-resource", "A test resource", "text/plain", "test content")
	err = mcpServer.RegisterResource(resource)
	if err != nil {
		t.Fatalf("Failed to register resource: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Test successful resource reading
	// Note: MCP servers typically don't expose direct resource reading through the Server interface
	// This would be tested through the HTTP endpoints in integration tests

	// Test reading non-existent resource
	// This would also be tested through HTTP endpoints
}

func TestMCPServerInitialization(t *testing.T) {
	logger := newMockLogger()
	server, err := NewMCPServer(
		WithLogger(logger),
		WithMCPConfig(MCPConfig{
			Config:          Config{Host: "localhost", Port: 0},
			ServerName:      "test-server",
			ServerVersion:   "1.0.0",
			ProtocolVersion: "2024-11-05",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	// Test that the server implements the Server interface
	var _ iface.Server = server
	var _ MCPServer = server
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Test health check
	isHealthy := server.IsHealthy(context.Background())
	if !isHealthy {
		t.Error("Expected server to be healthy")
	}
}

// MCP Protocol Tests

func TestMCPProtocolMessages(t *testing.T) {
	tests := []struct {
		name    string
		message map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid_initialize",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "initialize",
				"params": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{},
					"clientInfo":      map[string]interface{}{"name": "test-client"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_list_tools",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      2,
				"method":  "tools/list",
			},
			wantErr: false,
		},
		{
			name: "valid_list_resources",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      3,
				"method":  "resources/list",
			},
			wantErr: false,
		},
		{
			name: "invalid_method",
			message: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      4,
				"method":  "invalid_method",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// In a real test, you would send these messages to the MCP server
			// and verify the responses. For now, we just validate the message structure.
			data, err := json.Marshal(tt.message)
			if err != nil {
				t.Errorf("Failed to marshal message: %v", err)
				return
			}

			var msg map[string]interface{}
			err = json.Unmarshal(data, &msg)
			if err != nil {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Errorf("Failed to unmarshal message: %v", err)
			}

			if _, hasMethod := msg["method"]; !hasMethod {
				t.Error("Message missing method field")
			}
		})
	}
}

// Integration-style MCP tests

func TestMCPServerFullIntegration(t *testing.T) {
	logger := newMockLogger()
	tracer := newMockTracer()

	server, err := NewMCPServer(
		WithLogger(logger),
		WithTracer(tracer),
		WithMCPConfig(MCPConfig{
			Config: Config{
				Host: "localhost",
				Port: 0, // Use random port for testing
			},
			ServerName:      "test-mcp-server",
			ServerVersion:   "1.0.0",
			ProtocolVersion: "2024-11-05",
		}),
	)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}

	mcpServer := server.(MCPServer)

	// Register multiple tools
	tools := []iface.MCPTool{
		newMockMCPTool("calculator", "Performs calculations"),
		newMockMCPTool("text-processor", "Processes text"),
		newMockMCPTool("data-analyzer", "Analyzes data"),
	}

	for _, tool := range tools {
		err := mcpServer.RegisterTool(tool)
		if err != nil {
			t.Errorf("Failed to register tool %s: %v", tool.Name(), err)
		}
	}

	// Register multiple resources
	resources := []iface.MCPResource{
		newMockMCPResource("file://config.json", "config", "Configuration", "application/json", "{}"),
		newMockMCPResource("text://readme.md", "readme", "Documentation", "text/markdown", "# README"),
		newMockMCPResource("data://dataset.csv", "dataset", "Data set", "text/csv", "col1,col2\n1,2\n3,4"),
	}

	for _, resource := range resources {
		err := mcpServer.RegisterResource(resource)
		if err != nil {
			t.Errorf("Failed to register resource %s: %v", resource.URI(), err)
		}
	}

	// Test listing all tools
	listedTools, err := mcpServer.ListTools(context.Background())
	if err != nil {
		t.Errorf("Failed to list tools: %v", err)
	}
	if len(listedTools) != len(tools) {
		t.Errorf("Expected %d tools, got %d", len(tools), len(listedTools))
	}

	// Test listing all resources
	listedResources, err := mcpServer.ListResources(context.Background())
	if err != nil {
		t.Errorf("Failed to list resources: %v", err)
	}
	if len(listedResources) != len(resources) {
		t.Errorf("Expected %d resources, got %d", len(resources), len(listedResources))
	}

	// Test tool execution for each tool
	for _, tool := range tools {
		input := map[string]interface{}{
			"input": "test",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			"value": 123,
		}
		result, err := mcpServer.CallTool(context.Background(), tool.Name(), input)
		if err != nil {
			t.Errorf("Failed to execute tool %s: %v", tool.Name(), err)
		}
		if result == nil {
			t.Errorf("Tool %s returned nil result", tool.Name())
		}
	}
}

// Benchmark tests for MCP server

func BenchmarkMCPServerToolRegistration(b *testing.B) {
	logger := newMockLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	server, _ := NewMCPServer(
		WithLogger(logger),
		WithMCPConfig(MCPConfig{
			Config:     Config{Host: "localhost", Port: 0},
			ServerName: "bench-mcp-server",
		}),
	)
	mcpServer := server.(MCPServer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool := newMockMCPTool(fmt.Sprintf("tool-%d", i), fmt.Sprintf("Tool %d", i))
		mcpServer.RegisterTool(tool)
	}
}

func BenchmarkMCPServerToolExecution(b *testing.B) {
	logger := newMockLogger()
	server, _ := NewMCPServer(
		WithLogger(logger),
		WithMCPConfig(MCPConfig{
			Config:     Config{Host: "localhost", Port: 0},
			ServerName: "bench-mcp-server",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		}),
	)
	mcpServer := server.(MCPServer)

	tool := newMockMCPTool("bench-tool", "Benchmark tool")
	mcpServer.RegisterTool(tool)

	input := map[string]interface{}{
		"input": "benchmark data",
		"value": 42,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mcpServer.CallTool(context.Background(), "bench-tool", input)
	}
}

func BenchmarkMCPServerListTools(b *testing.B) {
	logger := newMockLogger()
	server, _ := NewMCPServer(
		WithLogger(logger),
		WithMCPConfig(MCPConfig{
			Config:     Config{Host: "localhost", Port: 0},
			ServerName: "bench-mcp-server",
		}),
	)
	mcpServer := server.(MCPServer)

	// Register some tools
	for i := 0; i < 100; i++ {
		tool := newMockMCPTool(fmt.Sprintf("tool-%d", i), fmt.Sprintf("Tool %d", i))
		mcpServer.RegisterTool(tool)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mcpServer.ListTools(context.Background())
	}
}
