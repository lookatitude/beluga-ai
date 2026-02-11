package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testConfig struct {
	Host    string `json:"host" default:"localhost" required:"true"`
	Port    int    `json:"port" default:"8080" min:"1" max:"65535"`
	Debug   bool   `json:"debug" default:"false"`
	Workers int    `json:"workers" default:"4" min:"1" max:"100"`
}

// requiredNoDefaultConfig has a required field with NO default — used to test
// that missing required fields trigger validation errors.
type requiredNoDefaultConfig struct {
	Name    string `json:"name" required:"true"`
	Timeout int    `json:"timeout" default:"30"`
}

type nestedConfig struct {
	App testConfig `json:"app"`
	Env string     `json:"env" default:"production"`
}

func TestLoad_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"host": "0.0.0.0", "port": 9090, "debug": true, "workers": 8}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load[testConfig](path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", cfg.Host, "0.0.0.0")
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9090)
	}
	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
	if cfg.Workers != 8 {
		t.Errorf("Workers = %d, want %d", cfg.Workers, 8)
	}
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Only provide host (required). Port and Workers should get defaults.
	data := `{"host": "myhost"}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load[testConfig](path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Host != "myhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "myhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d (default)", cfg.Port, 8080)
	}
	if cfg.Workers != 4 {
		t.Errorf("Workers = %d, want %d (default)", cfg.Workers, 4)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load[testConfig]("/nonexistent/config.json")
	if err == nil {
		t.Fatal("Load() expected error for missing file")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	if err := os.WriteFile(path, []byte(`{invalid json}`), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := Load[testConfig](path)
	if err == nil {
		t.Fatal("Load() expected error for invalid JSON")
	}
}

func TestLoad_UnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("host: localhost"), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := Load[testConfig](path)
	if err == nil {
		t.Fatal("Load() expected error for unsupported extension")
	}
}

func TestLoad_ValidationFailure_Required(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// requiredNoDefaultConfig has Name required with no default.
	// An empty JSON object means Name stays "" → required check fails.
	data := `{}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := Load[requiredNoDefaultConfig](path)
	if err == nil {
		t.Fatal("Load() expected validation error for missing required field")
	}
}

