package prompts

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
	"github.com/lookatitude/beluga-ai/pkg/prompts/internal"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestPromptManager_NewStringTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		template     string
		wantErr      bool
		expectedVars []string
	}{
		{
			name:         "valid template",
			templateName: "test_template",
			template:     "Hello {{.name}}!",
			wantErr:      false,
			expectedVars: []string{"name"},
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
			expectedVars: []string{},
		},
		{
			name:         "template with multiple variables",
			templateName: "multi_vars",
			template:     "{{.greeting}} {{.name}}, you are {{.age}} years old!",
			wantErr:      false,
			expectedVars: []string{"age", "greeting", "name"},
		},
		{
			name:         "template with duplicate variables",
			templateName: "duplicate_vars",
			template:     "Hello {{.name}}, {{.name}}!",
			wantErr:      false,
			expectedVars: []string{"name"},
		},
		{
			name:         "template with complex variable names",
			templateName: "complex_vars",
			template:     "{{.user_name}} logged in from {{.ip_address}}",
			wantErr:      false,
			expectedVars: []string{"ip_address", "user_name"},
		},
		{
			name:         "invalid template syntax",
			templateName: "invalid",
			template:     "Hello {{.name",
			wantErr:      true,
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

			if !tt.wantErr {
				if template == nil {
					t.Error("NewStringTemplate() returned nil template")
					return
				}

				// Check template name
				if template.Name() != tt.templateName {
					t.Errorf("Template name = %v, want %v", template.Name(), tt.templateName)
				}

				// Check input variables
				vars := template.GetInputVariables()
				if len(vars) != len(tt.expectedVars) {
					t.Errorf("Input variables length = %d, want %d", len(vars), len(tt.expectedVars))
					return
				}

				// Sort both slices for comparison (since variable extraction order may vary)
				sort.Strings(vars)
				sort.Strings(tt.expectedVars)
				for i, v := range vars {
					if v != tt.expectedVars[i] {
						t.Errorf("Input variable %d = %v, want %v", i, v, tt.expectedVars[i])
					}
				}
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
		{
			name:        "multiple variables",
			adapterName: "multi_var_adapter",
			template:    "{{.action}} {{.text}} to {{.language}}",
			variables:   []string{"action", "text", "language"},
			wantErr:     false,
		},
		{
			name:        "no variables",
			adapterName: "no_vars_adapter",
			template:    "Hello World!",
			variables:   []string{},
			wantErr:     false,
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

			if !tt.wantErr {
				if adapter == nil {
					t.Error("NewDefaultAdapter() returned nil adapter")
					return
				}

				// Check input variables
				vars := adapter.GetInputVariables()
				if len(vars) != len(tt.variables) {
					t.Errorf("Input variables length = %d, want %d", len(vars), len(tt.variables))
					return
				}

				sort.Strings(vars)
				sort.Strings(tt.variables)
				for i, v := range vars {
					if v != tt.variables[i] {
						t.Errorf("Input variable %d = %v, want %v", i, v, tt.variables[i])
					}
				}
			}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		})
	}
}

