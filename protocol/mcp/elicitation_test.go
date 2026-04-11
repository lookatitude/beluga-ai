package mcp

import (
	"strings"
	"testing"
)

func TestValidateElicitationRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     ElicitationRequest
		wantErr string
	}{
		{
			name:    "empty message",
			req:     ElicitationRequest{Type: ElicitationText},
			wantErr: "message is required",
		},
		{
			name: "valid text",
			req:  ElicitationRequest{Type: ElicitationText, Message: "Enter name"},
		},
		{
			name: "valid confirm",
			req:  ElicitationRequest{Type: ElicitationConfirm, Message: "Are you sure?"},
		},
		{
			name:    "select without options",
			req:     ElicitationRequest{Type: ElicitationSelect, Message: "Pick one"},
			wantErr: "at least one option",
		},
		{
			name: "valid select",
			req:  ElicitationRequest{Type: ElicitationSelect, Message: "Pick one", Options: []string{"a", "b"}},
		},
		{
			name:    "form without fields",
			req:     ElicitationRequest{Type: ElicitationForm, Message: "Fill out"},
			wantErr: "at least one field",
		},
		{
			name: "form with empty field name",
			req: ElicitationRequest{
				Type:    ElicitationForm,
				Message: "Fill out",
				Fields:  []ElicitationField{{Name: "", Label: "test"}},
			},
			wantErr: "field name is required",
		},
		{
			name: "form with duplicate field names",
			req: ElicitationRequest{
				Type:    ElicitationForm,
				Message: "Fill out",
				Fields: []ElicitationField{
					{Name: "foo", Label: "Foo"},
					{Name: "foo", Label: "Foo2"},
				},
			},
			wantErr: "duplicate field name",
		},
		{
			name: "valid form",
			req: ElicitationRequest{
				Type:    ElicitationForm,
				Message: "Fill out",
				Fields: []ElicitationField{
					{Name: "name", Label: "Name", Type: ElicitationText, Required: true},
					{Name: "color", Label: "Color", Type: ElicitationSelect, Options: []string{"red", "blue"}},
				},
			},
		},
		{
			name:    "unknown type",
			req:     ElicitationRequest{Type: "unknown", Message: "test"},
			wantErr: "unknown type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateElicitationRequest(tt.req)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateElicitationResponse(t *testing.T) {
	tests := []struct {
		name    string
		req     ElicitationRequest
		resp    ElicitationResponse
		wantErr string
	}{
		{
			name: "rejected response always valid",
			req:  ElicitationRequest{Type: ElicitationSelect, Message: "Pick", Options: []string{"a"}},
			resp: ElicitationResponse{Accepted: false},
		},
		{
			name:    "select missing selection",
			req:     ElicitationRequest{Type: ElicitationSelect, Message: "Pick", Options: []string{"a", "b"}},
			resp:    ElicitationResponse{Accepted: true},
			wantErr: "requires a selected value",
		},
		{
			name:    "select invalid option",
			req:     ElicitationRequest{Type: ElicitationSelect, Message: "Pick", Options: []string{"a", "b"}},
			resp:    ElicitationResponse{Accepted: true, Selected: "c"},
			wantErr: "not among options",
		},
		{
			name: "select valid",
			req:  ElicitationRequest{Type: ElicitationSelect, Message: "Pick", Options: []string{"a", "b"}},
			resp: ElicitationResponse{Accepted: true, Selected: "a"},
		},
		{
			name: "form missing values",
			req: ElicitationRequest{
				Type:    ElicitationForm,
				Message: "Fill",
				Fields:  []ElicitationField{{Name: "name", Required: true}},
			},
			resp:    ElicitationResponse{Accepted: true},
			wantErr: "form response requires values",
		},
		{
			name: "form missing required field",
			req: ElicitationRequest{
				Type:    ElicitationForm,
				Message: "Fill",
				Fields:  []ElicitationField{{Name: "name", Required: true}},
			},
			resp:    ElicitationResponse{Accepted: true, Values: map[string]any{}},
			wantErr: "required field",
		},
		{
			name: "form valid",
			req: ElicitationRequest{
				Type:    ElicitationForm,
				Message: "Fill",
				Fields:  []ElicitationField{{Name: "name", Required: true}},
			},
			resp: ElicitationResponse{Accepted: true, Values: map[string]any{"name": "test"}},
		},
		{
			name: "text accepted",
			req:  ElicitationRequest{Type: ElicitationText, Message: "Enter"},
			resp: ElicitationResponse{Accepted: true, Value: "hello"},
		},
		{
			name: "confirm accepted",
			req:  ElicitationRequest{Type: ElicitationConfirm, Message: "Sure?"},
			resp: ElicitationResponse{Accepted: true, Value: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateElicitationResponse(tt.req, tt.resp)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q does not contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestElicitationType_Constants(t *testing.T) {
	if ElicitationText != "text" {
		t.Error("unexpected text value")
	}
	if ElicitationSelect != "select" {
		t.Error("unexpected select value")
	}
	if ElicitationConfirm != "confirm" {
		t.Error("unexpected confirm value")
	}
	if ElicitationForm != "form" {
		t.Error("unexpected form value")
	}
}
