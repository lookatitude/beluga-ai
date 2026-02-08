package llm

import (
	"testing"
)

func TestApplyOptions_Empty(t *testing.T) {
	opts := ApplyOptions()
	if opts.Temperature != nil {
		t.Error("expected nil Temperature")
	}
	if opts.MaxTokens != 0 {
		t.Error("expected zero MaxTokens")
	}
	if opts.TopP != nil {
		t.Error("expected nil TopP")
	}
	if len(opts.StopSequences) != 0 {
		t.Error("expected empty StopSequences")
	}
	if opts.Format != nil {
		t.Error("expected nil Format")
	}
	if opts.ToolChoice != "" {
		t.Error("expected empty ToolChoice")
	}
	if opts.SpecificTool != "" {
		t.Error("expected empty SpecificTool")
	}
	if opts.Metadata != nil {
		t.Error("expected nil Metadata")
	}
}

func TestWithTemperature(t *testing.T) {
	tests := []struct {
		name string
		temp float64
	}{
		{"zero", 0.0},
		{"low", 0.1},
		{"default", 0.7},
		{"high", 2.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithTemperature(tt.temp))
			if opts.Temperature == nil {
				t.Fatal("Temperature should not be nil")
			}
			if *opts.Temperature != tt.temp {
				t.Errorf("Temperature = %v, want %v", *opts.Temperature, tt.temp)
			}
		})
	}
}

func TestWithMaxTokens(t *testing.T) {
	tests := []struct {
		name   string
		tokens int
	}{
		{"small", 100},
		{"medium", 4096},
		{"large", 128000},
		{"zero", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithMaxTokens(tt.tokens))
			if opts.MaxTokens != tt.tokens {
				t.Errorf("MaxTokens = %d, want %d", opts.MaxTokens, tt.tokens)
			}
		})
	}
}

func TestWithTopP(t *testing.T) {
	tests := []struct {
		name string
		topP float64
	}{
		{"zero", 0.0},
		{"medium", 0.5},
		{"max", 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithTopP(tt.topP))
			if opts.TopP == nil {
				t.Fatal("TopP should not be nil")
			}
			if *opts.TopP != tt.topP {
				t.Errorf("TopP = %v, want %v", *opts.TopP, tt.topP)
			}
		})
	}
}

func TestWithStopSequences(t *testing.T) {
	tests := []struct {
		name string
		seqs []string
	}{
		{"single", []string{"\n"}},
		{"multiple", []string{"STOP", "END", "---"}},
		{"empty", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithStopSequences(tt.seqs...))
			if len(opts.StopSequences) != len(tt.seqs) {
				t.Fatalf("StopSequences len = %d, want %d", len(opts.StopSequences), len(tt.seqs))
			}
			for i, s := range opts.StopSequences {
				if s != tt.seqs[i] {
					t.Errorf("StopSequences[%d] = %q, want %q", i, s, tt.seqs[i])
				}
			}
		})
	}
}

func TestWithResponseFormat(t *testing.T) {
	tests := []struct {
		name   string
		format ResponseFormat
	}{
		{
			name:   "json_object",
			format: ResponseFormat{Type: "json_object"},
		},
		{
			name: "json_schema",
			format: ResponseFormat{
				Type:   "json_schema",
				Schema: map[string]any{"type": "object"},
			},
		},
		{
			name:   "text",
			format: ResponseFormat{Type: "text"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithResponseFormat(tt.format))
			if opts.Format == nil {
				t.Fatal("Format should not be nil")
			}
			if opts.Format.Type != tt.format.Type {
				t.Errorf("Format.Type = %q, want %q", opts.Format.Type, tt.format.Type)
			}
		})
	}
}

func TestWithToolChoice(t *testing.T) {
	tests := []struct {
		name   string
		choice ToolChoice
	}{
		{"auto", ToolChoiceAuto},
		{"none", ToolChoiceNone},
		{"required", ToolChoiceRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ApplyOptions(WithToolChoice(tt.choice))
			if opts.ToolChoice != tt.choice {
				t.Errorf("ToolChoice = %q, want %q", opts.ToolChoice, tt.choice)
			}
		})
	}
}

