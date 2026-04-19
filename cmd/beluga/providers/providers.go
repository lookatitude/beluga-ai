// Package providers is a side-effect-only package that triggers init()
// registration for the curated set of providers shipped with the beluga CLI.
//
// The providers listed here MUST be CGo-free. CGO_ENABLED=0 is set in the
// goreleaser build; any CGo dependency will silently break cross-compilation
// on the CI runner. Each addition to this list requires an explicit audit —
// check the provider's imports (and its transitive SDK imports) for `import
// "C"` before adding it here. See docs/consultations/2026-04-19-loo-142-architect-plan.md
// (risks for reviewer-security).
//
// NOTE: blank imports are added in T5 of the DX-1 S1 plan. T2 creates this
// placeholder so cmd/beluga/main.go's side-effect import compiles without a
// commented-out line.
package providers
