// Package opik provides an Opik trace exporter for the Beluga AI observability
// system. It implements the [o11y.TraceExporter] interface and sends LLM call
// data to Opik via its HTTP tracing API.
//
// Opik (by Comet) provides LLM experiment tracking, tracing, and evaluation.
//
// # Usage
//
// Create an exporter with your Opik credentials and use it to export
// LLM call data:
//
//	exporter, err := opik.New(
//	    opik.WithAPIKey("opik-..."),
//	    opik.WithWorkspace("my-workspace"),
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
//   - [WithBaseURL] — sets the Opik API base URL (default: https://www.comet.com/opik/api)
//   - [WithAPIKey] — sets the Opik API key (required)
//   - [WithWorkspace] — sets the Opik workspace name (default: "default")
//   - [WithTimeout] — sets the HTTP client timeout (default: 10s)
package opik
