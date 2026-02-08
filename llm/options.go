package llm

// GenerateOption is a functional option applied to GenerateOptions before
// a Generate or Stream call.
type GenerateOption func(*GenerateOptions)

// ToolChoice controls how the model selects tools.
type ToolChoice string

const (
	// ToolChoiceAuto lets the model decide whether to call a tool.
	ToolChoiceAuto ToolChoice = "auto"
	// ToolChoiceNone prevents the model from calling any tool.
	ToolChoiceNone ToolChoice = "none"
	// ToolChoiceRequired forces the model to call at least one tool.
	ToolChoiceRequired ToolChoice = "required"
)

// ResponseFormat controls the structure of the model's output.
type ResponseFormat struct {
	// Type is the format type: "text", "json_object", or "json_schema".
	Type string
	// Schema is the JSON Schema to enforce when Type is "json_schema".
	Schema map[string]any
}

// GenerateOptions collects all parameters that can be passed to Generate or
// Stream via functional options. Providers read from this struct to configure
// their API requests.
type GenerateOptions struct {
	// Temperature controls randomness (0.0–2.0). A nil pointer means unset.
	Temperature *float64
	// MaxTokens is the maximum number of tokens to generate. 0 means unset.
	MaxTokens int
	// TopP controls nucleus sampling (0.0–1.0). A nil pointer means unset.
	TopP *float64
	// StopSequences causes generation to stop when any of these strings is produced.
	StopSequences []string
	// Format specifies the desired output format (JSON, JSON Schema, etc.).
	Format *ResponseFormat
	// ToolChoice controls tool selection behaviour.
	ToolChoice ToolChoice
	// SpecificTool names a specific tool the model must call (used when
	// ToolChoice is not one of the standard values).
	SpecificTool string
	// Metadata holds provider-specific options that don't map to standard fields.
	Metadata map[string]any
}

// ApplyOptions creates a GenerateOptions from a list of functional options.
func ApplyOptions(opts ...GenerateOption) GenerateOptions {
	var o GenerateOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithTemperature sets the sampling temperature.
func WithTemperature(t float64) GenerateOption {
	return func(o *GenerateOptions) {
		o.Temperature = &t
	}
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(n int) GenerateOption {
	return func(o *GenerateOptions) {
		o.MaxTokens = n
	}
}

// WithTopP sets the nucleus sampling parameter.
func WithTopP(p float64) GenerateOption {
	return func(o *GenerateOptions) {
		o.TopP = &p
	}
}

// WithStopSequences sets the stop sequences.
func WithStopSequences(seqs ...string) GenerateOption {
	return func(o *GenerateOptions) {
		o.StopSequences = seqs
	}
}

// WithResponseFormat sets the response format (e.g. JSON mode or JSON Schema).
func WithResponseFormat(format ResponseFormat) GenerateOption {
	return func(o *GenerateOptions) {
		o.Format = &format
	}
}

// WithToolChoice sets the tool choice mode.
func WithToolChoice(choice ToolChoice) GenerateOption {
	return func(o *GenerateOptions) {
		o.ToolChoice = choice
	}
}

// WithSpecificTool forces the model to call the named tool.
func WithSpecificTool(name string) GenerateOption {
	return func(o *GenerateOptions) {
		o.SpecificTool = name
	}
}

// WithMetadata merges provider-specific key-value pairs into the options.
func WithMetadata(kv map[string]any) GenerateOption {
	return func(o *GenerateOptions) {
		if o.Metadata == nil {
			o.Metadata = make(map[string]any, len(kv))
		}
		for k, v := range kv {
			o.Metadata[k] = v
		}
	}
}
