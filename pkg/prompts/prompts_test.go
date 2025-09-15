package prompts

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
)

func TestPromptManager_NewStringTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		template     string
		wantErr      bool
	}{
		{
			name:         "valid template",
			templateName: "test_template",
			template:     "Hello {{.name}}!",
			wantErr:      false,
		},
		{
			name:         "empty template name",
			templateName: "",
			template:     "Hello {{.name}}!",
			wantErr:      true,
		},
		{
			name:         "template without variables",
			templateName: "no_vars",
			template:     "Hello World!",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewPromptManager()
			if err != nil {
				t.Fatalf("Failed to create prompt manager: %v", err)
			}

			template, err := manager.NewStringTemplate(tt.templateName, tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStringTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && template == nil {
				t.Error("NewStringTemplate() returned nil template")
			}
		})
	}
}

func TestPromptManager_NewDefaultAdapter(t *testing.T) {
	tests := []struct {
		name        string
		adapterName string
		template    string
		variables   []string
		wantErr     bool
	}{
		{
			name:        "valid adapter",
			adapterName: "test_adapter",
			template:    "Translate: {{.text}}",
			variables:   []string{"text"},
			wantErr:     false,
		},
		{
			name:        "empty adapter name",
			adapterName: "",
			template:    "Translate: {{.text}}",
			variables:   []string{"text"},
			wantErr:     true,
		},
		{
			name:        "empty template",
			adapterName: "test_adapter",
			template:    "",
			variables:   []string{"text"},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewPromptManager()
			if err != nil {
				t.Fatalf("Failed to create prompt manager: %v", err)
			}

			adapter, err := manager.NewDefaultAdapter(tt.adapterName, tt.template, tt.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDefaultAdapter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && adapter == nil {
				t.Error("NewDefaultAdapter() returned nil adapter")
			}
		})
	}
}

func TestPromptManager_HealthCheck(t *testing.T) {
	manager, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	ctx := context.Background()
	err = manager.HealthCheck(ctx)
	if err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestNewStringPromptTemplate_ConvenienceFunction(t *testing.T) {
	template, err := NewStringPromptTemplate("test", "Hello {{.name}}!")
	if err != nil {
		t.Fatalf("NewStringPromptTemplate() error = %v", err)
	}

	if template == nil {
		t.Error("NewStringPromptTemplate() returned nil template")
	}

	variables := template.GetInputVariables()
	expected := []string{"name"}
	if len(variables) != len(expected) || variables[0] != expected[0] {
		t.Errorf("GetInputVariables() = %v, want %v", variables, expected)
	}
}

func TestNewDefaultPromptAdapter_ConvenienceFunction(t *testing.T) {
	adapter, err := NewDefaultPromptAdapter("test", "Translate: {{.text}}", []string{"text"})
	if err != nil {
		t.Fatalf("NewDefaultPromptAdapter() error = %v", err)
	}

	if adapter == nil {
		t.Error("NewDefaultPromptAdapter() returned nil adapter")
	}

	variables := adapter.GetInputVariables()
	expected := []string{"text"}
	if len(variables) != len(expected) || variables[0] != expected[0] {
		t.Errorf("GetInputVariables() = %v, want %v", variables, expected)
	}
}

func TestConfig_WithOptions(t *testing.T) {
	// Test configuration with various options
	config := DefaultConfig()
	config.EnableMetrics = false
	config.EnableTracing = false

	manager, err := NewPromptManager(
		WithConfig(config),
	)

	if err != nil {
		t.Fatalf("NewPromptManager() error = %v", err)
	}

	if manager.config.EnableMetrics != false {
		t.Error("WithConfig() did not set metrics configuration correctly")
	}

	if manager.config.EnableTracing != false {
		t.Error("WithConfig() did not set tracing configuration correctly")
	}
}

func TestErrorHandling(t *testing.T) {
	manager, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	// Test template creation with invalid name
	_, err = manager.NewStringTemplate("", "Hello {{.name}}")
	if err == nil {
		t.Error("Expected error for empty template name")
	}

	// Test adapter creation with invalid parameters
	_, err = manager.NewDefaultAdapter("", "Test {{.value}}", []string{"value"})
	if err == nil {
		t.Error("Expected error for empty adapter name")
	}

	_, err = manager.NewDefaultAdapter("test", "", []string{"value"})
	if err == nil {
		t.Error("Expected error for empty template")
	}
}

func BenchmarkStringPromptTemplate_Format(b *testing.B) {
	manager, _ := NewPromptManager()
	template, _ := manager.NewStringTemplate("bench", "Hello {{.name}}, you are {{.age}} years old!")

	inputs := map[string]interface{}{
		"name": "Alice",
		"age":  "30",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := template.Format(ctx, inputs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefaultPromptAdapter_Format(b *testing.B) {
	manager, _ := NewPromptManager()
	adapter, _ := manager.NewDefaultAdapter("bench", "Translate {{.text}} to {{.language}}", []string{"text", "language"})

	inputs := map[string]interface{}{
		"text":     "Hello World",
		"language": "Spanish",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := adapter.Format(ctx, inputs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestMetricsCollection(t *testing.T) {
	// Create metrics for testing
	metrics := NewMetrics(nil)

	// Test metrics recording (these should not panic)
	metrics.RecordTemplateCreated("string")
	metrics.RecordTemplateExecuted("test_template", 0.1)
	metrics.RecordFormattingRequest("default", 0.05)
	metrics.RecordCacheHit()
	metrics.RecordCacheMiss()

	// These should not panic
	metrics.RecordTemplateError("test_template", "parse_error")
	metrics.RecordFormattingError("default", "missing_variable")
	metrics.RecordAdapterError("default", "validation_error")
	metrics.RecordValidationError("missing_variable")
	metrics.RecordAdapterRequest("default")
}

func TestTracingIntegration(t *testing.T) {
	// Create a noop tracer for testing
	tracer := &iface.TracerNoOp{}

	manager, err := NewPromptManager(
		WithTracer(tracer),
	)
	if err != nil {
		t.Fatalf("NewPromptManager() error = %v", err)
	}

	template, err := manager.NewStringTemplate("test", "Hello {{.name}}!")
	if err != nil {
		t.Fatalf("NewStringTemplate() error = %v", err)
	}

	ctx := context.Background()
	_, err = template.Format(ctx, map[string]interface{}{"name": "Test"})
	if err != nil {
		t.Errorf("Format() error = %v", err)
	}
}

func TestTimeoutHandling(t *testing.T) {
	config := DefaultConfig()
	config.DefaultTemplateTimeout = 1 * time.Nanosecond // Very short timeout

	manager, err := NewPromptManager(WithConfig(config))
	if err != nil {
		t.Fatalf("NewPromptManager() error = %v", err)
	}

	template, err := manager.NewStringTemplate("test", "Hello {{.name}}!")
	if err != nil {
		t.Fatalf("NewStringTemplate() error = %v", err)
	}

	// This should not timeout for simple operations, but tests the timeout configuration
	ctx := context.Background()
	_, err = template.Format(ctx, map[string]interface{}{"name": "Test"})
	if err != nil {
		t.Errorf("Format() error = %v", err)
	}
}
