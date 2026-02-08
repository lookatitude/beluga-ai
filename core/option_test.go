package core

import (
	"testing"
)

// testConfig is a simple struct used as a target for option tests.
type testConfig struct {
	Name    string
	Value   int
	Enabled bool
}

func TestOptionFunc_Apply(t *testing.T) {
	tests := []struct {
		name     string
		opt      OptionFunc
		initial  testConfig
		expected testConfig
	}{
		{
			name: "set_name",
			opt: func(target any) {
				cfg := target.(*testConfig)
				cfg.Name = "hello"
			},
			initial:  testConfig{},
			expected: testConfig{Name: "hello"},
		},
		{
			name: "set_value",
			opt: func(target any) {
				cfg := target.(*testConfig)
				cfg.Value = 42
			},
			initial:  testConfig{},
			expected: testConfig{Value: 42},
		},
		{
			name: "toggle_enabled",
			opt: func(target any) {
				cfg := target.(*testConfig)
				cfg.Enabled = !cfg.Enabled
			},
			initial:  testConfig{Enabled: false},
			expected: testConfig{Enabled: true},
		},
		{
			name: "overwrite_existing",
			opt: func(target any) {
				cfg := target.(*testConfig)
				cfg.Name = "new"
			},
			initial:  testConfig{Name: "old"},
			expected: testConfig{Name: "new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.initial
			tt.opt.Apply(&cfg)
			if cfg != tt.expected {
				t.Errorf("Apply() result = %+v, want %+v", cfg, tt.expected)
			}
		})
	}
}

func TestOptionFunc_ImplementsOption(t *testing.T) {
	// Verify OptionFunc satisfies the Option interface at compile time.
	var _ Option = OptionFunc(func(_ any) {})
}

func TestApplyOptions(t *testing.T) {
	t.Run("multiple_options", func(t *testing.T) {
		opts := []Option{
			OptionFunc(func(target any) {
				cfg := target.(*testConfig)
				cfg.Name = "applied"
			}),
			OptionFunc(func(target any) {
				cfg := target.(*testConfig)
				cfg.Value = 99
			}),
			OptionFunc(func(target any) {
				cfg := target.(*testConfig)
				cfg.Enabled = true
			}),
		}

		cfg := testConfig{}
		ApplyOptions(&cfg, opts...)

		if cfg.Name != "applied" {
			t.Errorf("Name = %q, want %q", cfg.Name, "applied")
		}
		if cfg.Value != 99 {
			t.Errorf("Value = %d, want 99", cfg.Value)
		}
		if !cfg.Enabled {
			t.Error("Enabled = false, want true")
		}
	})

	t.Run("no_options", func(t *testing.T) {
		cfg := testConfig{Name: "unchanged", Value: 7, Enabled: true}
		ApplyOptions(&cfg)

		if cfg.Name != "unchanged" {
			t.Errorf("Name = %q, want %q", cfg.Name, "unchanged")
		}
		if cfg.Value != 7 {
			t.Errorf("Value = %d, want 7", cfg.Value)
		}
		if !cfg.Enabled {
			t.Error("Enabled = false, want true")
		}
	})

	t.Run("order_matters", func(t *testing.T) {
		opts := []Option{
			OptionFunc(func(target any) {
				cfg := target.(*testConfig)
				cfg.Name = "first"
			}),
			OptionFunc(func(target any) {
				cfg := target.(*testConfig)
				cfg.Name = "second"
			}),
		}

		cfg := testConfig{}
		ApplyOptions(&cfg, opts...)

		if cfg.Name != "second" {
			t.Errorf("Name = %q, want %q (last option wins)", cfg.Name, "second")
		}
	})

	t.Run("nil_target_no_panic_with_nil_safe_option", func(t *testing.T) {
		// An option that doesn't dereference target should work with nil.
		opt := OptionFunc(func(_ any) {
			// no-op
		})
		ApplyOptions(nil, opt)
	})

	t.Run("single_option", func(t *testing.T) {
		cfg := testConfig{}
		ApplyOptions(&cfg, OptionFunc(func(target any) {
			cfg := target.(*testConfig)
			cfg.Value = 1
		}))

		if cfg.Value != 1 {
			t.Errorf("Value = %d, want 1", cfg.Value)
		}
	})
}

func TestApplyOptions_Empty_Slice(t *testing.T) {
	cfg := testConfig{Name: "keep"}
	ApplyOptions(&cfg, []Option{}...)

	if cfg.Name != "keep" {
		t.Errorf("Name = %q, want %q", cfg.Name, "keep")
	}
}
