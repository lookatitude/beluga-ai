package mocks

import (
	"context"
	"time"
)

// MockImplementation represents a generated mock.
type MockImplementation struct {
	ComponentName            string
	InterfaceName            string
	Package                  string
	FilePath                 string
	Code                     string
	InterfaceMethods         []MethodSignature
	Status                   string
	RequiresManualCompletion bool
	GeneratedAt              time.Time
}

// MethodSignature represents a method signature.
type MethodSignature struct {
	Name       string
	Parameters []Parameter
	Returns    []Return
	Receiver   string
}

// Parameter represents a function parameter.
type Parameter struct {
	Name string
	Type string
}

// Return represents a return value.
type Return struct {
	Name string
	Type string
}

// MockPattern represents the structure of an existing mock pattern.
type MockPattern struct {
	StructName      string
	EmbeddedType    string
	OptionsType     string
	ConstructorName string
}

// MockGenerator is the interface for generating mock implementations.
type MockGenerator interface {
	// GenerateMock generates a mock implementation for an interface.
	GenerateMock(ctx context.Context, componentName, interfaceName, packagePath string) (*MockImplementation, error)

	// GenerateMockTemplate generates a mock template with TODOs for complex cases.
	GenerateMockTemplate(ctx context.Context, componentName, interfaceName, packagePath string, reason string) (*MockImplementation, error)

	// VerifyInterfaceCompatibility verifies that a mock implements the same interface as the actual implementation.
	VerifyInterfaceCompatibility(ctx context.Context, mock *MockImplementation, actualInterface string) (bool, error)
}

// generator implements the MockGenerator interface.
type generator struct {
	interfaceAnalyzer InterfaceAnalyzer
	patternExtractor  PatternExtractor
	codeGenerator     CodeGenerator
	templateGenerator TemplateGenerator
	validator         Validator
}

// NewGenerator creates a new MockGenerator instance.
func NewGenerator() MockGenerator {
	return &generator{
		interfaceAnalyzer: NewInterfaceAnalyzer(),
		patternExtractor:  NewPatternExtractor(),
		codeGenerator:     NewCodeGenerator(),
		templateGenerator: NewTemplateGenerator(),
		validator:         NewValidator(),
	}
}

// GenerateMock implements MockGenerator.GenerateMock.
func (g *generator) GenerateMock(ctx context.Context, componentName, interfaceName, packagePath string) (*MockImplementation, error) {
	// Analyze the interface
	methods, err := g.interfaceAnalyzer.AnalyzeInterface(ctx, interfaceName, packagePath)
	if err != nil {
		return nil, err
	}

	// Extract existing mock pattern
	pattern, err := g.patternExtractor.ExtractMockPattern(ctx, packagePath)
	if err != nil {
		// Use default pattern if extraction fails
		pattern = &MockPattern{
			StructName:      "AdvancedMock" + componentName,
			EmbeddedType:    "mock.Mock",
			OptionsType:     "Mock" + componentName + "Option",
			ConstructorName: "NewAdvancedMock" + componentName,
		}
	}

	// Generate mock code
	code, err := g.codeGenerator.GenerateMockCode(ctx, componentName, interfaceName, methods, pattern)
	if err != nil {
		// If code generation fails, generate template instead
		return g.GenerateMockTemplate(ctx, componentName, interfaceName, packagePath, "Code generation failed: "+err.Error())
	}

	return &MockImplementation{
		ComponentName:            componentName,
		InterfaceName:            interfaceName,
		Package:                  packagePath,
		FilePath:                 "test_utils.go",
		Code:                     code,
		InterfaceMethods:         methods,
		Status:                   "Complete",
		RequiresManualCompletion: false,
		GeneratedAt:              time.Now(),
	}, nil
}

// GenerateMockTemplate implements MockGenerator.GenerateMockTemplate.
func (g *generator) GenerateMockTemplate(ctx context.Context, componentName, interfaceName, packagePath string, reason string) (*MockImplementation, error) {
	// Analyze the interface
	methods, err := g.interfaceAnalyzer.AnalyzeInterface(ctx, interfaceName, packagePath)
	if err != nil {
		return nil, err
	}

	// Generate template code
	code, err := g.templateGenerator.GenerateTemplate(ctx, componentName, interfaceName, methods, reason)
	if err != nil {
		return nil, err
	}

	return &MockImplementation{
		ComponentName:            componentName,
		InterfaceName:            interfaceName,
		Package:                  packagePath,
		FilePath:                 "test_utils.go",
		Code:                     code,
		InterfaceMethods:         methods,
		Status:                   "Template",
		RequiresManualCompletion: true,
		GeneratedAt:              time.Now(),
	}, nil
}

// VerifyInterfaceCompatibility implements MockGenerator.VerifyInterfaceCompatibility.
func (g *generator) VerifyInterfaceCompatibility(ctx context.Context, mock *MockImplementation, actualInterface string) (bool, error) {
	return g.validator.VerifyInterfaceCompatibility(ctx, mock, actualInterface)
}
