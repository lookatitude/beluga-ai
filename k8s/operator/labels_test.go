package operator

import (
	"context"
	"testing"
)

// Regression (FI-45): user-supplied labels must not override the operator's
// reserved selector labels, otherwise a Service selector could match another
// workload's pods.
func TestReconcileAgent_ReservedLabelsCannotBeOverridden(t *testing.T) {
	a := agentWithModelRef("my-agent", "gpt")
	a.Meta.Labels = map[string]string{
		"app":          "other-workload",
		"beluga-agent": "someone-else",
		"managed-by":   "attacker",
		"team":         "blue",
	}
	res, err := NewDefaultReconciler().ReconcileAgent(context.Background(), a)
	if err != nil {
		t.Fatalf("ReconcileAgent: %v", err)
	}
	want := map[string]string{
		"app":          "my-agent",
		"beluga-agent": "my-agent",
		"managed-by":   "beluga-operator",
		"team":         "blue",
	}
	for _, labels := range []map[string]string{res.Deployment.Labels, res.Service.Selector} {
		for k, v := range want {
			if labels[k] != v {
				t.Errorf("label %q = %q, want %q", k, labels[k], v)
			}
		}
	}
}

func TestAgentLabels_DoesNotMutateInput(t *testing.T) {
	in := map[string]string{"app": "x"}
	_ = agentLabels("n", in)
	if in["app"] != "x" || len(in) != 1 {
		t.Errorf("input labels mutated: %v", in)
	}
}

func TestIsReservedLabel(t *testing.T) {
	tests := map[string]bool{
		"app":          true,
		"beluga-agent": true,
		"managed-by":   true,
		"team":         false,
		"beluga-team":  false,
		"App":          false,
		"":             false,
	}
	for k, want := range tests {
		if got := IsReservedLabel(k); got != want {
			t.Errorf("IsReservedLabel(%q) = %v, want %v", k, got, want)
		}
	}
	keys := ReservedLabelKeys()
	if len(keys) != 3 {
		t.Fatalf("ReservedLabelKeys() = %v, want 3 keys", keys)
	}
	// Returned slice is a copy.
	keys[0] = "mutated"
	if ReservedLabelKeys()[0] == "mutated" {
		t.Error("ReservedLabelKeys returned shared slice")
	}
}