func TestDefaultPromptAdapter_Format(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		variables   []string
		inputs      map[string]interface{}
		expected    string
		expectError bool
		errorType   string
	}{
		{
			name:      "simple replacement",
			template:  "Hello {{.name}}!",
			variables: []string{"name"},
			inputs: map[string]interface{}{
				"name": "Alice",
			},
			expected: "Hello Alice!",
		},
		{
			name:      "multiple variables",
			template:  "{{.greeting}} {{.name}}, welcome!",
			variables: []string{"greeting", "name"},
			inputs: map[string]interface{}{
				"greeting": "Hi",
				"name":     "Bob",
			},
			expected: "Hi Bob, welcome!",
		},
		{
			name:      "no variables",
			template:  "Hello World!",
			variables: []string{},
			inputs:    map[string]interface{}{},
			expected:  "Hello World!",
		},
		{
			name:        "missing required variable",
			template:    "Hello {{.name}}!",
			variables:   []string{"name"},
			inputs:      map[string]interface{}{},
			expectError: true,
			errorType:   "variable_missing",
		},
		{
			name:      "wrong variable type",
			template:  "Count: {{.number}}",
			variables: []string{"number"},
			inputs: map[string]interface{}{
				"number": 42, // int instead of string
			},
			expectError: true,
			errorType:   "variable_invalid",
		},
		{
			name:      "extra variables ignored",
			template:  "Hello {{.name}}!",
			variables: []string{"name"},
			inputs: map[string]interface{}{
				"name":  "Alice",
				"extra": "ignored",
			},
			expected: "Hello Alice!",
		},
		{
			name:      "special characters",
			template:  "Path: {{.path}}",
			variables: []string{"path"},
			inputs: map[string]interface{}{
				"path": "/usr/local/bin",
			},
			expected: "Path: /usr/local/bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewDefaultPromptAdapter("test_"+tt.name, tt.template, tt.variables)
			if err != nil {
				t.Fatalf("Failed to create adapter: %v", err)
			}

			ctx := context.Background()
			result, err := adapter.Format(ctx, tt.inputs)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" {
					var promptErr *PromptError
					if errors.As(err, &promptErr) {
						if promptErr.Code != tt.errorType {
							t.Errorf("Expected error type %s, got %s", tt.errorType, promptErr.Code)
						}
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Format returned nil result")
				return
			}

			actual, ok := result.(string)
			if !ok {
				t.Errorf("Expected string result, got %T", result)
				return
			}

			if actual != tt.expected {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Errorf("Format() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestPromptManager_NewChatAdapter(t *testing.T) {
	tests := []struct {
		name           string
		adapterName    string
		systemTemplate string
		userTemplate   string
		variables      []string
		wantErr        bool
	}{
		{
			name:           "valid chat adapter",
			adapterName:    "chat_adapter",
			systemTemplate: "You are a helpful assistant.",
			userTemplate:   "Please help with: {{.query}}",
			variables:      []string{"query"},
			wantErr:        false,
		},
		{
			name:           "empty adapter name",
			adapterName:    "",
			systemTemplate: "You are a helpful assistant.",
			userTemplate:   "Please help with: {{.query}}",
			variables:      []string{"query"},
			wantErr:        true,
		},
		{
			name:           "empty user template",
			adapterName:    "chat_adapter",
			systemTemplate: "You are a helpful assistant.",
			userTemplate:   "",
			variables:      []string{"query"},
			wantErr:        true,
		},
		{
			name:           "no system template",
			adapterName:    "chat_adapter",
			systemTemplate: "",
			userTemplate:   "Please help with: {{.query}}",
			variables:      []string{"query"},
			wantErr:        false,
		},
		{
			name:           "multiple variables",
			adapterName:    "complex_chat",
			systemTemplate: "You are {{.role}}.",
			userTemplate:   "{{.action}} {{.subject}}",
			variables:      []string{"role", "action", "subject"},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewPromptManager()
			if err != nil {
				t.Fatalf("Failed to create prompt manager: %v", err)
			}

			adapter, err := manager.NewChatAdapter(tt.adapterName, tt.systemTemplate, tt.userTemplate, tt.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewChatAdapter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

			if !tt.wantErr && adapter == nil {
				t.Error("NewChatAdapter() returned nil adapter")
			}
		})
	}
}

func TestChatPromptAdapter_Format(t *testing.T) {
	tests := []struct {
		name           string
		systemTemplate string
		userTemplate   string
		variables      []string
		inputs         map[string]interface{}
		expectSystem   bool
		expectedUser   string
		expectError    bool
		errorType      string
	}{
		{
			name:           "system and user messages",
			systemTemplate: "You are a {{.role}}.",
			userTemplate:   "Please {{.action}} {{.subject}}.",
			variables:      []string{"role", "action", "subject"},
			inputs: map[string]interface{}{
				"role":    "teacher",
				"action":  "explain",
				"subject": "quantum physics",
			},
			expectSystem: true,
			expectedUser: "Please explain quantum physics.",
		},
		{
			name:           "user message only",
			systemTemplate: "",
			userTemplate:   "Hello {{.name}}!",
			variables:      []string{"name"},
			inputs: map[string]interface{}{
				"name": "Alice",
			},
			expectSystem: false,
			expectedUser: "Hello Alice!",
		},
		{
			name:           "with chat history",
			systemTemplate: "You are helpful.",
			userTemplate:   "What is {{.topic}}?",
			variables:      []string{"topic"},
			inputs: map[string]interface{}{
				"topic": "AI",
				"history": []schema.Message{
					schema.NewChatMessage("user", "What is machine learning?"),
					schema.NewChatMessage("assistant", "Machine learning is..."),
				},
			},
			expectSystem: true,
			expectedUser: "What is AI?",
		},
		{
			name:           "missing variable in system template",
			systemTemplate: "You are {{.role}}.",
			userTemplate:   "Hello!",
			variables:      []string{"role"},
			inputs:         map[string]interface{}{},
			expectError:    true,
			errorType:      "variable_missing",
		},
		{
			name:           "missing variable in user template",
			systemTemplate: "",
			userTemplate:   "Hello {{.name}}!",
			variables:      []string{"name"},
			inputs:         map[string]interface{}{},
			expectError:    true,
			errorType:      "variable_missing",
		},
		{
			name:           "wrong variable type",
			systemTemplate: "",
			userTemplate:   "Count: {{.number}}",
			variables:      []string{"number"},
			inputs: map[string]interface{}{
				"number": 42, // should be string
			},
			expectError: true,
			errorType:   "variable_invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewChatPromptAdapter("test_"+tt.name, tt.systemTemplate, tt.userTemplate, tt.variables)
			if err != nil {
				t.Fatalf("Failed to create adapter: %v", err)
			}

			ctx := context.Background()
			result, err := adapter.Format(ctx, tt.inputs)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" {
					var promptErr *PromptError
					if errors.As(err, &promptErr) {
						if promptErr.Code != tt.errorType {
							t.Errorf("Expected error type %s, got %s", tt.errorType, promptErr.Code)
						}
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Format returned nil result")
				return
			}

			messages, ok := result.([]schema.Message)
			if !ok {
				t.Errorf("Expected []schema.Message result, got %T", result)
				return
			}

			if tt.expectSystem {
				if len(messages) < 2 {
					t.Errorf("Expected at least 2 messages (system + user), got %d", len(messages))
					return
				}

				systemMsg := messages[0]
				if systemMsg.GetType() != "system" {
					t.Errorf("First message type = %s, want system", systemMsg.GetType())
				}
			}

			// Find the user message (it should be the last one)
			userMsg := messages[len(messages)-1]
			if userMsg.GetType() != "user" {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				t.Errorf("Last message type = %s, want user", userMsg.GetType())
			}

			if userMsg.GetContent() != tt.expectedUser {
				t.Errorf("User message content = %q, want %q", userMsg.GetContent(), tt.expectedUser)
			}
		})
	}
}

func TestPromptManager_HealthCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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

func TestStringTemplate_Format(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		inputs      map[string]interface{}
		expected    string
		expectError bool
		errorType   string
	}{
		{
			name:     "simple replacement",
			template: "Hello {{.name}}!",
			inputs: map[string]interface{}{
				"name": "Alice",
			},
			expected: "Hello Alice!",
		},
		{
			name:     "multiple variables",
			template: "{{.greeting}} {{.name}}, you are {{.age}} years old!",
			inputs: map[string]interface{}{
				"greeting": "Hi",
				"name":     "Bob",
				"age":      "25",
			},
			expected: "Hi Bob, you are 25 years old!",
		},
		{
			name:     "no variables",
			template: "Hello World!",
			inputs:   map[string]interface{}{},
			expected: "Hello World!",
		},
		{
			name:     "numeric values",
			template: "Count: {{.count}}",
			inputs: map[string]interface{}{
				"count": 42,
			},
			expected: "Count: 42",
		},
		{
			name:     "boolean values",
			template: "Active: {{.active}}",
			inputs: map[string]interface{}{
				"active": true,
			},
			expected: "Active: true",
		},
		{
			name:        "missing required variable",
			template:    "Hello {{.name}}!",
			inputs:      map[string]interface{}{},
			expectError: true,
			errorType:   "variable_missing",
		},
		{
			name:     "extra variables provided",
			template: "Hello {{.name}}!",
			inputs: map[string]interface{}{
				"name":  "Alice",
				"extra": "ignored",
			},
			expected: "Hello Alice!",
		},
		{
			name:     "special characters",
			template: "{{.message}} - {{.timestamp}}",
			inputs: map[string]interface{}{
				"message":   "System started",
				"timestamp": "2024-01-01 12:00:00",
			},
			expected: "System started - 2024-01-01 12:00:00",
		},
		{
			name:     "template with newlines",
			template: "Line 1: {{.var1}}\nLine 2: {{.var2}}",
			inputs: map[string]interface{}{
				"var1": "value1",
				"var2": "value2",
			},
			expected: "Line 1: value1\nLine 2: value2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewStringPromptTemplate("test_"+tt.name, tt.template)
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}

			ctx := context.Background()
			result, err := template.Format(ctx, tt.inputs)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				// Check error type if specified
				if tt.errorType != "" {
					var promptErr *PromptError
					if errors.As(err, &promptErr) {
						if promptErr.Code != tt.errorType {
							t.Errorf("Expected error type %s, got %s", tt.errorType, promptErr.Code)
						}
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Format returned nil result")
				return
			}

			// Check if result is a StringPromptValue
			stringValue, ok := result.(internal.StringPromptValue)
			if !ok {
				t.Errorf("Expected StringPromptValue, got %T", result)
				return
			}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			actual := stringValue.ToString()
			if actual != tt.expected {
				t.Errorf("Format() = %q, want %q", actual, tt.expected)
			}

			// Test ToMessages conversion
			messages := stringValue.ToMessages()
			if len(messages) != 1 {
				t.Errorf("ToMessages() returned %d messages, want 1", len(messages))
			}
		})
	}
}

