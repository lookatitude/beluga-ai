package mcp

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// ElicitationType identifies the kind of user input to request.
type ElicitationType string

const (
	// ElicitationText requests free-form text input from the user.
	ElicitationText ElicitationType = "text"

	// ElicitationSelect requests the user to choose from a list of options.
	ElicitationSelect ElicitationType = "select"

	// ElicitationConfirm requests a yes/no confirmation from the user.
	ElicitationConfirm ElicitationType = "confirm"

	// ElicitationForm requests structured input via multiple fields.
	ElicitationForm ElicitationType = "form"
)

// ElicitationField describes a single field in a form elicitation.
type ElicitationField struct {
	// Name is the field identifier used in the response values map.
	Name string `json:"name"`

	// Label is the human-readable label displayed to the user.
	Label string `json:"label"`

	// Type is the field type (e.g., "text", "select", "confirm").
	Type ElicitationType `json:"type"`

	// Description provides additional context for the field.
	Description string `json:"description,omitempty"`

	// Required indicates whether the field must be filled in.
	Required bool `json:"required,omitempty"`

	// Options lists the available choices for select-type fields.
	Options []string `json:"options,omitempty"`

	// Default is the default value for the field.
	Default any `json:"default,omitempty"`
}

// ElicitationRequest describes a request for user input during an MCP operation.
type ElicitationRequest struct {
	// Type is the kind of elicitation being requested.
	Type ElicitationType `json:"type"`

	// Message is the prompt displayed to the user.
	Message string `json:"message"`

	// Options lists available choices for select-type elicitations.
	Options []string `json:"options,omitempty"`

	// Fields describes the form fields for form-type elicitations.
	Fields []ElicitationField `json:"fields,omitempty"`

	// Default is the default value for the response.
	Default any `json:"default,omitempty"`
}

// ElicitationResponse holds the user's response to an elicitation request.
type ElicitationResponse struct {
	// Accepted indicates whether the user accepted or dismissed the request.
	Accepted bool `json:"accepted"`

	// Value holds the user's response for text and confirm types.
	Value any `json:"value,omitempty"`

	// Selected holds the chosen option for select types.
	Selected string `json:"selected,omitempty"`

	// Values holds the field responses for form types.
	Values map[string]any `json:"values,omitempty"`
}

// ElicitationHandler handles user elicitation requests. Implementations
// connect to the appropriate user interface (CLI prompt, web UI, etc.).
type ElicitationHandler interface {
	// Elicit sends a request for user input and returns the response.
	// It must respect context cancellation for timeout handling.
	Elicit(ctx context.Context, req ElicitationRequest) (*ElicitationResponse, error)
}

// ValidateElicitationRequest checks that an ElicitationRequest is well-formed.
func ValidateElicitationRequest(req ElicitationRequest) error {
	if req.Message == "" {
		return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: message is required")
	}

	switch req.Type {
	case ElicitationText:
		// No additional validation needed.
	case ElicitationSelect:
		if len(req.Options) == 0 {
			return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: select type requires at least one option")
		}
	case ElicitationConfirm:
		// No additional validation needed.
	case ElicitationForm:
		if len(req.Fields) == 0 {
			return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: form type requires at least one field")
		}
		seen := make(map[string]struct{}, len(req.Fields))
		for _, f := range req.Fields {
			if f.Name == "" {
				return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: field name is required")
			}
			if _, exists := seen[f.Name]; exists {
				return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: duplicate field name %q", f.Name)
			}
			seen[f.Name] = struct{}{}
		}
	default:
		return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: unknown type %q", req.Type)
	}

	return nil
}

// ValidateElicitationResponse checks that an ElicitationResponse is valid
// for the given request. If the response is not accepted, no further
// validation is performed.
func ValidateElicitationResponse(req ElicitationRequest, resp ElicitationResponse) error {
	if !resp.Accepted {
		return nil
	}

	switch req.Type {
	case ElicitationSelect:
		if resp.Selected == "" {
			return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: select response requires a selected value")
		}
		found := false
		for _, opt := range req.Options {
			if opt == resp.Selected {
				found = true
				break
			}
		}
		if !found {
			return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: selected value %q is not among options", resp.Selected)
		}
	case ElicitationForm:
		if resp.Values == nil {
			return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: form response requires values")
		}
		for _, f := range req.Fields {
			if f.Required {
				if _, ok := resp.Values[f.Name]; !ok {
					return core.Errorf(core.ErrInvalidInput, "mcp/elicitation: required field %q is missing", f.Name)
				}
			}
		}
	case ElicitationText, ElicitationConfirm:
		// Accepted responses are valid as-is.
	}

	return nil
}
