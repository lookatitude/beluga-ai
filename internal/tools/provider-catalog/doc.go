// Command provider-catalog walks the Beluga AI provider directories and emits
// a Markdown catalog to stdout.
//
// The output is meant to be redirected into docs/reference/providers.md and
// checked for drift via make docs-providers. It is a Layer 7 (Application)
// internal build tool.
package main
