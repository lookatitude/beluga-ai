// Package langsmith provides a LangSmith trace exporter for the Beluga AI
// observability system. It implements the [o11y.TraceExporter] interface and
// sends LLM call data to LangSmith via its HTTP runs API.
//
// LangSmith is LangChain's platform for debugging, testing, evaluating,
// and monitoring LLM applications.
//
// # Usage
//
// Create an exporter with your LangSmith API key and use it to export
// LLM call data:
//
//	exporter, err := langsmith.New(
//	    langsmith.WithAPIKey("lsv2_..."),
//	    langsmith.WithProject("my-project"),
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
//   - [WithBaseURL] — sets the LangSmith API base URL (default: https://api.smith.langchain.com)
//   - [WithAPIKey] — sets the LangSmith API key (required)
//   - [WithProject] — sets the LangSmith project name (default: "default")
//   - [WithTimeout] — sets the HTTP client timeout (default: 10s)
package langsmith
