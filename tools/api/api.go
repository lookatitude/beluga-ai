// Package api provides a tool implementation for making HTTP requests to external APIs.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/tools"
)

// APITool allows making HTTP requests to external APIs.
type APITool struct {
	tools.BaseTool
	Def     tools.ToolDefinition // Store definition directly
	Client  *http.Client         // Use a shared client for potential connection reuse
	Timeout time.Duration        // Optional timeout per request
}

// APIToolInput defines the expected structure for the input arguments.
// Using a struct helps with clarity and potential schema generation.
type APIToolInput struct {
	URL     string            `json:"url"`               // Required: The URL to request
	Method  string            `json:"method,omitempty"`  // Optional: HTTP method (GET, POST, etc.), defaults to GET
	Headers map[string]string `json:"headers,omitempty"` // Optional: Request headers
	Body    any               `json:"body,omitempty"`    // Optional: Request body (can be string or JSON object)
}

// GenerateInputSchema generates the JSON schema for APIToolInput.
func GenerateInputSchema() (map[string]any, error) { // Changed return type
	// Basic schema generation (could be more sophisticated)
	schemaStr := `{
		"type": "object",
		"properties": {
			"url": {"type": "string", "description": "The URL to make the request to."},
			"method": {"type": "string", "description": "HTTP method (GET, POST, PUT, DELETE, etc.). Defaults to GET.", "default": "GET"},
			"headers": {"type": "object", "additionalProperties": {"type": "string"}, "description": "Key-value pairs for request headers."},
			"body": {"type": ["string", "object"], "description": "Request body. Can be a string or a JSON object."}
		},
		"required": ["url"]
	}`
	var schemaMap map[string]any
	err := json.Unmarshal([]byte(schemaStr), &schemaMap)
	return schemaMap, err
}

// NewAPITool creates a new APITool.
func NewAPITool(client *http.Client, timeout time.Duration) (*APITool, error) {
	inputSchema, err := GenerateInputSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to generate input schema: %w", err)
	}

	httpClient := client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &APITool{
		Def: tools.ToolDefinition{
			Name:        "api_request",
			Description: "Makes an HTTP request to a specified URL. Input must be a JSON object with 'url', and optional 'method', 'headers', 'body'.",
			InputSchema: inputSchema,
		},
		Client:  httpClient,
		Timeout: timeout,
	}, nil
}

// Definition returns the tool's definition.
func (at *APITool) Definition() tools.ToolDefinition {
	return at.Def
}

// Description returns the tool's description.
func (at *APITool) Description() string {
	return at.Definition().Description
}

// Name returns the tool's name.
func (at *APITool) Name() string {
	return at.Definition().Name
}

// Execute makes the HTTP request.
// Corrected input type to any and return type to any
func (at *APITool) Execute(ctx context.Context, input any) (any, error) {
	var apiInput APIToolInput

	// Handle input type: map[string]any or JSON string
	switch v := input.(type) {
	case map[string]any:
		// Marshal map back to JSON and unmarshal into struct for validation/typing
		argsJSON, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("internal error: failed to marshal input map: %w", err)
		}
		if err := json.Unmarshal(argsJSON, &apiInput); err != nil {
			return nil, fmt.Errorf("invalid input map format: %w. Expected keys: url (required), method, headers, body.", err)
		}
	case string:
		// Assume input is a JSON string
		if err := json.Unmarshal([]byte(v), &apiInput); err != nil {
			return nil, fmt.Errorf("invalid input JSON string format: %w. Expected keys: url (required), method, headers, body.", err)
		}
	default:
		return nil, fmt.Errorf("invalid input type: expected map[string]any or JSON string, got %T", input)
	}

	// Validate required fields
	if apiInput.URL == "" {
		return nil, fmt.Errorf("invalid input: 'url' is required")
	}

	// Default method to GET
	method := strings.ToUpper(apiInput.Method)
	if method == "" {
		method = http.MethodGet
	}

	// Prepare request body
	var reqBody io.Reader
	if apiInput.Body != nil {
		switch body := apiInput.Body.(type) {
		case string:
			reqBody = strings.NewReader(body)
		default:
			// Assume it's a JSON object, marshal it
			bodyBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			reqBody = bytes.NewReader(bodyBytes)
			// Ensure Content-Type is set if not provided
			if apiInput.Headers == nil {
				apiInput.Headers = make(map[string]string)
			}
			if _, found := apiInput.Headers["Content-Type"]; !found {
				apiInput.Headers["Content-Type"] = "application/json"
			}
		}
	}

	// Add timeout to context if specified
	reqCtx := ctx
	if at.Timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, at.Timeout)
		defer cancel()
	}

	// Create request
	req, err := http.NewRequestWithContext(reqCtx, method, apiInput.URL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range apiInput.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := at.Client.Do(req)
	if err != nil {
		// Check for context timeout
		if reqCtx.Err() == context.DeadlineExceeded {
			// Return timeout as output string, not error
			return fmt.Sprintf("Request timed out after %s", at.Timeout), nil
		}
		// Return network errors as output string, not error
		return fmt.Sprintf("Failed to execute request: %v", err), nil
	}
	defer resp.Body.Close()

	// Read response body
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// Return read errors as output string, not error
		return fmt.Sprintf("Failed to read response body (Status: %s): %v", resp.Status, err), nil
	}

	// Return status code and body as string
	// Consider truncating long responses?
	return fmt.Sprintf("Status: %s\nBody:\n%s", resp.Status, string(respBodyBytes)), nil
}

// Implement core.Runnable Invoke for APITool
func (at *APITool) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// Execute now takes any
	return at.Execute(ctx, input)
}

// Batch implementation
// Batch implements the tools.Tool interface
func (at *APITool) Batch(ctx context.Context, inputs []any) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := at.Execute(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("error processing batch item %d: %w", i, err)
		}
		results[i] = result
	}
	return results, nil
}

// Run implements the core.Runnable Batch method with options
func (at *APITool) Run(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	return at.Batch(ctx, inputs) // Options are ignored for now
}

// Stream is not applicable for standard API calls.
func (at *APITool) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	resultChan := make(chan any, 1)
	go func() {
		defer close(resultChan)
		output, err := at.Invoke(ctx, input, options...)
		if err != nil {
			resultChan <- err
		} else {
			resultChan <- output
		}
	}()
	return resultChan, nil
}

// Ensure implementation satisfies interfaces
// Make sure interfaces are correctly implemented
var _ tools.Tool = (*APITool)(nil)
// Define a custom interface that matches what we've implemented
type batcherWithOptions interface {
	Run(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
	Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error)
	Invoke(ctx context.Context, input any, options ...core.Option) (any, error)
}
var _ batcherWithOptions = (*APITool)(nil)
