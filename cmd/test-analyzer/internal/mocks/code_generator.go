package mocks

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
)

// CodeGenerator generates mock code using templates.
type CodeGenerator interface {
	GenerateMockCode(ctx context.Context, componentName, interfaceName string, methods []MethodSignature, pattern *MockPattern) (string, error)
}

// codeGenerator implements CodeGenerator.
type codeGenerator struct{}

// NewCodeGenerator creates a new CodeGenerator.
func NewCodeGenerator() CodeGenerator {
	return &codeGenerator{}
}

// GenerateMockCode implements CodeGenerator.GenerateMockCode.
func (g *codeGenerator) GenerateMockCode(ctx context.Context, componentName, interfaceName string, methods []MethodSignature, pattern *MockPattern) (string, error) {
	tmpl := `package {{.Package}}

import (
	"github.com/stretchr/testify/mock"
)

// {{.StructName}} is a mock implementation of {{.InterfaceName}}.
type {{.StructName}} struct {
	{{.EmbeddedType}}
	// Add configurable fields here
}

// {{.OptionsType}} is a functional option for {{.StructName}}.
type {{.OptionsType}} func(*{{.StructName}})

// {{.ConstructorName}} creates a new {{.StructName}}.
func {{.ConstructorName}}(options ...{{.OptionsType}}) *{{.StructName}} {
	m := &{{.StructName}}{}
	for _, opt := range options {
		opt(m)
	}
	return m
}

{{range .Methods}}
// {{.Name}} implements {{$.InterfaceName}}.{{.Name}}.
func (m *{{$.StructName}}) {{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}) {{if .Returns}}({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r.Type}}{{end}}){{end}} {
	args := m.Called({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}}{{end}})
	{{if .Returns}}return {{range $i, $r := .Returns}}{{if $i}}, {{end}}args.Get({{$i}}).({{$r.Type}}){{end}}{{end}}
}
{{end}}
`

	data := struct {
		Package         string
		StructName      string
		EmbeddedType    string
		OptionsType     string
		ConstructorName string
		InterfaceName   string
		Methods         []MethodSignature
	}{
		Package:         "testutils", // Will be determined from packagePath
		StructName:      pattern.StructName,
		EmbeddedType:    pattern.EmbeddedType,
		OptionsType:     pattern.OptionsType,
		ConstructorName: pattern.ConstructorName,
		InterfaceName:   interfaceName,
		Methods:         methods,
	}

	t, err := template.New("mock").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}
