package schema

import "testing"

func TestToolCall_Fields(t *testing.T) {
	tests := []struct {
		name      string
		tc        ToolCall
		wantID    string
		wantName  string
		wantArgs  string
	}{
		{
			name:     "fully_populated",
			tc:       ToolCall{ID: "call-123", Name: "search", Arguments: `{"query":"test"}`},
			wantID:   "call-123",
			wantName: "search",
			wantArgs: `{"query":"test"}`,
		},
		{
			name:     "empty_arguments",
			tc:       ToolCall{ID: "call-456", Name: "get_time", Arguments: ""},
			wantID:   "call-456",
			wantName: "get_time",
			wantArgs: "",
		},
		{
			name:     "complex_arguments",
			tc:       ToolCall{ID: "tc-789", Name: "calculate", Arguments: `{"a":1,"b":2,"op":"add"}`},
			wantID:   "tc-789",
			wantName: "calculate",
			wantArgs: `{"a":1,"b":2,"op":"add"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tc.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", tt.tc.ID, tt.wantID)
			}
			if tt.tc.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", tt.tc.Name, tt.wantName)
			}
			if tt.tc.Arguments != tt.wantArgs {
				t.Errorf("Arguments = %q, want %q", tt.tc.Arguments, tt.wantArgs)
			}
		})
	}
}

func TestToolCall_ZeroValue(t *testing.T) {
	var tc ToolCall
	if tc.ID != "" {
		t.Errorf("zero ID = %q, want empty", tc.ID)
	}
	if tc.Name != "" {
		t.Errorf("zero Name = %q, want empty", tc.Name)
	}
	if tc.Arguments != "" {
		t.Errorf("zero Arguments = %q, want empty", tc.Arguments)
	}
}

func TestToolResult_Fields(t *testing.T) {
	tests := []struct {
		name        string
		tr          ToolResult
		wantCallID  string
		wantIsError bool
		wantParts   int
	}{
		{
			name: "success_result",
			tr: ToolResult{
				CallID:  "call-123",
				Content: []ContentPart{TextPart{Text: "result data"}},
				IsError: false,
			},
			wantCallID:  "call-123",
			wantIsError: false,
			wantParts:   1,
		},
		{
			name: "error_result",
			tr: ToolResult{
				CallID:  "call-456",
				Content: []ContentPart{TextPart{Text: "tool execution failed"}},
				IsError: true,
			},
			wantCallID:  "call-456",
			wantIsError: true,
			wantParts:   1,
		},
		{
			name: "multiple_content_parts",
			tr: ToolResult{
				CallID: "call-789",
				Content: []ContentPart{
					TextPart{Text: "result summary"},
					ImagePart{URL: "http://example.com/chart.png"},
				},
				IsError: false,
			},
			wantCallID:  "call-789",
			wantIsError: false,
			wantParts:   2,
		},
		{
			name: "empty_content",
			tr: ToolResult{
				CallID:  "call-empty",
				Content: nil,
				IsError: false,
			},
			wantCallID:  "call-empty",
			wantIsError: false,
			wantParts:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tr.CallID != tt.wantCallID {
				t.Errorf("CallID = %q, want %q", tt.tr.CallID, tt.wantCallID)
			}
			if tt.tr.IsError != tt.wantIsError {
				t.Errorf("IsError = %v, want %v", tt.tr.IsError, tt.wantIsError)
			}
			if len(tt.tr.Content) != tt.wantParts {
				t.Errorf("len(Content) = %d, want %d", len(tt.tr.Content), tt.wantParts)
			}
		})
	}
}

func TestToolResult_ZeroValue(t *testing.T) {
	var tr ToolResult
	if tr.CallID != "" {
		t.Errorf("zero CallID = %q, want empty", tr.CallID)
	}
	if tr.Content != nil {
		t.Errorf("zero Content = %v, want nil", tr.Content)
	}
	if tr.IsError {
		t.Error("zero IsError = true, want false")
	}
}

func TestToolResult_ContentTypes(t *testing.T) {
	tr := ToolResult{
		CallID: "call-multi",
		Content: []ContentPart{
			TextPart{Text: "text result"},
			ImagePart{URL: "http://example.com/img.png", MimeType: "image/png"},
			FilePart{Name: "data.csv", MimeType: "text/csv"},
		},
	}

	expected := []ContentType{ContentText, ContentImage, ContentFile}
	for i, part := range tr.Content {
		if got := part.PartType(); got != expected[i] {
			t.Errorf("Content[%d].PartType() = %q, want %q", i, got, expected[i])
		}
	}
}

func TestToolDefinition_Fields(t *testing.T) {
	tests := []struct {
		name        string
		td          ToolDefinition
		wantName    string
		wantDesc    string
		wantSchema  bool
	}{
		{
			name: "fully_populated",
			td: ToolDefinition{
				Name:        "search",
				Description: "Search the web for information",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": map[string]any{
							"type":        "string",
							"description": "The search query",
						},
					},
					"required": []any{"query"},
				},
			},
			wantName:   "search",
			wantDesc:   "Search the web for information",
			wantSchema: true,
		},
		{
			name: "no_schema",
			td: ToolDefinition{
				Name:        "get_time",
				Description: "Get the current time",
				InputSchema: nil,
			},
			wantName:   "get_time",
			wantDesc:   "Get the current time",
			wantSchema: false,
		},
		{
			name: "empty_schema",
			td: ToolDefinition{
				Name:        "noop",
				Description: "Does nothing",
				InputSchema: map[string]any{},
			},
			wantName:   "noop",
			wantDesc:   "Does nothing",
			wantSchema: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.td.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", tt.td.Name, tt.wantName)
			}
			if tt.td.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", tt.td.Description, tt.wantDesc)
			}
			hasSchema := tt.td.InputSchema != nil
			if hasSchema != tt.wantSchema {
				t.Errorf("has InputSchema = %v, want %v", hasSchema, tt.wantSchema)
			}
		})
	}
}

func TestToolDefinition_ZeroValue(t *testing.T) {
	var td ToolDefinition
	if td.Name != "" {
		t.Errorf("zero Name = %q, want empty", td.Name)
	}
	if td.Description != "" {
		t.Errorf("zero Description = %q, want empty", td.Description)
	}
	if td.InputSchema != nil {
		t.Errorf("zero InputSchema = %v, want nil", td.InputSchema)
	}
}

func TestToolDefinition_SchemaAccess(t *testing.T) {
	td := ToolDefinition{
		Name: "calculate",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"expression": map[string]any{
					"type": "string",
				},
			},
		},
	}

	schemaType, ok := td.InputSchema["type"].(string)
	if !ok || schemaType != "object" {
		t.Errorf("InputSchema[\"type\"] = %v, want %q", td.InputSchema["type"], "object")
	}

	props, ok := td.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("InputSchema[\"properties\"] is not map[string]any")
	}
	if _, ok := props["expression"]; !ok {
		t.Error("InputSchema missing 'expression' property")
	}
}