func TestWithSpecificTool(t *testing.T) {
	opts := ApplyOptions(WithSpecificTool("search"))
	if opts.SpecificTool != "search" {
		t.Errorf("SpecificTool = %q, want %q", opts.SpecificTool, "search")
	}
}

func TestWithMetadata(t *testing.T) {
	opts := ApplyOptions(WithMetadata(map[string]any{
		"key1": "value1",
		"key2": 42,
	}))
	if len(opts.Metadata) != 2 {
		t.Fatalf("Metadata len = %d, want 2", len(opts.Metadata))
	}
	if opts.Metadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] = %v, want %q", opts.Metadata["key1"], "value1")
	}
	if opts.Metadata["key2"] != 42 {
		t.Errorf("Metadata[key2] = %v, want 42", opts.Metadata["key2"])
	}
}

func TestWithMetadata_MergesMultiple(t *testing.T) {
	opts := ApplyOptions(
		WithMetadata(map[string]any{"a": 1}),
		WithMetadata(map[string]any{"b": 2}),
	)
	if len(opts.Metadata) != 2 {
		t.Fatalf("Metadata len = %d, want 2", len(opts.Metadata))
	}
	if opts.Metadata["a"] != 1 {
		t.Errorf("Metadata[a] = %v, want 1", opts.Metadata["a"])
	}
	if opts.Metadata["b"] != 2 {
		t.Errorf("Metadata[b] = %v, want 2", opts.Metadata["b"])
	}
}

func TestWithMetadata_OverwritesKey(t *testing.T) {
	opts := ApplyOptions(
		WithMetadata(map[string]any{"key": "old"}),
		WithMetadata(map[string]any{"key": "new"}),
	)
	if opts.Metadata["key"] != "new" {
		t.Errorf("Metadata[key] = %v, want %q", opts.Metadata["key"], "new")
	}
}

func TestApplyOptions_MultipleOptions(t *testing.T) {
	temp := 0.5
	opts := ApplyOptions(
		WithTemperature(temp),
		WithMaxTokens(1024),
		WithTopP(0.9),
		WithStopSequences("END"),
		WithToolChoice(ToolChoiceRequired),
		WithSpecificTool("calculate"),
		WithMetadata(map[string]any{"provider": "openai"}),
	)

	if opts.Temperature == nil || *opts.Temperature != temp {
		t.Errorf("Temperature = %v, want %v", opts.Temperature, temp)
	}
	if opts.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, want 1024", opts.MaxTokens)
	}
	if opts.TopP == nil || *opts.TopP != 0.9 {
		t.Errorf("TopP = %v, want 0.9", opts.TopP)
	}
	if len(opts.StopSequences) != 1 || opts.StopSequences[0] != "END" {
		t.Errorf("StopSequences = %v, want [END]", opts.StopSequences)
	}
	if opts.ToolChoice != ToolChoiceRequired {
		t.Errorf("ToolChoice = %q, want %q", opts.ToolChoice, ToolChoiceRequired)
	}
	if opts.SpecificTool != "calculate" {
		t.Errorf("SpecificTool = %q, want %q", opts.SpecificTool, "calculate")
	}
	if opts.Metadata["provider"] != "openai" {
		t.Errorf("Metadata[provider] = %v, want %q", opts.Metadata["provider"], "openai")
	}
}

func TestToolChoice_Constants(t *testing.T) {
	if ToolChoiceAuto != "auto" {
		t.Errorf("ToolChoiceAuto = %q, want %q", ToolChoiceAuto, "auto")
	}
	if ToolChoiceNone != "none" {
		t.Errorf("ToolChoiceNone = %q, want %q", ToolChoiceNone, "none")
	}
	if ToolChoiceRequired != "required" {
		t.Errorf("ToolChoiceRequired = %q, want %q", ToolChoiceRequired, "required")
	}
}

func TestWithTemperature_OverwritesPrevious(t *testing.T) {
	opts := ApplyOptions(
		WithTemperature(0.5),
		WithTemperature(1.5),
	)
	if opts.Temperature == nil || *opts.Temperature != 1.5 {
		t.Errorf("Temperature = %v, want 1.5 (last applied wins)", opts.Temperature)
	}
}
