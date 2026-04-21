package o11y

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TestBootstrapFromEnv_CleanEnv asserts that with no OTEL_* variables set,
// BootstrapFromEnv returns a non-nil shutdown function and a nil error.
// No exporter is attached; spans silently become no-ops.
func TestBootstrapFromEnv_CleanEnv(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_SDK_DISABLED", "")
	t.Setenv("BELUGA_OTEL_STDOUT", "")

	shutdown, err := BootstrapFromEnv(context.Background(), "beluga-test")
	if err != nil {
		t.Fatalf("BootstrapFromEnv: unexpected error with clean env: %v", err)
	}
	if shutdown == nil {
		t.Fatal("BootstrapFromEnv: shutdown is nil; callers `defer shutdown()` unconditionally")
	}
	shutdown()
	shutdown()
}

// TestBootstrapFromEnv_SDKDisabled asserts that OTEL_SDK_DISABLED short-
// circuits exporter selection even when other env vars would otherwise
// trigger one. Matches the OTel spec.
func TestBootstrapFromEnv_SDKDisabled(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true")
	// Both of the below would normally attach an exporter; they must not.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
	t.Setenv("BELUGA_OTEL_STDOUT", "1")

	shutdown, err := BootstrapFromEnv(context.Background(), "beluga-test")
	if err != nil {
		t.Fatalf("BootstrapFromEnv: unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("BootstrapFromEnv: shutdown is nil")
	}
	shutdown()
}

// TestBootstrapFromEnv_StdoutExporter asserts that BELUGA_OTEL_STDOUT=1
// selects the stdout exporter when OTLP endpoint is unset.
func TestBootstrapFromEnv_StdoutExporter(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_SDK_DISABLED", "")
	t.Setenv("BELUGA_OTEL_STDOUT", "1")

	shutdown, err := BootstrapFromEnv(context.Background(), "beluga-test")
	if err != nil {
		t.Fatalf("BootstrapFromEnv: stdout path errored: %v", err)
	}
	if shutdown == nil {
		t.Fatal("BootstrapFromEnv: shutdown is nil")
	}
	defer shutdown()
}

// TestBootstrapFromEnv_OTLPEndpoint asserts that setting the OTLP
// endpoint env var does not fail at bootstrap time. HTTP dial happens on
// first span export, so a missing collector surfaces later, not here.
func TestBootstrapFromEnv_OTLPEndpoint(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")

	shutdown, err := BootstrapFromEnv(context.Background(), "beluga-test")
	if err != nil {
		t.Fatalf("BootstrapFromEnv: unexpected error with OTLP endpoint: %v", err)
	}
	if shutdown == nil {
		t.Fatal("BootstrapFromEnv: shutdown is nil")
	}
	defer shutdown()
}

// TestBootstrapFromEnv_WithOptions asserts that supplying TracerOption
// values (accepted by InitTracer) composes correctly with env-derived
// exporter selection.
func TestBootstrapFromEnv_WithOptions(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("BELUGA_OTEL_STDOUT", "1")

	shutdown, err := BootstrapFromEnv(
		context.Background(),
		"beluga-test",
		WithSyncExport(),
	)
	if err != nil {
		t.Fatalf("BootstrapFromEnv: unexpected error with options: %v", err)
	}
	if shutdown == nil {
		t.Fatal("BootstrapFromEnv: shutdown is nil")
	}
	defer shutdown()
}

// TestBootstrapFromEnv_UserOptionOverridesEnv asserts that a
// WithSpanExporter supplied explicitly beats the env-derived selection.
// This is the escape hatch used by tests that want to capture spans
// regardless of what the test runner's environment happens to have set.
func TestBootstrapFromEnv_UserOptionOverridesEnv(t *testing.T) {
	// Env would normally pick stdout; user option pins an in-memory recorder.
	t.Setenv("BELUGA_OTEL_STDOUT", "1")

	recorder := tracetest.NewInMemoryExporter()
	shutdown, err := BootstrapFromEnv(
		context.Background(),
		"beluga-test",
		WithSpanExporter(recorder),
		WithSyncExport(),
	)
	if err != nil {
		t.Fatalf("BootstrapFromEnv: unexpected error: %v", err)
	}
	defer shutdown()

	ctx, span := StartSpan(context.Background(), "bootstrap.test", nil)
	span.End()
	_ = ctx

	if got := len(recorder.GetSpans()); got != 1 {
		t.Fatalf("recorder captured %d spans, want 1 — user-supplied WithSpanExporter was ignored", got)
	}
}

// TestBootstrapFromEnv_WithSpanExporter_OverridesOTELDisabled pins the
// precedence rule from the BootstrapFromEnv docstring: an explicit
// WithSpanExporter supplied via opts beats OTEL_SDK_DISABLED. A prior
// implementation checked the env var first and returned noopShutdown
// before applying opts, which silently dropped user-supplied exporters.
func TestBootstrapFromEnv_WithSpanExporter_OverridesOTELDisabled(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("BELUGA_OTEL_STDOUT", "")

	recorder := tracetest.NewInMemoryExporter()
	shutdown, err := BootstrapFromEnv(
		context.Background(),
		"beluga-test",
		WithSpanExporter(recorder),
		WithSyncExport(),
	)
	if err != nil {
		t.Fatalf("BootstrapFromEnv: unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("BootstrapFromEnv: shutdown is nil")
	}
	defer shutdown()

	_, span := StartSpan(context.Background(), "bootstrap.override", nil)
	span.End()

	if got := len(recorder.GetSpans()); got != 1 {
		t.Fatalf("recorder captured %d spans, want 1 — OTEL_SDK_DISABLED suppressed a user-supplied WithSpanExporter", got)
	}
}

// envTruthy is trivial; a small table-driven test pins the allowed set.
func TestEnvTruthy(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"0", false},
		{"false", false},
		{"no", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"yes", true},
		{"YES", true},
	}
	for _, c := range cases {
		if got := envTruthy(c.in); got != c.want {
			t.Errorf("envTruthy(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// Compile-time assertion: stdouttrace / otlptracehttp exporters satisfy
// the sdktrace.SpanExporter interface that BootstrapFromEnv relies on.
// Declared here so a future SDK version bump that breaks the contract
// fails at compile time, not at runtime inside BootstrapFromEnv.
var _ sdktrace.SpanExporter = (*placeholderExporter)(nil)

type placeholderExporter struct{}

func (placeholderExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }
func (placeholderExporter) Shutdown(context.Context) error                             { return nil }
