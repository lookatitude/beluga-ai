package o11y

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

// BootstrapFromEnv configures the global OTel tracer provider from the
// standard OTEL_* environment variables and returns a shutdown function.
//
// Resolution order (first match wins):
//
//  1. Explicit WithSpanExporter(...) in opts overrides everything below.
//  2. OTEL_SDK_DISABLED truthy → no exporter attached; spans are no-ops.
//     Truthy values: "1", "true" (case-insensitive), "yes".
//  3. OTEL_EXPORTER_OTLP_ENDPOINT set → OTLP/HTTP exporter. The SDK reads
//     OTEL_EXPORTER_OTLP_HEADERS, OTEL_EXPORTER_OTLP_TIMEOUT, etc. directly.
//  4. BELUGA_OTEL_STDOUT truthy → pretty-printed stdout JSON exporter.
//     Intended for local `beluga dev` debugging — never for production.
//  5. Default: no exporter; spans are silent no-ops.
//
// The shutdown function is always non-nil and safe to call. A non-nil
// error means SDK initialisation failed (e.g. malformed endpoint) —
// callers should log and continue rather than hard-fail. Additional
// TracerOption values (sampler, sync-export) compose with env-derived
// exporter selection.
func BootstrapFromEnv(ctx context.Context, serviceName string, opts ...TracerOption) (func(), error) {
	cfg := &tracerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// User-supplied WithSpanExporter wins unconditionally — the docstring
	// contract promises this precedence so tests can pin an in-memory
	// recorder even when the ambient env would otherwise disable the SDK.
	if cfg.exporter == nil {
		if envTruthy(os.Getenv("OTEL_SDK_DISABLED")) {
			return noopShutdown, nil
		}
		switch {
		case os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "":
			exp, err := otlptracehttp.New(ctx)
			if err != nil {
				return noopShutdown, fmt.Errorf("o11y: OTLP HTTP exporter: %w", err)
			}
			cfg.exporter = exp
		case envTruthy(os.Getenv("BELUGA_OTEL_STDOUT")):
			exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
			if err != nil {
				return noopShutdown, fmt.Errorf("o11y: stdout exporter: %w", err)
			}
			cfg.exporter = exp
		}
	}

	if cfg.exporter == nil {
		return noopShutdown, nil
	}

	finalOpts := []TracerOption{WithSpanExporter(cfg.exporter)}
	if cfg.syncExport {
		finalOpts = append(finalOpts, WithSyncExport())
	}
	if cfg.sampler != nil {
		finalOpts = append(finalOpts, WithSampler(cfg.sampler))
	}
	return InitTracer(serviceName, finalOpts...)
}

// noopShutdown is the sentinel returned whenever bootstrap elects not to
// attach an exporter. Callers always `defer shutdown()`.
func noopShutdown() {
	// Intentionally empty: with no exporter attached there is nothing to
	// flush, drain, or release on shutdown — this sentinel exists only
	// so callers can unconditionally `defer shutdown()` without a nil
	// check.
}

// envTruthy reports whether an environment variable's value should be
// treated as true. Matches the conservative set the OTel spec uses for
// OTEL_SDK_DISABLED.
func envTruthy(v string) bool {
	switch v {
	case "1", "true", "TRUE", "True", "yes", "YES":
		return true
	}
	return false
}
