package mocks

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// InterfaceAnalyzer analyzes interface definitions.
type InterfaceAnalyzer interface {
	AnalyzeInterface(ctx context.Context, interfaceName, packagePath string) ([]MethodSignature, error)
}

// interfaceAnalyzer implements InterfaceAnalyzer.
type interfaceAnalyzer struct{}

// NewInterfaceAnalyzer creates a new InterfaceAnalyzer.
func NewInterfaceAnalyzer() InterfaceAnalyzer {
	return &interfaceAnalyzer{}
}

// AnalyzeInterface implements InterfaceAnalyzer.AnalyzeInterface.
func (a *interfaceAnalyzer) AnalyzeInterface(ctx context.Context, interfaceName, packagePath string) ([]MethodSignature, error) {
	// Find the interface definition in the package
	interfaceDef, err := a.findInterface(ctx, interfaceName, packagePath)
	if err != nil {
		return nil, fmt.Errorf("finding interface: %w", err)
	}

	// Get the InterfaceType from TypeSpec
	interfaceType, ok := interfaceDef.Type.(*ast.InterfaceType)
	if !ok {
		return nil, fmt.Errorf("type is not an interface")
	}

	// Extract method signatures
	methods := make([]MethodSignature, 0)
	if interfaceType.Methods != nil {
		for _, method := range interfaceType.Methods.List {
			if len(method.Names) == 0 {
				continue
			}

			methodName := method.Names[0].Name
			signature := a.extractMethodSignature(method, methodName)
			methods = append(methods, signature)
		}
	}

	return methods, nil
}

// findInterface finds an interface definition in a package.
func (a *interfaceAnalyzer) findInterface(ctx context.Context, interfaceName, packagePath string) (*ast.TypeSpec, error) {
	fset := token.NewFileSet()

	// Walk package directory to find interface
	var interfaceDef *ast.TypeSpec
	err := filepath.Walk(packagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Parse file
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that don't parse
		}

		// Look for interface
		ast.Inspect(file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if ts.Name.Name == interfaceName {
					if _, ok := ts.Type.(*ast.InterfaceType); ok {
						interfaceDef = ts
						return false
					}
				}
			}
			return true
		})

		if interfaceDef != nil {
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if interfaceDef == nil {
		return nil, fmt.Errorf("interface %s not found in package %s", interfaceName, packagePath)
	}

	return interfaceDef, nil
}

// extractMethodSignature extracts method signature from AST.
func (a *interfaceAnalyzer) extractMethodSignature(field *ast.Field, methodName string) MethodSignature {
	signature := MethodSignature{
		Name: methodName,
	}

	// Extract function type
	if ft, ok := field.Type.(*ast.FuncType); ok {
		// Extract parameters
		if ft.Params != nil {
			for _, param := range ft.Params.List {
				paramType := a.typeToString(param.Type)
				for _, name := range param.Names {
					signature.Parameters = append(signature.Parameters, Parameter{
						Name: name.Name,
						Type: paramType,
					})
				}
				if len(param.Names) == 0 {
					signature.Parameters = append(signature.Parameters, Parameter{
						Name: "",
						Type: paramType,
					})
				}
			}
		}

		// Extract return values
		if ft.Results != nil {
			for _, result := range ft.Results.List {
				resultType := a.typeToString(result.Type)
				for _, name := range result.Names {
					signature.Returns = append(signature.Returns, Return{
						Name: name.Name,
						Type: resultType,
					})
				}
				if len(result.Names) == 0 {
					signature.Returns = append(signature.Returns, Return{
						Name: "",
						Type: resultType,
					})
				}
			}
		}
	}

	return signature
}

// typeToString converts an AST type expression to a string.
func (a *interfaceAnalyzer) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return a.typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + a.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + a.typeToString(t.Key) + "]" + a.typeToString(t.Value)
	case *ast.ChanType:
		return "chan " + a.typeToString(t.Value)
	case *ast.StarExpr:
		return "*" + a.typeToString(t.X)
	case *ast.FuncType:
		return "func(...)" // Simplified
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
	}
}
