package prompt

import (
	"strings"
	"testing"
)

func TestTemplate_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    Template
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid template",
			tmpl: Template{
				Name:    "greeting",
				Content: "Hello, {{.name}}!",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			tmpl: Template{
				Name:    "",
				Content: "Hello",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "empty content",
			tmpl: Template{
				Name:    "test",
				Content: "",
			},
			wantErr: true,
			errMsg:  "content is required",
		},
		{
			name: "invalid template syntax",
			tmpl: Template{
				Name:    "bad",
				Content: "Hello {{.name",
			},
			wantErr: true,
			errMsg:  "parse error",
		},
		{
			name: "plain text template",
			tmpl: Template{
				Name:    "plain",
				Content: "No variables here.",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTemplate_Render_Basic(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    Template
		vars    map[string]any
		want    string
		wantErr bool
	}{
		{
			name: "simple variable substitution",
			tmpl: Template{
				Name:    "greeting",
				Content: "Hello, {{.name}}!",
			},
			vars: map[string]any{"name": "Alice"},
			want: "Hello, Alice!",
		},
		{
			name: "multiple variables",
			tmpl: Template{
				Name:    "intro",
				Content: "I am {{.name}}, a {{.role}}.",
			},
			vars: map[string]any{"name": "Bob", "role": "developer"},
			want: "I am Bob, a developer.",
		},
		{
			name: "no variables",
			tmpl: Template{
				Name:    "static",
				Content: "Hello, world!",
			},
			vars: nil,
			want: "Hello, world!",
		},
		{
			name: "invalid template",
			tmpl: Template{
				Name:    "",
				Content: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.tmpl.Render(tt.vars)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.want {
				t.Errorf("Render() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestTemplate_Render_DefaultVariables(t *testing.T) {
	tmpl := Template{
		Name:    "greeting",
		Content: "Hello, {{.name}}! You are {{.role}}.",
		Variables: map[string]string{
			"name": "World",
			"role": "user",
		},
	}

	// Use default values.
	result, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, World! You are user." {
		t.Errorf("with defaults: %q", result)
	}

	// Override one default.
	result, err = tmpl.Render(map[string]any{"name": "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, Alice! You are user." {
		t.Errorf("with partial override: %q", result)
	}

	// Override all defaults.
	result, err = tmpl.Render(map[string]any{"name": "Bob", "role": "admin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, Bob! You are admin." {
		t.Errorf("with full override: %q", result)
	}
}

func TestTemplate_Render_ConditionalTemplate(t *testing.T) {
	tmpl := Template{
		Name:    "conditional",
		Content: "{{if .verbose}}Detailed output{{else}}Brief output{{end}}",
	}

	result, err := tmpl.Render(map[string]any{"verbose": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Detailed output" {
		t.Errorf("verbose=true: %q", result)
	}

	result, err = tmpl.Render(map[string]any{"verbose": false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Brief output" {
		t.Errorf("verbose=false: %q", result)
	}
}

func TestTemplate_Render_RangeTemplate(t *testing.T) {
	tmpl := Template{
		Name:    "list",
		Content: "Items:{{range .items}} {{.}}{{end}}",
	}

	result, err := tmpl.Render(map[string]any{
		"items": []string{"a", "b", "c"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Items: a b c" {
		t.Errorf("range result: %q", result)
	}
}

func TestTemplate_Render_EmptyVarsWithDefaults(t *testing.T) {
	tmpl := Template{
		Name:    "test",
		Content: "{{.greeting}}",
		Variables: map[string]string{
			"greeting": "hi",
		},
	}

	result, err := tmpl.Render(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hi" {
		t.Errorf("expected default, got %q", result)
	}
}
