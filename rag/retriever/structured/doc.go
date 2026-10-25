// Package structured provides a Text2Cypher/Text2SQL retriever that converts
// natural-language questions into structured database queries (Cypher or SQL),
// executes them, and returns the results as [schema.Document] values.
//
// The retriever follows a generate-execute-evaluate pipeline:
//
//  1. A [QueryGenerator] translates the question into a database query using
//     the provided schema information.
//  2. A [QueryExecutor] runs the query against the target database. The
//     [ReadOnlyExecutor] wrapper rejects queries containing write, procedure
//     or file/network I/O keywords. It is a best-effort denylist, not a
//     security boundary: run LLM-generated queries under a read-only
//     database role or session as the real control.
//  3. A [ResultEvaluator] scores the results for relevance. If the score is
//     below the threshold the pipeline retries with feedback, up to a
//     configurable maximum.
//
// The package registers itself as "structured" in the retriever registry.
// Import it for side effects:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/retriever/structured"
package structured