func TestStringTemplate_Validate(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		expectError bool
	}{
		{
			name:        "valid template",
			template:    "Hello {{.name}}!",
			expectError: false,
		},
		{
			name:        "template without variables",
			template:    "Hello World!",
			expectError: false,
		},
		{
			name:        "complex template",
			template:    "{{.greeting}} {{.name}}, welcome to {{.place}}!",
			expectError: false,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := NewStringPromptTemplate("test_"+tt.name, tt.template)
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}

			err = template.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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

func TestVariableValidation(t *testing.T) {
	tests := []struct {
		name         string
		template     string
		requiredVars []string
		providedVars map[string]interface{}
		expectError  bool
		errorType    string
		validateType bool
	}{
		{
			name:         "all variables provided",
			template:     "Hello {{.name}}, you are {{.age}} years old.",
			requiredVars: []string{"name", "age"},
			providedVars: map[string]interface{}{
				"name": "Alice",
				"age":  "25",
			},
			expectError: false,
		},
		{
			name:         "missing required variable",
			template:     "Hello {{.name}}!",
			requiredVars: []string{"name"},
			providedVars: map[string]interface{}{},
			expectError:  true,
			errorType:    "variable_missing",
		},
		{
			name:         "extra variables provided",
			template:     "Hello {{.name}}!",
			requiredVars: []string{"name"},
			providedVars: map[string]interface{}{
				"name":  "Alice",
				"extra": "ignored",
			},
			expectError: false,
		},
		{
			name:         "nil variable value",
			template:     "Hello {{.name}}!",
			requiredVars: []string{"name"},
			providedVars: map[string]interface{}{
				"name": nil,
			},
			expectError: true,
			errorType:   "variable_invalid",
		},
		{
			name:         "wrong variable type - int instead of string",
			template:     "Count: {{.number}}",
			requiredVars: []string{"number"},
			providedVars: map[string]interface{}{
				"number": 42,
			},
			expectError:  true,
			errorType:    "variable_invalid",
			validateType: true,
		},
		{
			name:         "correct variable type - string",
			template:     "Count: {{.number}}",
			requiredVars: []string{"number"},
			providedVars: map[string]interface{}{
				"number": "42",
			},
			expectError: false,
		},
		{
			name:         "multiple missing variables",
			template:     "{{.greeting}} {{.name}}!",
			requiredVars: []string{"greeting", "name"},
			providedVars: map[string]interface{}{
				"greeting": "Hello",
			},
			expectError: true,
			errorType:   "variable_missing",
		},
		{
			name:         "empty string variable",
			template:     "Hello {{.name}}!",
			requiredVars: []string{"name"},
			providedVars: map[string]interface{}{
				"name": "",
			},
			expectError: false, // Empty strings are valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create template
			template, err := NewStringPromptTemplate("test_"+tt.name, tt.template)
			if err != nil {
				t.Fatalf("Failed to create template: %v", err)
			}

			// Test with validation enabled
			config := DefaultConfig()
			config.ValidateVariables = true
			config.StrictVariableCheck = tt.validateType

			manager, err := NewPromptManager(WithConfig(config))
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			// Create template through manager to get validation
			mgrTemplate, err := manager.NewStringTemplate("test_"+tt.name+"_mgr", tt.template)
			if err != nil {
				t.Fatalf("Failed to create template through manager: %v", err)
			}

			ctx := context.Background()
			_, err = mgrTemplate.Format(ctx, tt.providedVars)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}

				if tt.errorType != "" {
					var promptErr *PromptError
					if errors.As(err, &promptErr) {
						if promptErr.Code != tt.errorType {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
							t.Errorf("Expected error type %s, got %s", tt.errorType, promptErr.Code)
						}
					} else {
						t.Errorf("Expected PromptError, got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Also test direct template formatting (should work regardless of validation settings)
			_, directErr := template.Format(ctx, tt.providedVars)
			if tt.expectError && tt.errorType == "variable_missing" {
				// Direct template should still fail for missing variables
				if directErr == nil {
					t.Error("Expected direct template to fail for missing variables")
				}
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringPromptTemplate_Format_Complex(b *testing.B) {
	template, _ := NewStringPromptTemplate("bench_complex",
		"Dear {{.customer_name}}, your order #{{.order_id}} for {{.product_name}} has been {{.status}}. Total: ${{.total_amount}}. Thank you for shopping with {{.company_name}}!")

	inputs := map[string]interface{}{
		"customer_name": "John Doe",
		"order_id":      "12345",
		"product_name":  "Wireless Headphones",
		"status":        "confirmed",
		"total_amount":  "299.99",
		"company_name":  "TechStore Inc",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := template.Format(ctx, inputs)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStringPromptTemplate_Format_LargeTemplate(b *testing.B) {
	// Create a large template with many variables
	var templateStr strings.Builder
	var inputs map[string]interface{} = make(map[string]interface{})

	templateStr.WriteString("Welcome ")
	for i := 0; i < 50; i++ {
		if i > 0 {
			templateStr.WriteString(", ")
		}
		templateStr.WriteString(fmt.Sprintf("{{.field%d}}", i))
		inputs[fmt.Sprintf("field%d", i)] = fmt.Sprintf("value%d", i)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}

	template, _ := NewStringPromptTemplate("bench_large", templateStr.String())
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

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := adapter.Format(ctx, inputs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDefaultPromptAdapter_Format_LongText(b *testing.B) {
	manager, _ := NewPromptManager()
	adapter, _ := manager.NewDefaultAdapter("bench_long", "Analyze the following text: {{.text}}", []string{"text"})

	// Create a long text
	var longText strings.Builder
	for i := 0; i < 1000; i++ {
		longText.WriteString(fmt.Sprintf("Word%d ", i))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	inputs := map[string]interface{}{
		"text": longText.String(),
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

func BenchmarkChatPromptAdapter_Format(b *testing.B) {
	adapter, _ := NewChatPromptAdapter("bench_chat",
		"You are a {{.role}} assistant.",
		"{{.question}}",
		[]string{"role", "question"})

	inputs := map[string]interface{}{
		"role":     "helpful",
		"question": "What is the capital of France?",
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := adapter.Format(ctx, inputs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkChatPromptAdapter_Format_WithHistory(b *testing.B) {
	adapter, _ := NewChatPromptAdapter("bench_chat_history",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		"You are a helpful assistant.",
		"{{.question}}",
		[]string{"question"})

	// Create chat history
	history := []schema.Message{
		schema.NewChatMessage("user", "What is AI?"),
		schema.NewChatMessage("assistant", "AI stands for Artificial Intelligence..."),
		schema.NewChatMessage("user", "How does it work?"),
		schema.NewChatMessage("assistant", "AI works by..."),
	}

	inputs := map[string]interface{}{
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		"question": "What are the applications?",
		"history":  history,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := adapter.Format(ctx, inputs)
		if err != nil {
			b.Fatal(err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		}
	}
}

func BenchmarkPromptManager_NewStringTemplate(b *testing.B) {
	manager, _ := NewPromptManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("template_%d", i)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		template := fmt.Sprintf("Hello {{.name%d}}!", i)
		_, err := manager.NewStringTemplate(name, template)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPromptManager_NewDefaultAdapter(b *testing.B) {
	manager, _ := NewPromptManager()

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("adapter_%d", i)
		template := fmt.Sprintf("Process {{.data%d}}", i)
		variables := []string{fmt.Sprintf("data%d", i)}
		_, err := manager.NewDefaultAdapter(name, template, variables)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPromptManager_HealthCheck(b *testing.B) {
	manager, _ := NewPromptManager()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := manager.HealthCheck(ctx)
		if err != nil {
			b.Fatal(err)
		}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
}

func BenchmarkTemplateCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("template_%d", i)
		template := fmt.Sprintf("Template with {{.var%d}}", i)
		_, err := NewStringPromptTemplate(name, template)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdapterCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("adapter_%d", i)
		template := fmt.Sprintf("Adapter for {{.input%d}}", i)
		variables := []string{fmt.Sprintf("input%d", i)}
		_, err := NewDefaultPromptAdapter(name, template, variables)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVariableValidation(b *testing.B) {
	config := DefaultConfig()
	config.ValidateVariables = true
	manager, _ := NewPromptManager(WithConfig(config))

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	template, _ := manager.NewStringTemplate("bench_validation", "Hello {{.name}}, {{.age}}, {{.city}}!")

	inputs := map[string]interface{}{
		"name": "Alice",
		"age":  "25",
		"city": "New York",
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

func BenchmarkConcurrentOperations(b *testing.B) {
	manager, _ := NewPromptManager()
	template, _ := manager.NewStringTemplate("concurrent_bench", "Hello {{.name}} from iteration {{.iter}}!")

	ctx := context.Background()
	numGoroutines := runtime.NumCPU()
	iterationsPerGoroutine := b.N / numGoroutines

	if iterationsPerGoroutine == 0 {
		iterationsPerGoroutine = 1
	}

	b.ResetTimer()

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < iterationsPerGoroutine; j++ {
				inputs := map[string]interface{}{
					"name": fmt.Sprintf("User%d", goroutineID),
					"iter": j,
				}
				_, err := template.Format(ctx, inputs)
				if err != nil {
					b.Error(err)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestCacheConfiguration(t *testing.T) {
	tests := []struct {
		name               string
		enableCache        bool
		cacheTTL           time.Duration
		maxCacheSize       int
		expectEnableCache  bool
		expectCacheTTL     time.Duration
		expectMaxCacheSize int
	}{
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		{
			name:               "default cache settings",
			enableCache:        true,
			cacheTTL:           5 * time.Minute,
			maxCacheSize:       100,
			expectEnableCache:  true,
			expectCacheTTL:     5 * time.Minute,
			expectMaxCacheSize: 100,
		},
		{
			name:               "cache disabled",
			enableCache:        false,
			cacheTTL:           10 * time.Minute,
			maxCacheSize:       200,
			expectEnableCache:  false,
			expectCacheTTL:     10 * time.Minute,
			expectMaxCacheSize: 200,
		},
		{
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			name:               "custom cache settings",
			enableCache:        true,
			cacheTTL:           30 * time.Minute,
			maxCacheSize:       500,
			expectEnableCache:  true,
			expectCacheTTL:     30 * time.Minute,
			expectMaxCacheSize: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.EnableTemplateCache = tt.enableCache
			// config.CacheTTL = tt.cacheTTL // Duration interface - handled by DefaultConfig
			config.MaxCacheSize = tt.maxCacheSize

			manager, err := NewPromptManager(WithConfig(config))
			if err != nil {
				t.Fatalf("Failed to create prompt manager: %v", err)
			}

			mgrConfig := manager.GetConfig()
			if mgrConfig.EnableTemplateCache != tt.expectEnableCache {
				t.Errorf("EnableTemplateCache = %v, want %v", mgrConfig.EnableTemplateCache, tt.expectEnableCache)
			}

			// Note: CacheTTL is an interface, we can't directly compare
			// This test focuses on the configuration being set correctly

			if mgrConfig.MaxCacheSize != tt.expectMaxCacheSize {
				t.Errorf("MaxCacheSize = %d, want %d", mgrConfig.MaxCacheSize, tt.expectMaxCacheSize)
			}
		})
	}
}

func TestCacheMetrics(t *testing.T) {
	// Test that cache metrics can be recorded without panicking
	metrics := NoOpMetrics()

	// These should not panic even though metrics are no-op
	ctx := context.Background()
	metrics.RecordCacheHit(ctx, "test")
	metrics.RecordCacheMiss(ctx, "test")
	metrics.RecordCacheSize(ctx, 42, "test")

	// Test multiple calls
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			metrics.RecordCacheHit(ctx, "test")
		} else {
			metrics.RecordCacheMiss(ctx, "test")
		}
		metrics.RecordCacheSize(ctx, int64(i*10), "test")
	}
}

func TestTemplateCachingBehavior(t *testing.T) {
	// Test that demonstrates expected caching behavior
	// Note: This test documents the expected behavior for when caching is implemented
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	config := DefaultConfig()
	config.EnableTemplateCache = true
	// CacheTTL configuration testing
	config.MaxCacheSize = 10

	manager, err := NewPromptManager(WithConfig(config))
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	ctx := context.Background()

	// Create the same template multiple times
	templateName := "cached_template"
	templateStr := "Hello {{.name}}!"

	// First creation
	template1, err := manager.NewStringTemplate(templateName, templateStr)
	if err != nil {
		t.Fatalf("Failed to create first template: %v", err)
	}

	// Second creation with same name (should reuse if caching implemented)
	template2, err := manager.NewStringTemplate(templateName+"_different", templateStr)
	if err != nil {
		t.Fatalf("Failed to create second template: %v", err)
	}

	// Templates should be different instances even with same content
	if template1 == template2 {
		t.Log("Note: Templates are identical instances - caching may be implemented")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Both should work identically
	inputs := map[string]interface{}{"name": "Test"}

	result1, err := template1.Format(ctx, inputs)
	if err != nil {
		t.Fatalf("Failed to format first template: %v", err)
	}

	result2, err := template2.Format(ctx, inputs)
	if err != nil {
		t.Fatalf("Failed to format second template: %v", err)
	}

	// Results should be identical
	stringResult1, ok1 := result1.(internal.StringPromptValue)
	stringResult2, ok2 := result2.(internal.StringPromptValue)

	if !ok1 || !ok2 {
		t.Fatalf("Expected StringPromptValue results")
	}

	if stringResult1.ToString() != stringResult2.ToString() {
		t.Errorf("Template results differ: %q vs %q", stringResult1.ToString(), stringResult2.ToString())
	}
}

func TestCacheSizeLimits(t *testing.T) {
	// Test behavior with cache size limits
	config := DefaultConfig()
	config.EnableTemplateCache = true
	config.MaxCacheSize = 3 // Very small cache

	manager, err := NewPromptManager(WithConfig(config))
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	ctx := context.Background()

	// Create multiple templates
	for i := 0; i < 10; i++ {
		templateName := fmt.Sprintf("template_%d", i)
		templateStr := fmt.Sprintf("Template %d: {{.value}}", i)

		template, err := manager.NewStringTemplate(templateName, templateStr)
		if err != nil {
			t.Fatalf("Failed to create template %d: %v", i, err)
		}

		// Test that template works
		inputs := map[string]interface{}{"value": fmt.Sprintf("test%d", i)}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		_, err = template.Format(ctx, inputs)
		if err != nil {
			t.Fatalf("Failed to format template %d: %v", i, err)
		}
	}

	// All templates should still work (no cache eviction errors expected)
	// This test ensures that even with small cache limits, functionality isn't broken
}

func TestCacheExpiration(t *testing.T) {
	// Test cache expiration behavior
	config := DefaultConfig()
	config.EnableTemplateCache = true
	// CacheTTL configuration testing

	manager, err := NewPromptManager(WithConfig(config))
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	ctx := context.Background()

	// Create a template
	template, err := manager.NewStringTemplate("expiring_template", "Hello {{.name}}!")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	// Use template immediately
	inputs := map[string]interface{}{"name": "Test"}
	_, err = template.Format(ctx, inputs)
	if err != nil {
		t.Fatalf("Failed to format template: %v", err)
	}

	// Wait for cache expiration (if implemented)
	time.Sleep(10 * time.Millisecond)

	// Template should still work even if cache expired
	_, err = template.Format(ctx, inputs)
	if err != nil {
		t.Fatalf("Failed to format template after cache expiration: %v", err)
	}
}

func TestMetricsCollection(t *testing.T) {
	// Create metrics for testing
	metrics := NoOpMetrics()

	// Test metrics recording (these should not panic)
	ctx := context.Background()
	metrics.RecordTemplateCreated(ctx, "string")
	metrics.RecordTemplateExecuted(ctx, "test_template", 100*time.Millisecond, true)
	metrics.RecordFormattingRequest(ctx, "default", 50*time.Millisecond, true)
	metrics.RecordCacheHit(ctx, "test")
	metrics.RecordCacheMiss(ctx, "test")

	// These should not panic
	metrics.RecordTemplateError(ctx, "test_template", "parse_error")
	metrics.RecordFormattingError(ctx, "default", "missing_variable")
	metrics.RecordAdapterError(ctx, "default", "validation_error")
	metrics.RecordValidationError(ctx, "missing_variable")
	metrics.RecordAdapterRequest(ctx, "default", true)
}

func TestErrorTypesAndContext(t *testing.T) {
	// Test that different error types provide appropriate context
	tests := []struct {
		name         string
		setupFunc    func() error
		expectedCode string
		checkContext func(t *testing.T, err error)
	}{
		{
			name: "template parse error",
			setupFunc: func() error {
				_, err := NewStringPromptTemplate("test", "Hello {{.name")
				return err
			},
			expectedCode: "template_parse_error",
			checkContext: func(t *testing.T, err error) {
				var promptErr *PromptError
				if errors.As(err, &promptErr) {
					if promptErr.Context == nil {
						t.Error("Expected context in parse error")
						return
					}
					if templateName, ok := promptErr.Context["template_name"]; !ok || templateName != "test" {
						t.Errorf("Expected template_name in context, got %v", promptErr.Context)
					}
				}
			},
		},
		{
			name: "variable missing error",
			setupFunc: func() error {
				template, _ := NewStringPromptTemplate("test", "Hello {{.name}}!")
				ctx := context.Background()
				_, err := template.Format(ctx, map[string]interface{}{})
				return err
			},
			expectedCode: "variable_missing",
			checkContext: func(t *testing.T, err error) {
				var promptErr *PromptError
				if errors.As(err, &promptErr) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
					if promptErr.Context == nil {
						t.Error("Expected context in variable missing error")
						return
					}
					if variableName, ok := promptErr.Context["variable_name"]; !ok || variableName != "name" {
						t.Errorf("Expected variable_name in context, got %v", promptErr.Context)
					}
					if templateName, ok := promptErr.Context["template_name"]; !ok || templateName != "test" {
						t.Errorf("Expected template_name in context, got %v", promptErr.Context)
					}
				}
			},
		},
		{
			name: "variable invalid type error",
			setupFunc: func() error {
				adapter, _ := NewDefaultPromptAdapter("test", "Count: {{.number}}", []string{"number"})
				ctx := context.Background()
				_, err := adapter.Format(ctx, map[string]interface{}{"number": 42})
				return err
			},
			expectedCode: "variable_invalid",
			checkContext: func(t *testing.T, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				var promptErr *PromptError
				if errors.As(err, &promptErr) {
					if promptErr.Context == nil {
						t.Error("Expected context in variable invalid error")
						return
					}
					if variableName, ok := promptErr.Context["variable_name"]; !ok || variableName != "number" {
						t.Errorf("Expected variable_name in context, got %v", promptErr.Context)
					}
					if expectedType, ok := promptErr.Context["expected_type"]; !ok || expectedType != "string" {
						t.Errorf("Expected expected_type in context, got %v", promptErr.Context)
					}
					if actualType, ok := promptErr.Context["actual_type"]; !ok || actualType != "int" {
						t.Errorf("Expected actual_type in context, got %v", promptErr.Context)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupFunc()
			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			var promptErr *PromptError
			if !errors.As(err, &promptErr) {
				t.Errorf("Expected PromptError, got %T", err)
				return
			}

			if promptErr.Code != tt.expectedCode {
				t.Errorf("Expected error code %s, got %s", tt.expectedCode, promptErr.Code)
			}

			if tt.checkContext != nil {
				tt.checkContext(t, err)
			}
		})
	}
}

func TestErrorWrappingAndUnwrapping(t *testing.T) {
	// Test that errors can be properly wrapped and unwrapped
	originalErr := fmt.Errorf("original error")
	wrappedErr := fmt.Errorf("wrapped: %w", originalErr)

	// Create a prompt error that wraps the original
	promptErr := NewTemplateParseError("test_op", "test_template", wrappedErr)

	// Test unwrapping
	if !errors.Is(promptErr, originalErr) {
		t.Error("Expected to find original error when unwrapping")
	}

	// Test As
	var target *PromptError
	if !errors.As(promptErr, &target) {
		t.Error("Expected to be able to cast to PromptError")
	}

	// Test that the wrapped error is accessible
	if promptErr.Err != wrappedErr {
		t.Error("Expected wrapped error to be accessible")
	}
}

func TestErrorEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		testFunc    func() error
		expectError bool
	}{
		{
			name: "empty template string",
			testFunc: func() error {
				_, err := NewStringPromptTemplate("test", "")
				return err
			},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			expectError: false, // Empty templates are allowed (static templates)
		},
		{
			name: "template with only whitespace",
			testFunc: func() error {
				_, err := NewStringPromptTemplate("test", "   ")
				return err
			},
			expectError: false, // Should work
		},
		{
			name: "nil inputs map",
			testFunc: func() error {
				template, _ := NewStringPromptTemplate("test", "Hello {{.name}}!")
				ctx := context.Background()
				_, err := template.Format(ctx, nil)
				return err
			},
			expectError: true,
		},
		{
			name: "extremely long template",
			testFunc: func() error {
				var longTemplate strings.Builder
				longTemplate.WriteString("Template: ")
				for i := 0; i < 10000; i++ {
					longTemplate.WriteString(fmt.Sprintf("{{.var%d}} ", i))
				}
				_, err := NewStringPromptTemplate("long_test", longTemplate.String())
				return err
			},
			expectError: false, // Should handle large templates
		},
		{
			name: "template with special characters",
			testFunc: func() error {
				_, err := NewStringPromptTemplate("special", "Price: ${{.price}} ({{.currency}})")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				return err
			},
			expectError: false,
		},
		{
			name: "adapter with empty variables list",
			testFunc: func() error {
				_, err := NewDefaultPromptAdapter("test", "Hello World!", []string{})
				return err
			},
			expectError: false,
		},
		{
			name: "chat adapter with empty system template",
			testFunc: func() error {
				_, err := NewChatPromptAdapter("test", "", "Hello {{.name}}!", []string{"name"})
				return err
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestErrorRecovery(t *testing.T) {
	// Test that components can recover from errors and continue working

	manager, err := NewPromptManager()
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	ctx := context.Background()

	// First, try to create a template with invalid syntax (should fail)
	_, err = manager.NewStringTemplate("invalid", "Hello {{.name")
	if err == nil {
		t.Error("Expected error for invalid template syntax")
	}

	// Now create a valid template (should succeed)
	template, err := manager.NewStringTemplate("valid", "Hello {{.name}}!")
	if err != nil {
		t.Fatalf("Failed to create valid template after error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Format the valid template (should work)
	result, err := template.Format(ctx, map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("Failed to format valid template: %v", err)
	}

	stringResult, ok := result.(internal.StringPromptValue)
	if !ok {
		t.Fatalf("Expected StringPromptValue, got %T", result)
	}

	expected := "Hello Alice!"
	if stringResult.ToString() != expected {
		t.Errorf("Expected %q, got %q", expected, stringResult.ToString())
	}
}

func TestConfigurationError(t *testing.T) {
	// Test configuration-related errors
	tests := []struct {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		name        string
		configFunc  func() *Config
		expectError bool
	}{
		{
			name: "valid configuration",
			configFunc: func() *Config {
				return DefaultConfig()
			},
			expectError: false,
		},
		{
			name: "negative cache size",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.MaxCacheSize = -1
				return config
			},
			expectError: false, // Configuration validation might not be implemented yet
		},
		{
			name: "zero TTL",
			configFunc: func() *Config {
				config := DefaultConfig()
				// config.CacheTTL = Duration(0)
				return config
			},
			expectError: false,
		},
		{
			name: "extremely large template size limit",
			configFunc: func() *Config {
				config := DefaultConfig()
				config.MaxTemplateSize = 1024 * 1024 * 1024 // 1GB
				return config
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.configFunc()
			_, err := NewPromptManager(WithConfig(config))

			if tt.expectError && err == nil {
				t.Error("Expected configuration error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected configuration error: %v", err)
			}
		})
	}
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

func TestIntegration_EndToEndWorkflow(t *testing.T) {
	// Test a complete workflow from manager creation to formatted output
	config := DefaultConfig()
	config.EnableMetrics = true
	config.EnableTracing = true
	config.ValidateVariables = true

	manager, err := NewPromptManager(WithConfig(config))
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	ctx := context.Background()

	// Test 1: String template workflow
	t.Run("string_template_workflow", func(t *testing.T) {
		template, err := manager.NewStringTemplate("greeting", "Hello {{.name}}, welcome to {{.place}}!")
		if err != nil {
			t.Fatalf("Failed to create string template: %v", err)
		}

		inputs := map[string]interface{}{
			"name":  "Alice",
			"place": "Wonderland",
		}

		result, err := template.Format(ctx, inputs)
		if err != nil {
			t.Fatalf("Failed to format template: %v", err)
		}

		stringResult, ok := result.(internal.StringPromptValue)
		if !ok {
			t.Fatalf("Expected StringPromptValue, got %T", result)
		}

		expected := "Hello Alice, welcome to Wonderland!"
		if stringResult.ToString() != expected {
			t.Errorf("Expected %q, got %q", expected, stringResult.ToString())
		}

		// Test conversion to messages
		messages := stringResult.ToMessages()
		if len(messages) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(messages))
		}

		if messages[0].GetContent() != expected {
			t.Errorf("Message content = %q, want %q", messages[0].GetContent(), expected)
		}
	})

	// Test 2: Default adapter workflow
	t.Run("default_adapter_workflow", func(t *testing.T) {
		adapter, err := manager.NewDefaultAdapter("translator", "Translate '{{.text}}' to {{.language}}.", []string{"text", "language"})
		if err != nil {
			t.Fatalf("Failed to create default adapter: %v", err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		}

		inputs := map[string]interface{}{
			"text":     "Hello World",
			"language": "Spanish",
		}

		result, err := adapter.Format(ctx, inputs)
		if err != nil {
			t.Fatalf("Failed to format with adapter: %v", err)
		}

		stringResult, ok := result.(string)
		if !ok {
			t.Fatalf("Expected string result, got %T", result)
		}

		expected := "Translate 'Hello World' to Spanish."
		if stringResult != expected {
			t.Errorf("Expected %q, got %q", expected, stringResult)
		}
	})

	// Test 3: Chat adapter workflow
	t.Run("chat_adapter_workflow", func(t *testing.T) {
		adapter, err := manager.NewChatAdapter("chat_assistant",
			"You are a {{.role}} assistant.",
			"{{.question}}",
			[]string{"role", "question"})
		if err != nil {
			t.Fatalf("Failed to create chat adapter: %v", err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		}

		inputs := map[string]interface{}{
			"role":     "helpful",
			"question": "What is AI?",
		}

		result, err := adapter.Format(ctx, inputs)
		if err != nil {
			t.Fatalf("Failed to format with chat adapter: %v", err)
		}

		messages, ok := result.([]schema.Message)
		if !ok {
			t.Fatalf("Expected []schema.Message result, got %T", result)
		}

		if len(messages) != 2 {
			t.Fatalf("Expected 2 messages, got %d", len(messages))
		}

		// Check system message
		systemMsg := messages[0]
		if systemMsg.GetType() != "system" {
			t.Errorf("First message type = %s, want system", systemMsg.GetType())
		}
		if systemMsg.GetContent() != "You are a helpful assistant." {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			t.Errorf("System message = %q, want %q", systemMsg.GetContent(), "You are a helpful assistant.")
		}

		// Check user message
		userMsg := messages[1]
		if userMsg.GetType() != "user" {
			t.Errorf("Second message type = %s, want user", userMsg.GetType())
		}
		if userMsg.GetContent() != "What is AI?" {
			t.Errorf("User message = %q, want %q", userMsg.GetContent(), "What is AI?")
		}
	})

	// Test 4: Health check
	t.Run("health_check", func(t *testing.T) {
		err := manager.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})
}

func TestIntegration_ErrorPropagation(t *testing.T) {
	// Test that errors are properly propagated through the system
	config := DefaultConfig()
	config.ValidateVariables = true

	manager, err := NewPromptManager(WithConfig(config))
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	ctx := context.Background()

	// Test missing variable error propagation
	template, err := manager.NewStringTemplate("test", "Hello {{.name}}!")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	_, err = template.Format(ctx, map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing variable")
	}

	var promptErr *PromptError
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	if !errors.As(err, &promptErr) {
		t.Errorf("Expected PromptError, got %T", err)
	}

	if promptErr.Code != "variable_missing" {
		t.Errorf("Expected error code 'variable_missing', got %s", promptErr.Code)
	}
}

func TestIntegration_ConfigurationInheritance(t *testing.T) {
	// Test that configuration is properly inherited by components
	config := DefaultConfig()
	config.EnableMetrics = true
	config.EnableTracing = true
	config.ValidateVariables = true
	config.MaxTemplateSize = 1024

	manager, err := NewPromptManager(WithConfig(config))
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	// Verify configuration is accessible
	mgrConfig := manager.GetConfig()
	if mgrConfig.EnableMetrics != true {
		t.Error("Configuration not properly inherited: EnableMetrics")
	}
	if mgrConfig.EnableTracing != true {
		t.Error("Configuration not properly inherited: EnableTracing")
	}
	if mgrConfig.ValidateVariables != true {
		t.Error("Configuration not properly inherited: ValidateVariables")
	}
	if mgrConfig.MaxTemplateSize != 1024 {
		t.Error("Configuration not properly inherited: MaxTemplateSize")
	}
}

func TestConcurrency_TemplateFormatting(t *testing.T) {
	// Test concurrent template formatting
	template, err := NewStringPromptTemplate("concurrent_test", "Hello {{.name}} from goroutine {{.id}}!")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	numGoroutines := runtime.NumCPU() * 2
	numIterations := 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				inputs := map[string]interface{}{
					"name": "User" + string(rune(j%26+65)), // A-Z cycling
					"id":   goroutineID,
				}

				_, err := template.Format(ctx, inputs)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent formatting error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Total errors in concurrent test: %d", errorCount)
	}
}

func TestConcurrency_AdapterFormatting(t *testing.T) {
	// Test concurrent adapter formatting
	adapter, err := NewDefaultPromptAdapter("concurrent_adapter", "Process {{.data}} for user {{.user}}", []string{"data", "user"})
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	ctx := context.Background()
	numGoroutines := runtime.NumCPU() * 2
	numIterations := 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				inputs := map[string]interface{}{
					"data": "item" + string(rune(j%10+48)),           // 0-9 cycling
					"user": "user" + string(rune(goroutineID%26+65)), // A-Z cycling
				}

				_, err := adapter.Format(ctx, inputs)
				if err != nil {
					errors <- err
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent adapter error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Total errors in concurrent adapter test: %d", errorCount)
	}
}

func TestConcurrency_ManagerOperations(t *testing.T) {
	// Test concurrent operations on PromptManager
	config := DefaultConfig()
	config.EnableMetrics = true
	config.EnableTracing = true

	manager, err := NewPromptManager(WithConfig(config))
	if err != nil {
		t.Fatalf("Failed to create prompt manager: %v", err)
	}

	ctx := context.Background()
	numGoroutines := runtime.NumCPU()
	numIterations := 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				templateName := fmt.Sprintf("template_g%d_i%d", goroutineID, j)

				// Create template
				template, err := manager.NewStringTemplate(templateName, "Hello {{.name}}!")
				if err != nil {
					errors <- fmt.Errorf("create template: %w", err)
					continue
				}

				// Format with template
				inputs := map[string]interface{}{
					"name": fmt.Sprintf("User%d", goroutineID),
				}

				_, err = template.Format(ctx, inputs)
				if err != nil {
					errors <- fmt.Errorf("format template: %w", err)
					continue
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
				}

				// Health check occasionally
				if j%10 == 0 {
					err = manager.HealthCheck(ctx)
					if err != nil {
						errors <- fmt.Errorf("health check: %w", err)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent manager error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Total errors in concurrent manager test: %d", errorCount)
	}
}

func TestConcurrency_ChatAdapterFormatting(t *testing.T) {
	// Test concurrent chat adapter formatting
	adapter, err := NewChatPromptAdapter("concurrent_chat",
		"You are assistant {{.id}}.",
		"Question: {{.question}}",
		[]string{"id", "question"})
	if err != nil {
		t.Fatalf("Failed to create chat adapter: %v", err)
	}

	ctx := context.Background()
	numGoroutines := runtime.NumCPU() * 2
	numIterations := 50

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				inputs := map[string]interface{}{
					"id":       fmt.Sprintf("%d", goroutineID), // Convert to string as adapter expects strings
					"question": fmt.Sprintf("What is %d?", j),
				}

				result, err := adapter.Format(ctx, inputs)
				if err != nil {
					errors <- err
					continue
				}

				// Verify result structure
				messages, ok := result.([]schema.Message)
				if !ok {
					errors <- fmt.Errorf("expected []schema.Message, got %T", result)
					continue
				}

				if len(messages) != 2 {
					errors <- fmt.Errorf("expected 2 messages, got %d", len(messages))
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent chat adapter error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Total errors in concurrent chat adapter test: %d", errorCount)
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
