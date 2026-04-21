// Package playground implements the minimal HTTP dev-UI that `beluga dev
// --playground <port>` mounts on 127.0.0.1:<port>. It is a Layer 7
// subpackage of the beluga CLI — stdlib + OpenTelemetry only, no
// imports from other beluga packages.
//
// The server exposes three panels backed by a single Server-Sent-Events
// stream at /events:
//
//  1. Last 50 OTel spans (name, duration, status, gen_ai.request.model,
//     input/output tokens).
//  2. 100-line live tail of the scaffolded project's stderr.
//  3. Per-run token count + cost estimate, derived from span attributes.
//
// Data enters the server via two channels the supervisor writes to:
// [Server.SpanSink] for OTel span exports (via a small [SpanExporter]
// adapter in span_export.go) and [Server.StderrSink] for tee'd stderr
// lines. The server fans these out to every connected SSE subscriber
// and retains the bounded history needed to populate a late-arriving
// browser tab.
//
// Security defaults are strict: the listener binds 127.0.0.1 only
// (never 0.0.0.0), CORS is set to the exact playground origin (never
// '*'), POST handlers check Sec-Fetch-Site=same-origin, and HTTP
// timeouts satisfy gosec G112.
package playground
