# Variable Placeholder Syntax

Simple string replacement for template variables.

## Syntax
```
{{.variableName}}
```

## Replacement Logic
```go
placeholder := fmt.Sprintf("{{.%s}}", key)
formatted = strings.ReplaceAll(template, placeholder, value)
```
- Simple `strings.ReplaceAll` - no Go template engine
- All variables must be strings (type-checked)
- Variables declared in `inputVariables` slice at construction

## Validation
- `Config.ValidateVariables = true`: Check all variables exist before formatting
- `Config.StrictVariableCheck = true`: Also validates types upfront
- Missing variable → `ErrCodeVariableMissing`
- Wrong type → `ErrCodeVariableInvalid`

## Limitations
- No nested templates
- No conditionals or loops
- No default values
- Variables extracted via regex: `{{\\.(\w+)}}`
