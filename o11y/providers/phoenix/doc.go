// Package phoenix provides an Arize Phoenix trace exporter for the Beluga AI
// observability system. It implements the [o11y.TraceExporter] interface and
// sends LLM call data to an Arize Phoenix instance via its HTTP API.
//
// This exporter translates LLM call data into a JSON span document and POSTs
// it to {BaseURL}/v1/traces.
//
// # Compatibility warning
//
// A stock Phoenix server's /v1/traces endpoint is an OTLP/HTTP collector: it
// accepts OTLP (protobuf) ExportTraceServiceRequest bodies, not the JSON shape
// this exporter sends, so exports to an unmodified Phoenix instance are
// expected to be rejected. To send traces to Phoenix today, use the
// framework's OpenTelemetry pipeline instead of this package:
//
//	// OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:6006
//	shutdown, err := o11y.BootstrapFromEnv(ctx, "my-agent")
//
// o11y.BootstrapFromEnv installs the OTLP/HTTP exporter, which appends
// /v1/traces and speaks the protocol Phoenix expects. Keep this package only
// for a proxy or shim that accepts the JSON format documented on
// [Exporter.ExportLLMCall].
//
// # Usage
//
// Create an exporter pointing to your Phoenix instance:
//
//	exporter, err := phoenix.New(
//	    phoenix.WithBaseURL("http://localhost:6006"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = exporter.ExportLLMCall(ctx, data)
//
// The exporter can be used standalone or composed with other exporters
// via [o11y.MultiExporter].
//
// # Configuration Options
//
//   - [WithBaseURL] — sets the Phoenix API base URL (default: http://localhost:6006)
//   - [WithAPIKey] — sets the Phoenix API key for authentication (optional)
//   - [WithTimeout] — sets the HTTP client timeout (default: 10s)
package phoenix
