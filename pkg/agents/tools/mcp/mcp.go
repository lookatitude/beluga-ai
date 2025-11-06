// Package mcp provides tools for interacting with Minecraft servers.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Tnze/go-mc/bot"
	"github.com/jltobler/go-rcon"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// MCPingTool pings a Minecraft server to get its status.
type MCPingTool struct {
	tools.BaseTool // Embed BaseTool for default Batch implementation
	toolName       string
}

// NewMCPingTool creates a new MCPingTool.
func NewMCPingTool() *MCPingTool {
	return &MCPingTool{
		toolName: "minecraft_server_ping",
	}
}

// Definition returns the tool's definition.
func (t *MCPingTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        t.toolName,
		Description: "Pings a Minecraft server (Java Edition) to get its status including version, player count, and MOTD. Input should be the server address (e.g., \"mc.example.com:25565\" or just \"mc.example.com\").",
		InputSchema: map[string]any{"type": "string"}, // Expects a string address
	}
}

// Description returns the tool's description.
func (t *MCPingTool) Description() string {
	return t.Definition().Description
}

// Name returns the tool's name.
func (t *MCPingTool) Name() string {
	return t.toolName
}

// Execute pings the server.
// Input: any - expects string server address (host:port or host)
// Output: string - JSON representation of the server status
func (t *MCPingTool) Execute(ctx context.Context, input any) (any, error) {
	address, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("invalid input type for %s: expected string (server address), got %T", t.Name(), input)
	}

	// go-mc ping function needs host and optionally port
	host := address
	port := 25565 // Default Minecraft port
	if strings.Contains(address, ":") {
		parts := strings.SplitN(address, ":", 2)
		host = parts[0]
		p, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid port in address \"%s\": %w", address, err)
		}
		port = p
	}

	// Add a timeout to the context for the ping operation
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // 10-second timeout for ping
	defer cancel()

	formattedAddress := fmt.Sprintf("%s:%d", host, port)
	resp, delay, err := bot.PingAndListContext(pingCtx, formattedAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to ping server %s: %w", address, err)
	}

	// Define a custom struct matching the Minecraft server JSON response format
	type ServerPingResponse struct {
		Version struct {
			Name     string `json:"name"`
			Protocol int    `json:"protocol"`
		} `json:"version"`
		Players struct {
			Max    int `json:"max"`
			Online int `json:"online"`
			// Sample array of players omitted for simplicity
		} `json:"players"`
		Description interface{} `json:"description"` // Can be string or complex chat component
	}

	// Process the response (which is JSON) into our structured format
	var status ServerPingResponse
	err = json.Unmarshal(resp, &status)
	if err != nil {
		// Fallback: return raw JSON if unmarshaling fails
		fmt.Printf("Warning: Failed to unmarshal ping response JSON for %s: %v. Returning raw JSON.\n", address, err)
		return string(resp), nil
	}

	// Extract MOTD (Message of the Day) from the description field
	// The description can be either a simple string or a complex chat component
	var motd string
	switch desc := status.Description.(type) {
	case string:
		motd = desc
	case map[string]interface{}:
		// Try to extract text from chat component
		if text, ok := desc["text"].(string); ok {
			motd = text
		} else {
			// Convert the whole component to JSON as fallback
			descJSON, _ := json.Marshal(desc)
			motd = string(descJSON)
		}
	default:
		// If we can't determine the type, stringify it
		descJSON, _ := json.Marshal(desc)
		motd = string(descJSON)
	}

	result := map[string]any{
		"version":        status.Version.Name,
		"protocol":       status.Version.Protocol,
		"motd":           motd,
		"players_online": status.Players.Online,
		"players_max":    status.Players.Max,
		"latency_ms":     delay.Milliseconds(),
	}

	// Convert map to JSON string for consistent tool output
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ping result to JSON: %w", err)
	}

	return string(resultJSON), nil
}

// Ensure MCPingTool implements the interface.
var _ tools.Tool = (*MCPingTool)(nil)

