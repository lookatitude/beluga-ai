// Package langfuse provides a Langfuse trace exporter for the Beluga AI
// observability system. It implements the [o11y.TraceExporter] interface and
// sends LLM call data to a Langfuse instance via its HTTP ingestion API.
//
// Langfuse is an open-source LLM engineering platform for tracing, evaluation,
// prompt management, and analytics.
//
// # Usage
//
// Create an exporter with your Langfuse credentials and use it to export
// LLM call data:
//
//	exporter, err := langfuse.New(
//	    langfuse.WithBaseURL("https://cloud.langfuse.com"),
//	    langfuse.WithPublicKey("pk-..."),
//	    langfuse.WithSecretKey("sk-..."),
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
//   - [WithBaseURL] — sets the Langfuse API base URL (default: https://cloud.langfuse.com)
//   - [WithPublicKey] — sets the Langfuse public key (required)
//   - [WithSecretKey] — sets the Langfuse secret key (required)
//   - [WithTimeout] — sets the HTTP client timeout (default: 10s)
package langfuse
