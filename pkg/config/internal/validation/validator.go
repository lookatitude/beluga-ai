package validation

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-playground/validator/v10"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

// ConfigValidator implements enhanced configuration validation with custom rules
type ConfigValidator struct {
	validator    *validator.Validate
	mu           sync.RWMutex
	customRules  map[string]CustomValidationRule
	crossRules   []CrossFieldRule
	strictMode   bool
}

// CustomValidationRule defines a custom validation function
type CustomValidationRule func(config *iface.Config) []iface.ValidationError

// CrossFieldRule defines a cross-field validation rule
type CrossFieldRule struct {
	Name      string
	Validator func(config *iface.Config) []iface.ValidationError
	Message   string
}

// ValidatorOptions configures the validator
type ValidatorOptions struct {
	EnableCustomRules     bool
	EnableCrossFieldRules bool
	StrictMode           bool
}

// NewConfigValidator creates a new enhanced configuration validator
func NewConfigValidator(options ValidatorOptions) (*ConfigValidator, error) {
	v := validator.New()
	
	// Register custom tag functions
	err := v.RegisterValidation("provider_exists", validateProviderExists)
	if err != nil {
		return nil, fmt.Errorf("failed to register provider_exists validation: %w", err)
	}

	cv := &ConfigValidator{
		validator:   v,
		customRules: make(map[string]CustomValidationRule),
		crossRules:  make([]CrossFieldRule, 0),
		strictMode:  options.StrictMode,
	}

	// Add default cross-field rules if enabled
	if options.EnableCrossFieldRules {
		cv.addDefaultCrossFieldRules()
	}

	return cv, nil
}

// ValidateConfig validates the entire configuration structure
func (cv *ConfigValidator) ValidateConfig(ctx context.Context, cfg *iface.Config) []iface.ValidationError {
	if cfg == nil {
		return []iface.ValidationError{
			{
				Field:   "config",
				Message: "Configuration cannot be nil",
			},
		}
	}

	var errors []iface.ValidationError

	// Check context cancellation
	select {
	case <-ctx.Done():
		return []iface.ValidationError{
			{
				Field:   "context",
				Message: "Validation cancelled by context",
			},
		}
	default:
	}

	// Use existing validation function from iface package
	if err := iface.ValidateConfig(cfg); err != nil {
		if validationErrors, ok := err.(iface.ValidationErrors); ok {
			for _, validationErr := range validationErrors {
				errors = append(errors, iface.ValidationError{
					Field:   validationErr.Field,
					Message: validationErr.Message,
				})
			}
		} else {
			errors = append(errors, iface.ValidationError{
				Field:   "config",
				Message: err.Error(),
			})
		}
	}

	// Apply custom rules
	cv.mu.RLock()
	customRules := make(map[string]CustomValidationRule)
	for k, v := range cv.customRules {
		customRules[k] = v
	}
	crossRules := make([]CrossFieldRule, len(cv.crossRules))
	copy(crossRules, cv.crossRules)
	cv.mu.RUnlock()

	for ruleName, rule := range customRules {
		ruleErrors := rule(cfg)
		for _, ruleError := range ruleErrors {
			ruleError.Field = fmt.Sprintf("%s[%s]", ruleError.Field, ruleName)
			errors = append(errors, ruleError)
		}
	}

	// Apply cross-field rules
	for _, crossRule := range crossRules {
		ruleErrors := crossRule.Validator(cfg)
		errors = append(errors, ruleErrors...)
	}

	return errors
}

// ValidateProvider validates a single provider configuration
// Note: Using interface{} since ProviderConfig is not defined in iface package
func (cv *ConfigValidator) ValidateProvider(ctx context.Context, provider interface{}) []iface.ValidationError {
	var errors []iface.ValidationError

	// Check context cancellation
	select {
	case <-ctx.Done():
		return []iface.ValidationError{
			{
				Field:   "context",
				Message: "Validation cancelled by context",
			},
		}
	default:
	}

	// Basic validation using struct tags
	err := cv.validator.Struct(provider)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, validationErr := range validationErrors {
				errors = append(errors, iface.ValidationError{
					Field:   validationErr.Field(),
					Message: validationErr.Error(),
				})
			}
		} else {
			errors = append(errors, iface.ValidationError{
				Field:   "provider",
				Message: err.Error(),
			})
		}
	}

	// Additional provider-specific validation
	// Note: Basic validation since provider structure is not strictly typed
	if provider == nil {
		errors = append(errors, iface.ValidationError{
			Field:   "provider",
			Message: "Provider cannot be nil",
		})
		return errors
	}

	// Skip detailed field validation since provider type is interface{}

	return errors
}

// AddCustomRule adds a custom validation rule
func (cv *ConfigValidator) AddCustomRule(name string, rule CustomValidationRule) error {
	cv.mu.Lock()
	defer cv.mu.Unlock()

	if name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}

	if rule == nil {
		return fmt.Errorf("rule function cannot be nil")
	}

	cv.customRules[name] = rule
	return nil
}

// addDefaultCrossFieldRules adds default cross-field validation rules
func (cv *ConfigValidator) addDefaultCrossFieldRules() {
	// Rule: Agent LLM provider references must exist
	cv.crossRules = append(cv.crossRules, CrossFieldRule{
		Name: "agent_llm_reference",
		Validator: func(config *iface.Config) []iface.ValidationError {
			var errors []iface.ValidationError
			
			// Build map of available LLM providers
			llmProviders := make(map[string]bool)
			for _, provider := range config.LLMProviders {
				llmProviders[provider.Name] = true
			}

			// Check agent references
			for i, agent := range config.Agents {
				if agent.LLMProviderName != "" && !llmProviders[agent.LLMProviderName] {
				errors = append(errors, iface.ValidationError{
					Field:   fmt.Sprintf("agents[%d].llm_provider_name", i),
					Message: fmt.Sprintf("Agent references non-existent LLM provider: %s", agent.LLMProviderName),
				})
				}
			}

			return errors
		},
		Message: "Agent LLM provider references must exist",
	})

	// Rule: Tool provider references must exist  
	cv.crossRules = append(cv.crossRules, CrossFieldRule{
		Name: "tool_provider_reference",
		Validator: func(config *iface.Config) []iface.ValidationError {
			var errors []iface.ValidationError

			// For tools, validate that referenced providers exist
			// (This would need to be implemented based on actual tool provider registry)
			for i, tool := range config.Tools {
				if tool.Provider == "" {
				errors = append(errors, iface.ValidationError{
					Field:   fmt.Sprintf("tools[%d].provider", i),
					Message: "Tool provider is required",
				})
				}
			}

			return errors
		},
		Message: "Tool provider references must be valid",
	})

	// Rule: Vector store provider references must exist
	cv.crossRules = append(cv.crossRules, CrossFieldRule{
		Name: "vectorstore_embedding_reference", 
		Validator: func(config *iface.Config) []iface.ValidationError {
			var errors []iface.ValidationError

			// Build map of available embedding providers
			embeddingProviders := make(map[string]bool)
			for _, provider := range config.EmbeddingProviders {
				embeddingProviders[provider.Name] = true
			}

		// Skip vector store embedding reference validation for now since schema structure may vary

			return errors
		},
		Message: "Vector store embedding provider references must exist",
	})
}

// validateProviderExists is a custom validator tag function
func validateProviderExists(fl validator.FieldLevel) bool {
	// This would integrate with the provider registry to check if provider exists
	providerName := fl.Field().String()
	return providerName != "" // Basic check for now
}
