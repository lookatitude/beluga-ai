// Package operator provides Kubernetes operator types and reconciliation logic
// for the Beluga AI Agent and Team custom resources (CRDs).
//
// This package is intentionally free of any Kubernetes library dependencies.
// It defines Go structs that mirror the CRD schemas and a [Reconciler]
// interface whose implementations generate plain [ReconcileResult] values
// (deployment spec, service spec, optional HPA spec) that callers can map to
// actual Kubernetes API objects using whichever K8s client library they prefer.
//
// # CRD Types
//
// [AgentResource] and [TeamResource] represent the top-level custom resources.
// Their embedded Spec structs match the YAML schemas used in deploy/k8s/*.yaml.
//
// # Reconciler
//
// [Reconciler] is the primary extension point. It exposes two methods:
//
//   - [Reconciler.ReconcileAgent] – computes the desired Kubernetes resources
//     for a single Agent CRD.
//   - [Reconciler.ReconcileTeam] – computes the desired Kubernetes resources
//     for a Team CRD (one resource set per member agent).
//
// [NewDefaultReconciler] returns the built-in implementation that derives
// resource names, replica counts, resource limits, and autoscaling parameters
// directly from the CRD spec.
//
// # Usage
//
//	r := operator.NewDefaultReconciler()
//	result, err := r.ReconcileAgent(ctx, agentResource)
//	if err != nil { ... }
//	// result.Deployment, result.Service, result.HPA are ready to apply.
//
// # No External Dependencies
//
// This package imports only the Go standard library and
// github.com/lookatitude/beluga-ai/v2/core. It must never be imported by core
// library packages.
package operator
