package o11y

import (
	"context"
	"os"
)

// BootstrapFromEnv configures the global OTel tracer provider from the
// standard OTEL_* environment variables and returns a shutdown function.
//
// S1 ships this as a skeleton: no S1 subcommand invokes it. S3+ subcommands
// (beluga run, beluga dev, beluga eval) will call it from their RunE. It
// lives in o11y/ (not cmd/) so application developers writing their own
// main.go can reuse the same one-call bootstrap.
//
// Behaviour (target contract, partially realised in S3+):
//   - If OTEL_EXPORTER_OTLP_ENDPOINT is set, the SDK's auto-configuration
//     dials an OTLP exporter.
//   - If BELUGA_OTEL_STDOUT=1 is set and no OTLP endpoint, fall back to a
//     stdout JSON exporter (useful for local debugging).
//   - Otherwise, no exporter is attached; spans become no-ops silently.
//
// The returned shutdown function is always safe to call (nil-safe and
// idempotent). A non-nil error means SDK initialisation failed — callers
// should log and continue rather than hard-fail.
//
// Additional TracerOption values override the env-derived configuration
// and compose with the existing WithSpanExporter / WithSampler /
// WithSyncExport options in tracer.go.
func BootstrapFromEnv(ctx context.Context, serviceName string, opts ...TracerOption) (shutdown func(), err error) {
	_ = ctx
	_ = serviceName

	// S1 skeleton: read env to validate the contract compiles; do not attach
	// an exporter. S3+ will wire the real OTLP exporter behind these env vars.
	_ = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	_ = os.Getenv("BELUGA_OTEL_STDOUT")

	_ = opts // reserved; composes with existing tracerConfig in S3+.

	// Nil-safe, idempotent no-op. Callers always `defer shutdown()`.
	return func() { /* S1 skeleton: no exporter to shut down */ }, nil
}
