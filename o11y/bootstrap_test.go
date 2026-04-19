package o11y

import (
	"context"
	"testing"
)

// TestBootstrapFromEnv_CleanEnv asserts that with no OTEL_* variables set,
// BootstrapFromEnv returns a non-nil shutdown function and a nil error.
// The shutdown must be nil-safe and safe to call multiple times (idempotent).
func TestBootstrapFromEnv_CleanEnv(t *testing.T) {
	// S1 skeleton does not dial exporters; callers rely on the nil-safe
	// contract regardless of environment.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("BELUGA_OTEL_STDOUT", "")

	shutdown, err := BootstrapFromEnv(context.Background(), "beluga-test")
	if err != nil {
		t.Fatalf("BootstrapFromEnv: unexpected error with clean env: %v", err)
	}
	if shutdown == nil {
		t.Fatal("BootstrapFromEnv: shutdown is nil; callers `defer shutdown()` unconditionally")
	}
	// Double-call must not panic.
	shutdown()
	shutdown()
}

// TestBootstrapFromEnv_OTLPEndpoint asserts that setting the OTLP endpoint
// env var does not cause failure at bootstrap time. The SDK's
// dial-on-send semantics mean a missing collector surfaces later, not here.
func TestBootstrapFromEnv_OTLPEndpoint(t *testing.T) {
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
// values (the same ones accepted by InitTracer) does not panic or error
// at bootstrap time. S1 does not apply them to a provider, but the
// contract reserves the slot for S3+ to compose with env-derived config.
func TestBootstrapFromEnv_WithOptions(t *testing.T) {
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
