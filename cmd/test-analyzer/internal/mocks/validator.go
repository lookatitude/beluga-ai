package mocks

import (
	"context"
	"fmt"
	"reflect"
)

// Validator verifies interface compatibility.
type Validator interface {
	VerifyInterfaceCompatibility(ctx context.Context, mock *MockImplementation, actualInterface string) (bool, error)
}

// validator implements Validator.
type validator struct{}

// NewValidator creates a new Validator.
func NewValidator() Validator {
	return &validator{}
}

// VerifyInterfaceCompatibility implements Validator.VerifyInterfaceCompatibility.
func (v *validator) VerifyInterfaceCompatibility(ctx context.Context, mock *MockImplementation, actualInterface string) (bool, error) {
	// This is a simplified validation
	// Full implementation would:
	// 1. Parse the actual interface definition
	// 2. Compare method signatures
	// 3. Use reflection to verify at runtime
	
	// For now, we check that the mock has methods for all interface methods
	if len(mock.InterfaceMethods) == 0 {
		return false, fmt.Errorf("mock has no methods")
	}

	// Basic validation: check that mock code contains all method names
	for _, method := range mock.InterfaceMethods {
		if !containsMethod(mock.Code, method.Name) {
			return false, fmt.Errorf("mock missing method: %s", method.Name)
		}
	}

	return true, nil
}

// containsMethod checks if code contains a method definition.
func containsMethod(code, methodName string) bool {
	// Simple string search - in production, would use AST analysis
	return contains(code, "func (m *") && contains(code, methodName+"(")
}

// contains is a simple string contains check.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && 
			(s[:len(substr)] == substr || 
			 s[len(s)-len(substr):] == substr ||
			 containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// verifyWithReflection uses reflection to verify interface compatibility.
func verifyWithReflection(mockType reflect.Type, interfaceType reflect.Type) bool {
	// This would be used in a full implementation
	// For now, return true as placeholder
	return true
}

