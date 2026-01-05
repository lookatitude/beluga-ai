package internal

// STSOption is a functional option for configuring S2S operations.
type STSOption func(*STSOptions)

// STSOptions represents options for S2S operations.
type STSOptions struct {
	// Language preference
	Language string

	// Voice preferences
	VoiceID string

	// Latency target (hint for provider optimization)
	LatencyTarget string // "low", "medium", "high"

	// Enable streaming mode
	EnableStreaming bool

	// Reasoning mode (built-in vs external agent)
	ReasoningMode string // "built-in", "external"

	// Additional provider-specific options
	ProviderOptions map[string]any
}

// WithLanguage sets the language preference.
func WithLanguage(language string) STSOption {
	return func(opts *STSOptions) {
		opts.Language = language
	}
}

// WithVoiceID sets the voice ID preference.
func WithVoiceID(voiceID string) STSOption {
	return func(opts *STSOptions) {
		opts.VoiceID = voiceID
	}
}

// WithLatencyTarget sets the latency target hint.
func WithLatencyTarget(target string) STSOption {
	return func(opts *STSOptions) {
		opts.LatencyTarget = target
	}
}

// WithEnableStreaming enables streaming mode.
func WithEnableStreaming(enable bool) STSOption {
	return func(opts *STSOptions) {
		opts.EnableStreaming = enable
	}
}

// WithReasoningMode sets the reasoning mode.
func WithReasoningMode(mode string) STSOption {
	return func(opts *STSOptions) {
		opts.ReasoningMode = mode
	}
}

// WithProviderOption sets a provider-specific option.
func WithProviderOption(key string, value any) STSOption {
	return func(opts *STSOptions) {
		if opts.ProviderOptions == nil {
			opts.ProviderOptions = make(map[string]any)
		}
		opts.ProviderOptions[key] = value
	}
}