// --- RCON Tool ---

// MCRconTool executes commands on a Minecraft server via RCON.
type MCRconTool struct {
	tools.BaseTool // Embed BaseTool for default Batch implementation
	toolName       string
}

// NewMCRconTool creates a new MCRconTool.
func NewMCRconTool() *MCRconTool {
	return &MCRconTool{
		toolName: "minecraft_rcon_command",
	}
}

// Definition returns the tool's definition.
func (t *MCRconTool) Definition() tools.ToolDefinition {
	return tools.ToolDefinition{
		Name:        t.toolName,
		Description: "Executes a command on a Minecraft server (Java Edition) using RCON. Input must be a JSON string or map[string]any with keys: \"address\" (string, e.g., \"mc.example.com:25575\"), \"password\" (string), and \"command\" (string).",
		InputSchema: map[string]any{
			"type": "object", // Can be string (JSON) or map
			"properties": map[string]any{
				"address":  map[string]any{"type": "string"},
				"password": map[string]any{"type": "string"},
				"command":  map[string]any{"type": "string"},
			},
			"required": []string{"address", "password", "command"},
		},
	}
}

// Description returns the tool's description.
func (t *MCRconTool) Description() string {
	return t.Definition().Description
}

// Name returns the tool's name.
func (t *MCRconTool) Name() string {
	return t.toolName
}

// RconInput defines the structure for the JSON input.
type RconInput struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	Command  string `json:"command"`
}

// Execute connects via RCON and runs the command.
// Input: any - expects JSON string or map[string]any matching RconInput structure
// Output: string - Response from the server command
func (t *MCRconTool) Execute(ctx context.Context, input any) (any, error) {
	var rconInput RconInput
	var err error

	inputStr, isStr := input.(string)
	inputMap, isMap := input.(map[string]any)

	if isStr {
		err = json.Unmarshal([]byte(inputStr), &rconInput)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON string input for %s: %w. Input: %s", t.Name(), err, inputStr)
		}
	} else if isMap {
		// Convert map to JSON then unmarshal (simple way to handle map)
		jsonBytes, jsonErr := json.Marshal(inputMap)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to marshal map input for %s: %w", t.Name(), jsonErr)
		}
		err = json.Unmarshal(jsonBytes, &rconInput)
		if err != nil {
			// Should not happen if marshal succeeded, but check anyway
			return nil, fmt.Errorf("failed to unmarshal map input after marshaling for %s: %w", t.Name(), err)
		}
	} else {
		return nil, fmt.Errorf("invalid input type for %s: expected JSON string or map[string]any, got %T", t.Name(), input)
	}

	if rconInput.Address == "" || rconInput.Password == "" || rconInput.Command == "" {
		return nil, errors.New("invalid input for minecraft_rcon_command: 'address', 'password', and 'command' keys are required")
	}

	// Add a timeout to the context for the RCON operation
	// rconCtx, cancel := context.WithTimeout(ctx, 15*time.Second) // Use parent context for now
	// defer cancel()

	conn, err := rcon.Dial(rconInput.Address, rconInput.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to connect via RCON to %s: %w", rconInput.Address, err)
	}
	defer conn.Close()

	response, err := conn.SendCommand(rconInput.Command)
	if err != nil {
		return nil, fmt.Errorf("failed to send RCON command to %s: %w", rconInput.Address, err)
	}

	response = StripMinecraftFormatting(response)
	return response, nil
}

// Ensure MCRconTool implements the interface.
var _ tools.Tool = (*MCRconTool)(nil)

// StripMinecraftFormatting removes Minecraft color and formatting codes (§[0-9a-fk-or]).
func StripMinecraftFormatting(input string) string {
	var result strings.Builder
	result.Grow(len(input))
	strip := false
	for _, r := range input {
		if r == '§' {
			strip = true
			continue
		}
		if strip {
			strip = false // Skip the character after §
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
