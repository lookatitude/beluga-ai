// Package phoenix provides an Arize Phoenix trace exporter for the Beluga AI
// observability system. It implements the [o11y.TraceExporter] interface and
// sends LLM call data to an Arize Phoenix instance via its HTTP API.
//
// Phoenix uses OTel-compatible spans, so this exporter translates LLM call
// data into the Phoenix /v1/traces JSON format.
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
