package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestNewBaseTool(t *testing.T) {
	name := "test_tool"
	description := "A test tool"
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type": "string",
			},
		},
	}

	tool := NewBaseTool(name, description, schema)

	if tool.GetName() != name {
		t.Errorf("Expected tool name to be %q, got %q", name, tool.GetName())
	}

	if tool.GetDescription() != description {
		t.Errorf("Expected tool description to be %q, got %q", description, tool.GetDescription())
	}

	toolSchema := tool.GetInputSchema()
	if toolSchema == nil {
		t.Error("Expected non-nil input schema")
	}
}

func TestBaseTool_GetName(t *testing.T) {
	tool := NewBaseTool("test_tool", "description", nil)

	if tool.GetName() != "test_tool" {
		t.Errorf("Expected name to be 'test_tool', got %q", tool.GetName())
	}
}

func TestBaseTool_GetDescription(t *testing.T) {
	tool := NewBaseTool("test", "Test description", nil)

	if tool.GetDescription() != "Test description" {
		t.Errorf("Expected description to be 'Test description', got %q", tool.GetDescription())
	}
}

func TestBaseTool_GetInputSchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "string",
	}
	tool := NewBaseTool("test", "desc", schema)

	result := tool.GetInputSchema()
	if result["type"] != "string" {
		t.Errorf("Expected schema type to be 'string', got %v", result["type"])
	}
}

func TestBaseTool_GetInputSchemaString(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
		},
	}
	tool := NewBaseTool("test", "desc", schema)

	schemaStr, err := tool.GetInputSchemaString()
	if err != nil {
		t.Errorf("GetInputSchemaString() error = %v", err)
		return
	}

	// Try to parse the JSON to ensure it's valid
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(schemaStr), &parsed); err != nil {
		t.Errorf("GetInputSchemaString() returned invalid JSON: %v", err)
	}
}

func TestBaseTool_GetInputSchemaString_NilSchema(t *testing.T) {
	tool := NewBaseTool("test", "desc", nil)

	schemaStr, err := tool.GetInputSchemaString()
	if err != nil {
		t.Errorf("GetInputSchemaString() with nil schema error = %v", err)
		return
	}

	expected := "{}"
	if schemaStr != expected {
		t.Errorf("Expected schema string to be %q, got %q", expected, schemaStr)
	}
}

func TestBaseTool_Execute(t *testing.T) {
	tool := NewBaseTool("test", "desc", nil)

	_, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Error("Expected BaseTool.Execute to return an error (not implemented)")
	}
}

func TestToolAgentAction_Struct(t *testing.T) {
	action := ToolAgentAction{
		ToolName:  "test_tool",
		ToolInput: map[string]interface{}{"key": "value"},
		Log:       "Test log",
	}

	if action.ToolName != "test_tool" {
		t.Errorf("Expected ToolName to be 'test_tool', got %q", action.ToolName)
	}

	if action.Log != "Test log" {
		t.Errorf("Expected Log to be 'Test log', got %q", action.Log)
	}

	input, ok := action.ToolInput.(map[string]interface{})
	if !ok {
		t.Error("Expected ToolInput to be map[string]interface{}")
		return
	}

	if input["key"] != "value" {
		t.Errorf("Expected ToolInput['key'] to be 'value', got %v", input["key"])
	}
}