func TestLoad_ValidationFailure_Min(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Use a negative value to bypass default (which only applies to zero values).
	data := `{"host": "localhost", "port": -1}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := Load[testConfig](path)
	if err == nil {
		t.Fatal("Load() expected validation error for port < min")
	}
}

func TestLoad_ValidationFailure_Max(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"host": "localhost", "port": 99999}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := Load[testConfig](path)
	if err == nil {
		t.Fatal("Load() expected validation error for port > max")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     testConfig
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     testConfig{Host: "localhost", Port: 8080, Workers: 4},
			wantErr: false,
		},
		{
			name:    "missing_required",
			cfg:     testConfig{Host: "", Port: 8080, Workers: 4},
			wantErr: true,
		},
		{
			name:    "port_below_min",
			cfg:     testConfig{Host: "localhost", Port: 0, Workers: 4},
			wantErr: true,
		},
		{
			name:    "port_above_max",
			cfg:     testConfig{Host: "localhost", Port: 70000, Workers: 4},
			wantErr: true,
		},
		{
			name:    "workers_below_min",
			cfg:     testConfig{Host: "localhost", Port: 80, Workers: 0},
			wantErr: true,
		},
		{
			name:    "workers_above_max",
			cfg:     testConfig{Host: "localhost", Port: 80, Workers: 200},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_NonStruct(t *testing.T) {
	n := 42
	err := Validate(&n)
	if err == nil {
		t.Error("Validate() expected error for non-struct")
	}
}

func TestMergeEnv(t *testing.T) {
	cfg := testConfig{Host: "original", Port: 3000, Workers: 2}

	t.Setenv("TEST_HOST", "fromenv")
	t.Setenv("TEST_PORT", "9999")

	err := MergeEnv(&cfg, "TEST")
	if err != nil {
		t.Fatalf("MergeEnv() error = %v", err)
	}

	if cfg.Host != "fromenv" {
		t.Errorf("Host = %q, want %q", cfg.Host, "fromenv")
	}
	if cfg.Port != 9999 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9999)
	}
	// Workers should remain unchanged (no env var set).
	if cfg.Workers != 2 {
		t.Errorf("Workers = %d, want %d (unchanged)", cfg.Workers, 2)
	}
}

func TestMergeEnv_Bool(t *testing.T) {
	cfg := testConfig{Host: "h", Port: 80, Workers: 1}

	t.Setenv("TEST_DEBUG", "true")

	if err := MergeEnv(&cfg, "TEST"); err != nil {
		t.Fatalf("MergeEnv() error = %v", err)
	}
	if !cfg.Debug {
		t.Error("Debug = false, want true")
	}
}

func TestMergeEnv_NonPointer(t *testing.T) {
	cfg := testConfig{}
	err := MergeEnv(cfg, "TEST")
	if err == nil {
		t.Error("MergeEnv() expected error for non-pointer argument")
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("APP_HOST", "envhost")
	t.Setenv("APP_PORT", "4000")
	t.Setenv("APP_WORKERS", "16")

	cfg, err := LoadFromEnv[testConfig]("APP")
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Host != "envhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "envhost")
	}
	if cfg.Port != 4000 {
		t.Errorf("Port = %d, want %d", cfg.Port, 4000)
	}
	if cfg.Workers != 16 {
		t.Errorf("Workers = %d, want %d", cfg.Workers, 16)
	}
}

func TestLoadFromEnv_DefaultsApplied(t *testing.T) {
	// No env vars set; defaults should be used.
	// But host is required, so set it.
	t.Setenv("DEF_HOST", "required-host")

	cfg, err := LoadFromEnv[testConfig]("DEF")
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d (default)", cfg.Port, 8080)
	}
	if cfg.Workers != 4 {
		t.Errorf("Workers = %d, want %d (default)", cfg.Workers, 4)
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := &ValidationError{
		Field:   "port",
		Message: "value 0 is less than minimum 1",
	}
	got := ve.Error()
	want := `config: validation failed for "port": value 0 is less than minimum 1`
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestToEnvName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Host", "HOST"},
		{"Port", "PORT"},
		{"BaseURL", "BASE_URL"}, // Consecutive uppercase kept together (acronym).
		{"MaxRetryCount", "MAX_RETRY_COUNT"},
		{"simple", "SIMPLE"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toEnvName(tt.input)
			if got != tt.want {
				t.Errorf("toEnvName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// isValidationError checks if err's chain contains a *ValidationError.
func isValidationError(err error, target **ValidationError) bool {
	if err == nil {
		return false
	}
	// Try to unwrap to find ValidationError.
	if ve, ok := err.(*ValidationError); ok {
		*target = ve
		return true
	}
	return false
}

// Types for testing additional field types and edge cases.

type floatConfig struct {
	Rate    float64 `json:"rate" default:"0.5" min:"0.0" max:"1.0"`
	Percent float32 `json:"percent" default:"50.0"`
}

type uintConfig struct {
	Count uint   `json:"count" default:"10"`
	Small uint8  `json:"small" default:"5"`
	Big   uint64 `json:"big" default:"1000"`
}

type boolConfig struct {
	Enabled bool `json:"enabled" default:"true"`
}

type durationConfig struct {
	Timeout time.Duration `json:"timeout" default:"5000000000"` // 5s in nanoseconds
}

type unsupportedFieldConfig struct {
	Name  string   `json:"name"`
	Slice []string `json:"slice" default:"a,b,c"` // unsupported type for setFieldFromString
}

type nestedRequiredConfig struct {
	Outer string `json:"outer" required:"true"`
	Inner struct {
		Value string `json:"value" required:"true"`
	} `json:"inner"`
}

type invalidMinMaxConfig struct {
	Count int `json:"count" min:"notanumber"`
}

type invalidMaxConfig struct {
	Count int `json:"count" max:"notanumber"`
}

type uintBoundsConfig struct {
	Count uint `json:"count" min:"1" max:"100"`
}

type floatBoundsConfig struct {
	Rate float64 `json:"rate" min:"0.0" max:"1.0"`
}

func TestSetFieldFromString_Float(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"rate": 0.75, "percent": 33.3}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load[floatConfig](path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Rate != 0.75 {
		t.Errorf("Rate = %f, want %f", cfg.Rate, 0.75)
	}
}

func TestSetFieldFromString_Float_Default(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load[floatConfig](path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Rate != 0.5 {
		t.Errorf("Rate = %f, want %f (default)", cfg.Rate, 0.5)
	}
}

func TestSetFieldFromString_Uint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Empty JSON should get defaults.
	data := `{}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load[uintConfig](path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Count != 10 {
		t.Errorf("Count = %d, want %d (default)", cfg.Count, 10)
	}
	if cfg.Small != 5 {
		t.Errorf("Small = %d, want %d (default)", cfg.Small, 5)
	}
	if cfg.Big != 1000 {
		t.Errorf("Big = %d, want %d (default)", cfg.Big, 1000)
	}
}

func TestSetFieldFromString_Bool_Default(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load[boolConfig](path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Enabled {
		t.Error("Enabled = false, want true (default)")
	}
}

func TestSetFieldFromString_Duration_Default(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load[durationConfig](path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want %v (default)", cfg.Timeout, 5*time.Second)
	}
}

func TestLoadFromEnv_Float(t *testing.T) {
	t.Setenv("FLT_RATE", "0.8")

	cfg, err := LoadFromEnv[floatConfig]("FLT")
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.Rate != 0.8 {
		t.Errorf("Rate = %f, want %f", cfg.Rate, 0.8)
	}
}

func TestLoadFromEnv_Uint(t *testing.T) {
	t.Setenv("UINT_COUNT", "42")
	t.Setenv("UINT_BIG", "999")

	cfg, err := LoadFromEnv[uintConfig]("UINT")
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.Count != 42 {
		t.Errorf("Count = %d, want %d", cfg.Count, 42)
	}
	if cfg.Big != 999 {
		t.Errorf("Big = %d, want %d", cfg.Big, 999)
	}
}

func TestLoadFromEnv_Duration(t *testing.T) {
	t.Setenv("DUR_TIMEOUT", "10000000000") // 10s

	cfg, err := LoadFromEnv[durationConfig]("DUR")
	if err != nil {
		t.Fatalf("LoadFromEnv() error = %v", err)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 10*time.Second)
	}
}

func TestMergeEnv_InvalidIntValue(t *testing.T) {
	cfg := testConfig{Host: "h", Port: 80, Workers: 1}
	t.Setenv("BAD_PORT", "not-a-number")

	err := MergeEnv(&cfg, "BAD")
	if err == nil {
		t.Error("expected error for invalid int env value")
	}
}

func TestMergeEnv_InvalidBoolValue(t *testing.T) {
	cfg := testConfig{Host: "h", Port: 80, Workers: 1}
	t.Setenv("BAD_DEBUG", "not-a-bool")

	err := MergeEnv(&cfg, "BAD")
	if err == nil {
		t.Error("expected error for invalid bool env value")
	}
}

func TestMergeEnv_InvalidUintValue(t *testing.T) {
	cfg := uintConfig{Count: 1}
	t.Setenv("BAD_COUNT", "not-a-number")

	err := MergeEnv(&cfg, "BAD")
	if err == nil {
		t.Error("expected error for invalid uint env value")
	}
}

func TestMergeEnv_InvalidFloatValue(t *testing.T) {
	cfg := floatConfig{Rate: 0.5}
	t.Setenv("BAD_RATE", "not-a-float")

	err := MergeEnv(&cfg, "BAD")
	if err == nil {
		t.Error("expected error for invalid float env value")
	}
}

func TestMergeEnv_UnsupportedType(t *testing.T) {
	cfg := unsupportedFieldConfig{Name: "test"}
	t.Setenv("BAD_SLICE", "a,b,c")

	err := MergeEnv(&cfg, "BAD")
	if err == nil {
		t.Error("expected error for unsupported slice type")
	}
}

func TestMergeEnv_NestedStruct(t *testing.T) {
	cfg := nestedConfig{
		App: testConfig{Host: "original", Port: 80, Workers: 1},
		Env: "dev",
	}

	t.Setenv("NEST_APP_HOST", "nested-host")
	t.Setenv("NEST_ENV", "production")

	err := MergeEnv(&cfg, "NEST")
	if err != nil {
		t.Fatalf("MergeEnv() error = %v", err)
	}
	if cfg.App.Host != "nested-host" {
		t.Errorf("App.Host = %q, want %q", cfg.App.Host, "nested-host")
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want %q", cfg.Env, "production")
	}
}

func TestLoad_NestedDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"app": {"host": "myhost"}}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	cfg, err := Load[nestedConfig](path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Host != "myhost" {
		t.Errorf("App.Host = %q, want %q", cfg.App.Host, "myhost")
	}
	if cfg.App.Port != 8080 {
		t.Errorf("App.Port = %d, want %d (default)", cfg.App.Port, 8080)
	}
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want %q (default)", cfg.Env, "production")
	}
}

