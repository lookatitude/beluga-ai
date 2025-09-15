// Package iface provides shared types and interfaces for the server package.
// This package exists to break circular dependencies between the server and implementation packages.
package iface

import (
	"context"
)

// MCPTool represents a tool that can be invoked via MCP
type MCPTool interface {
	// Name returns the tool's name
	Name() string
	// Description returns the tool's description
	Description() string
	// InputSchema returns the JSON schema for the tool's input
	InputSchema() map[string]interface{}
	// Execute runs the tool with the given input
	Execute(ctx context.Context, input map[string]interface{}) (interface{}, error)
}

// MCPResource represents a resource that can be accessed via MCP
type MCPResource interface {
	// URI returns the resource's URI
	URI() string
	// Name returns the resource's name
	Name() string
	// Description returns the resource's description
	Description() string
	// MimeType returns the resource's MIME type
	MimeType() string
	// Read reads the resource content
	Read(ctx context.Context) ([]byte, error)
}

// Logger represents a logging interface
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Tracer represents a tracing interface
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

// Span represents a tracing span
type Span interface {
	End()
	SetAttributes(attrs ...interface{})
	RecordError(err error)
	SetStatus(code int, msg string)
}
