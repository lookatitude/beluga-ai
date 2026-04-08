package contract

import (
	"fmt"
	"sort"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
)

// CompatibilityError describes a single incompatibility between two agent
// contracts. The SourceAgent and TargetAgent fields identify the agents involved,
// while Field and Reason describe the specific mismatch.
type CompatibilityError struct {
	// SourceAgent is the ID of the agent producing output.
	SourceAgent string
	// TargetAgent is the ID of the agent consuming input.
	TargetAgent string
	// Field is the schema field path where the incompatibility was found.
	Field string
	// Reason is a human-readable explanation of the incompatibility.
	Reason string
}

// Error returns a formatted string for the compatibility error.
func (e CompatibilityError) Error() string {
	return fmt.Sprintf("%s -> %s: field %q: %s", e.SourceAgent, e.TargetAgent, e.Field, e.Reason)
}

// CheckCompatibility checks whether the output schema of contract "from"
// is compatible with the input schema of contract "to". Returns a list of
// incompatibilities. An empty list means the contracts are fully compatible.
//
// Rules:
//   - A nil schema is treated as a wildcard (always compatible).
//   - All required fields in the target input must exist in the source output.
//   - Types must match, with int->number widening allowed.
func CheckCompatibility(from, to *schema.Contract) []CompatibilityError {
	if from == nil || to == nil {
		return nil
	}

	fromName := from.Name
	toName := to.Name

	return checkSchemaCompat(from.OutputSchema, to.InputSchema, fromName, toName)
}

// CheckPipelineCompatibility checks sequential compatibility for a list of
// agents. Each agent's output contract is checked against the next agent's
// input contract. Agents without contracts are skipped (wildcard behavior).
func CheckPipelineCompatibility(agents []agent.Agent) []CompatibilityError {
	if len(agents) < 2 {
		return nil
	}

	var errs []CompatibilityError
	for i := 0; i < len(agents)-1; i++ {
		from := ContractOf(agents[i])
		to := ContractOf(agents[i+1])
		errs = append(errs, CheckCompatibility(from, to)...)
	}
	return errs
}

// checkSchemaCompat checks whether outputSchema can satisfy inputSchema.
func checkSchemaCompat(outputSchema, inputSchema map[string]any, sourceName, targetName string) []CompatibilityError {
	// nil = wildcard.
	if outputSchema == nil || inputSchema == nil {
		return nil
	}

	var errs []CompatibilityError

	// Check type compatibility at the top level.
	outType, _ := outputSchema["type"].(string)
	inType, _ := inputSchema["type"].(string)
	if inType != "" && outType != "" && !typesCompatible(outType, inType) {
		errs = append(errs, CompatibilityError{
			SourceAgent: sourceName,
			TargetAgent: targetName,
			Field:       "(root)",
			Reason:      fmt.Sprintf("type mismatch: source produces %q, target expects %q", outType, inType),
		})
		return errs
	}

	// For object types, check property compatibility.
	if inType == "object" || (inType == "" && inputSchema["properties"] != nil) {
		errs = append(errs, checkPropertyCompat(outputSchema, inputSchema, sourceName, targetName)...)
	}

	return errs
}

// checkPropertyCompat verifies required input properties exist in the output.
func checkPropertyCompat(outputSchema, inputSchema map[string]any, sourceName, targetName string) []CompatibilityError {
	var errs []CompatibilityError

	outProps, _ := outputSchema["properties"].(map[string]any)
	inProps, _ := inputSchema["properties"].(map[string]any)

	// Check required fields.
	required := extractRequired(inputSchema)
	for _, fieldName := range required {
		if outProps == nil {
			errs = append(errs, CompatibilityError{
				SourceAgent: sourceName,
				TargetAgent: targetName,
				Field:       fieldName,
				Reason:      "required by target but source has no properties",
			})
			continue
		}
		outPropRaw, exists := outProps[fieldName]
		if !exists {
			errs = append(errs, CompatibilityError{
				SourceAgent: sourceName,
				TargetAgent: targetName,
				Field:       fieldName,
				Reason:      "required by target but missing from source output",
			})
			continue
		}

		// Check type compatibility for the field.
		if inProps != nil {
			inPropRaw, inExists := inProps[fieldName]
			if inExists {
				outProp, _ := outPropRaw.(map[string]any)
				inProp, _ := inPropRaw.(map[string]any)
				if outProp != nil && inProp != nil {
					outFieldType, _ := outProp["type"].(string)
					inFieldType, _ := inProp["type"].(string)
					if outFieldType != "" && inFieldType != "" && !typesCompatible(outFieldType, inFieldType) {
						errs = append(errs, CompatibilityError{
							SourceAgent: sourceName,
							TargetAgent: targetName,
							Field:       fieldName,
							Reason:      fmt.Sprintf("type mismatch: source produces %q, target expects %q", outFieldType, inFieldType),
						})
					}
				}
			}
		}
	}

	return errs
}

// extractRequired extracts the "required" array from a JSON Schema.
func extractRequired(sch map[string]any) []string {
	reqRaw, ok := sch["required"].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(reqRaw))
	for _, r := range reqRaw {
		if s, ok := r.(string); ok {
			result = append(result, s)
		}
	}
	sort.Strings(result)
	return result
}

// typesCompatible checks if sourceType can satisfy targetType.
// int -> number widening is allowed.
func typesCompatible(sourceType, targetType string) bool {
	if sourceType == targetType {
		return true
	}
	// int -> number widening.
	if sourceType == "integer" && targetType == "number" {
		return true
	}
	return false
}