func TestLoad_NestedRequired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Missing inner required value.
	data := `{"outer": "provided"}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	_, err := Load[nestedRequiredConfig](path)
	if err == nil {
		t.Fatal("expected validation error for missing nested required field")
	}
}

func TestValidate_UintBounds(t *testing.T) {
	tests := []struct {
		name    string
		cfg     uintBoundsConfig
		wantErr bool
	}{
		{"valid", uintBoundsConfig{Count: 50}, false},
		{"below min", uintBoundsConfig{Count: 0}, true},
		{"above max", uintBoundsConfig{Count: 200}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_FloatBounds(t *testing.T) {
	tests := []struct {
		name    string
		cfg     floatBoundsConfig
		wantErr bool
	}{
		{"valid", floatBoundsConfig{Rate: 0.5}, false},
		{"below min", floatBoundsConfig{Rate: -0.1}, true},
		{"above max", floatBoundsConfig{Rate: 1.5}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_InvalidMinTag(t *testing.T) {
	err := Validate(&invalidMinMaxConfig{Count: 5})
	if err == nil {
		t.Fatal("expected error for invalid min tag")
	}
}

func TestValidate_InvalidMaxTag(t *testing.T) {
	err := Validate(&invalidMaxConfig{Count: 5})
	if err == nil {
		t.Fatal("expected error for invalid max tag")
	}
}

func TestLoadFromEnv_ValidationFailure(t *testing.T) {
	// Set a value that violates validation.
	t.Setenv("VFAIL_HOST", "h")
	t.Setenv("VFAIL_PORT", "0") // Below min:1

	_, err := LoadFromEnv[testConfig]("VFAIL")
	if err == nil {
		t.Fatal("expected validation error for port below min")
	}
}

func TestLoad_RequiredWithZeroProvided(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Provide host as empty string explicitly — should satisfy "provided" check.
	data := `{"host": "", "port": 8080, "workers": 4}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	// host is required but explicitly provided as "" — validateRequired should
	// consider it provided (it was in the JSON). However, Validate will still
	// fail because required:"true" checks zero-valued.
	_, err := Load[testConfig](path)
	// The second-phase Validate sees empty host and fails.
	if err == nil {
		t.Fatal("expected validation error for empty host")
	}
}

func TestToEnvName_AdditionalCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a", "A"},
		{"ABC", "ABC"},
		{"URLParser", "URL_PARSER"},
		{"myHTTPClient", "MY_HTTP_CLIENT"},
		{"ID", "ID"},
		{"userID", "USER_ID"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toEnvName(tt.input)
			if got != tt.want {
				t.Errorf("toEnvName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestApplyDefaults_NonStruct(t *testing.T) {
	n := 42
	applyDefaults(&n) // Should not panic for non-struct.
}

func TestApplyDefaultsSelective_NonStruct(t *testing.T) {
	n := 42
	applyDefaultsSelective(&n, nil) // Should not panic for non-struct.
}

func TestValidateRequired_NonStruct(t *testing.T) {
	n := 42
	err := validateRequired(&n, nil)
	if err != nil {
		t.Errorf("expected nil for non-struct, got %v", err)
	}
}

func TestMergeEnv_InvalidDurationValue(t *testing.T) {
	cfg := durationConfig{}
	t.Setenv("DUR_TIMEOUT", "not-a-number")

	err := MergeEnv(&cfg, "DUR")
	if err == nil {
		t.Error("expected error for invalid duration env value")
	}
}
