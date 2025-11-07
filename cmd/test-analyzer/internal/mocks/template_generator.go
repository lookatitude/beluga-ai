package mocks

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
)

// TemplateGenerator generates mock templates with TODOs.
type TemplateGenerator interface {
	GenerateTemplate(ctx context.Context, componentName, interfaceName string, methods []MethodSignature, reason string) (string, error)
}

// templateGenerator implements TemplateGenerator.
type templateGenerator struct{}

// NewTemplateGenerator creates a new TemplateGenerator.
func NewTemplateGenerator() TemplateGenerator {
	return &templateGenerator{}
}

// GenerateTemplate implements TemplateGenerator.GenerateTemplate.
func (g *templateGenerator) GenerateTemplate(ctx context.Context, componentName, interfaceName string, methods []MethodSignature, reason string) (string, error) {
	tmpl := `package {{.Package}}

import (
	"github.com/stretchr/testify/mock"
)

// {{.StructName}} is a mock implementation of {{.InterfaceName}}.
// TODO: Complete this mock implementation
// Reason: {{.Reason}}
type {{.StructName}} struct {
	mock.Mock
	// TODO: Add configurable fields here
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
// TODO: Implement this method properly
func (m *{{$.StructName}}) {{.Name}}({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}} {{$p.Type}}{{end}}) {{if .Returns}}({{range $i, $r := .Returns}}{{if $i}}, {{end}}{{$r.Type}}{{end}}){{end}} {
	// TODO: Implement method logic
	args := m.Called({{range $i, $p := .Parameters}}{{if $i}}, {{end}}{{$p.Name}}{{end}})
	{{if .Returns}}// TODO: Handle return values properly
	return {{range $i, $r := .Returns}}{{if $i}}, {{end}}args.Get({{$i}}).({{$r.Type}}){{end}}{{end}}
}
{{end}}
`

	data := struct {
		Package        string
		StructName     string
		OptionsType    string
		ConstructorName string
		InterfaceName  string
		Methods        []MethodSignature
		Reason         string
	}{
		Package:         "testutils",
		StructName:      "AdvancedMock" + componentName,
		OptionsType:     "Mock" + componentName + "Option",
		ConstructorName: "NewAdvancedMock" + componentName,
		InterfaceName:   interfaceName,
		Methods:         methods,
		Reason:          reason,
	}

	t, err := template.New("mock_template").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return buf.String(), nil
}

