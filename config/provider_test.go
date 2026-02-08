package config

import (
	"testing"
	"time"
)

func TestProviderConfig_Fields(t *testing.T) {
	cfg := ProviderConfig{
		Provider: "openai",
		APIKey:   "sk-test-key",
		Model:    "gpt-4o",
		BaseURL:  "https://api.openai.com/v1",
		Timeout:  30 * time.Second,
		Options: map[string]any{
			"temperature": 0.7,
		},
	}

	if cfg.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", cfg.Provider, "openai")
	}
	if cfg.APIKey != "sk-test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test-key")
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-4o")
	}
	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.openai.com/v1")
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 30*time.Second)
	}
}

func TestGetOption_Found(t *testing.T) {
	cfg := ProviderConfig{
		Options: map[string]any{
			"temperature": 0.7,
			"max_tokens":  4096,
			"stream":      true,
		},
	}

	tests := []struct {
		name string
		key  string
		run  func(t *testing.T)
	}{
		{
			name: "float64",
			key:  "temperature",
			run: func(t *testing.T) {
				v, ok := GetOption[float64](cfg, "temperature")
				if !ok {
					t.Fatal("expected ok=true")
				}
				if v != 0.7 {
					t.Errorf("value = %v, want 0.7", v)
				}
			},
		},
		{
			name: "int",
			key:  "max_tokens",
			run: func(t *testing.T) {
				v, ok := GetOption[int](cfg, "max_tokens")
				if !ok {
					t.Fatal("expected ok=true")
				}
				if v != 4096 {
					t.Errorf("value = %v, want 4096", v)
				}
			},
		},
		{
			name: "bool",
			key:  "stream",
			run: func(t *testing.T) {
				v, ok := GetOption[bool](cfg, "stream")
				if !ok {
					t.Fatal("expected ok=true")
				}
				if !v {
					t.Error("expected true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestGetOption_NotFound(t *testing.T) {
	cfg := ProviderConfig{
		Options: map[string]any{
			"temperature": 0.7,
		},
	}

	v, ok := GetOption[float64](cfg, "nonexistent")
	if ok {
		t.Error("expected ok=false for missing key")
	}
	if v != 0 {
		t.Errorf("expected zero value, got %v", v)
	}
}

func TestGetOption_TypeMismatch(t *testing.T) {
	cfg := ProviderConfig{
		Options: map[string]any{
			"temperature": "not a float",
		},
	}

	v, ok := GetOption[float64](cfg, "temperature")
	if ok {
		t.Error("expected ok=false for type mismatch")
	}
	if v != 0 {
		t.Errorf("expected zero value, got %v", v)
	}
}

func TestGetOption_NilOptions(t *testing.T) {
	cfg := ProviderConfig{
		Options: nil,
	}

	v, ok := GetOption[string](cfg, "any_key")
	if ok {
		t.Error("expected ok=false for nil Options")
	}
	if v != "" {
		t.Errorf("expected empty string, got %q", v)
	}
}

func TestGetOption_EmptyOptions(t *testing.T) {
	cfg := ProviderConfig{
		Options: map[string]any{},
	}

	v, ok := GetOption[int](cfg, "key")
	if ok {
		t.Error("expected ok=false for empty Options")
	}
	if v != 0 {
		t.Errorf("expected zero value, got %v", v)
	}
}

func TestGetOption_StringValue(t *testing.T) {
	cfg := ProviderConfig{
		Options: map[string]any{
			"model": "gpt-4o",
		},
	}

	v, ok := GetOption[string](cfg, "model")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if v != "gpt-4o" {
		t.Errorf("value = %q, want %q", v, "gpt-4o")
	}
}

func TestGetOption_MapValue(t *testing.T) {
	inner := map[string]any{"nested": true}
	cfg := ProviderConfig{
		Options: map[string]any{
			"complex": inner,
		},
	}

	v, ok := GetOption[map[string]any](cfg, "complex")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if v["nested"] != true {
		t.Errorf("nested value = %v, want true", v["nested"])
	}
}

func TestProviderConfig_ZeroValue(t *testing.T) {
	var cfg ProviderConfig
	if cfg.Provider != "" {
		t.Error("expected empty Provider")
	}
	if cfg.APIKey != "" {
		t.Error("expected empty APIKey")
	}
	if cfg.Model != "" {
		t.Error("expected empty Model")
	}
	if cfg.BaseURL != "" {
		t.Error("expected empty BaseURL")
	}
	if cfg.Timeout != 0 {
		t.Error("expected zero Timeout")
	}
	if cfg.Options != nil {
		t.Error("expected nil Options")
	}
}
