// Package server provides configuration structures for server implementations.
package server

import (
	"github.com/lookatitude/beluga-ai/pkg/server/iface"
)

// Re-export types from iface to maintain backward compatibility.
type (
	Config           = iface.Config
	RESTConfig       = iface.RESTConfig
	MCPConfig        = iface.MCPConfig
	Option           = iface.Option
	Middleware       = iface.Middleware
	Logger           = iface.Logger
	Tracer           = iface.Tracer
	Span             = iface.Span
	Meter            = iface.Meter
	Int64Counter     = iface.Int64Counter
	Float64Histogram = iface.Float64Histogram
	MCPTool          = iface.MCPTool
	MCPResource      = iface.MCPResource
	StreamingHandler = iface.StreamingHandler
	Server           = iface.Server
	RESTServer       = iface.RESTServer
	MCPServer        = iface.MCPServer
)

// Re-export functions from iface.
var (
	WithConfig               = iface.WithConfig
	WithRESTConfig           = iface.WithRESTConfig
	WithMCPConfig            = iface.WithMCPConfig
	WithLogger               = iface.WithLogger
	WithTracer               = iface.WithTracer
	WithMeter                = iface.WithMeter
	WithMiddleware           = iface.WithMiddleware
	WithMCPTool     = iface.WithMCPTool
	WithMCPResource = iface.WithMCPResource
)
