package webhooks

import (
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/k8s/operator"
)

// validAgent returns a fully-populated AgentResource that passes all validation.
func validAgent() *operator.AgentResource {
	return &operator.AgentResource{
		APIVersion: "beluga.ai/v1",
		Kind:       "Agent",
		Meta:       operator.ObjectMeta{Name: "test-agent", Namespace: "default"},
		Spec: operator.AgentSpec{
			ModelRef:      "gpt-4o-ref",
			Planner:       "react",
			MaxIterations: 5,
			Persona:       operator.Persona{Role: "executor"},
		},
	}
}

// validTeam returns a fully-populated TeamResource that passes all validation.
func validTeam() *operator.TeamResource {
	return &operator.TeamResource{
		APIVersion: "beluga.ai/v1",
		Kind:       "Team",
		Meta:       operator.ObjectMeta{Name: "test-team", Namespace: "default"},
		Spec: operator.TeamSpec{
			Pattern: "sequential",
			Members: []operator.TeamMemberRef{
				{Name: "agent-a", Role: "planner"},
				{Name: "agent-b", Role: "executor"},
			},
		},
	}
}

// ----------------------------------------------------------------------------
// ValidateAgent
// ----------------------------------------------------------------------------

func TestValidateAgent(t *testing.T) {
	tests := []struct {
		name        string
		agent       *operator.AgentResource
		wantAllowed bool
		wantReason  string // substring expected in Reason when !wantAllowed
	}{
		{
			name:        "valid agent passes",
			agent:       validAgent(),
			wantAllowed: true,
		},
		{
			name:        "nil agent denied",
			agent:       nil,
			wantAllowed: false,
			wantReason:  "nil",
		},
		{
			name: "missing modelRef denied",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.ModelRef = ""
				return a
			}(),
			wantAllowed: false,
			wantReason:  "modelRef",
		},
		{
			name: "whitespace-only modelRef denied",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.ModelRef = "   "
				return a
			}(),
			wantAllowed: false,
			wantReason:  "modelRef",
		},
		{
			name: "invalid planner denied",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.Planner = "unknown-planner"
				return a
			}(),
			wantAllowed: false,
			wantReason:  "planner",
		},
		{
			name: "react planner accepted",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.Planner = "react"
				return a
			}(),
			wantAllowed: true,
		},
		{
			name: "openai-functions planner accepted",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.Planner = "openai-functions"
				return a
			}(),
			wantAllowed: true,
		},
		{
			name: "plan-and-execute planner accepted",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.Planner = "plan-and-execute"
				return a
			}(),
			wantAllowed: true,
		},
		{
			name: "maxIterations zero denied",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.MaxIterations = 0
				return a
			}(),
			wantAllowed: false,
			wantReason:  "maxIterations",
		},
		{
			name: "maxIterations negative denied",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.MaxIterations = -1
				return a
			}(),
			wantAllowed: false,
			wantReason:  "maxIterations",
		},
		{
			name: "empty persona role denied",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.Persona.Role = ""
				return a
			}(),
			wantAllowed: false,
			wantReason:  "persona.role",
		},
		{
			name: "whitespace persona role denied",
			agent: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.Persona.Role = "  "
				return a
			}(),
			wantAllowed: false,
			wantReason:  "persona.role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertAgentValidation(t, ValidateAgent(tt.agent), tt.wantAllowed, tt.wantReason)
		})
	}
}

// assertAgentValidation checks the Allowed flag and Reason of an agent
// validation result against the expected outcome.
func assertAgentValidation(t *testing.T, got ValidationResult, wantAllowed bool, wantReason string) {
	t.Helper()
	if got.Allowed != wantAllowed {
		t.Errorf("Allowed = %v, want %v (Reason: %q)", got.Allowed, wantAllowed, got.Reason)
	}
	if !wantAllowed && wantReason != "" && !containsSubstring(got.Reason, wantReason) {
		t.Errorf("Reason = %q, want it to contain %q", got.Reason, wantReason)
	}
	if wantAllowed && got.Reason != "" {
		t.Errorf("Reason should be empty when Allowed, got %q", got.Reason)
	}
}

// ----------------------------------------------------------------------------
// ValidateTeam
// ----------------------------------------------------------------------------

func TestValidateTeam(t *testing.T) {
	tests := []struct {
		name        string
		team        *operator.TeamResource
		wantAllowed bool
		wantReason  string
	}{
		{
			name:        "valid team passes",
			team:        validTeam(),
			wantAllowed: true,
		},
		{
			name:        "nil team denied",
			team:        nil,
			wantAllowed: false,
			wantReason:  "nil",
		},
		{
			name: "invalid pattern denied",
			team: func() *operator.TeamResource {
				t := validTeam()
				t.Spec.Pattern = "bad-pattern"
				return t
			}(),
			wantAllowed: false,
			wantReason:  "pattern",
		},
		{
			name: "sequential pattern accepted",
			team: func() *operator.TeamResource {
				t := validTeam()
				t.Spec.Pattern = "sequential"
				return t
			}(),
			wantAllowed: true,
		},
		{
			name: "parallel pattern accepted",
			team: func() *operator.TeamResource {
				t := validTeam()
				t.Spec.Pattern = "parallel"
				return t
			}(),
			wantAllowed: true,
		},
		{
			name: "supervisor pattern accepted",
			team: func() *operator.TeamResource {
				t := validTeam()
				t.Spec.Pattern = "supervisor"
				return t
			}(),
			wantAllowed: true,
		},
		{
			name: "empty members denied",
			team: func() *operator.TeamResource {
				t := validTeam()
				t.Spec.Members = nil
				return t
			}(),
			wantAllowed: false,
			wantReason:  "members",
		},
		{
			name: "zero-length members slice denied",
			team: func() *operator.TeamResource {
				t := validTeam()
				t.Spec.Members = []operator.TeamMemberRef{}
				return t
			}(),
			wantAllowed: false,
			wantReason:  "members",
		},
		{
			name: "duplicate member names denied",
			team: func() *operator.TeamResource {
				t := validTeam()
				t.Spec.Members = []operator.TeamMemberRef{
					{Name: "agent-a"},
					{Name: "agent-a"},
				}
				return t
			}(),
			wantAllowed: false,
			wantReason:  "duplicate",
		},
		{
			name: "single member passes",
			team: func() *operator.TeamResource {
				t := validTeam()
				t.Spec.Members = []operator.TeamMemberRef{{Name: "solo"}}
				return t
			}(),
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateTeam(tt.team)
			if got.Allowed != tt.wantAllowed {
				t.Errorf("Allowed = %v, want %v (Reason: %q)", got.Allowed, tt.wantAllowed, got.Reason)
			}
			if !tt.wantAllowed && tt.wantReason != "" {
				if !containsSubstring(got.Reason, tt.wantReason) {
					t.Errorf("Reason = %q, want it to contain %q", got.Reason, tt.wantReason)
				}
			}
		})
	}
}

// ----------------------------------------------------------------------------
// MutateAgent
// ----------------------------------------------------------------------------

func TestMutateAgent(t *testing.T) {
	tests := []struct {
		name              string
		input             *operator.AgentResource
		wantNil           bool
		wantMaxIterations int
		wantLabel         string // expected value of beluga.ai/component label
	}{
		{
			name:    "nil input returns nil",
			input:   nil,
			wantNil: true,
		},
		{
			name: "zero maxIterations gets default 10",
			input: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.MaxIterations = 0
				return a
			}(),
			wantMaxIterations: 10,
			wantLabel:         "agent",
		},
		{
			name: "negative maxIterations gets default 10",
			input: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.MaxIterations = -5
				return a
			}(),
			wantMaxIterations: 10,
			wantLabel:         "agent",
		},
		{
			name: "positive maxIterations preserved",
			input: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.MaxIterations = 7
				return a
			}(),
			wantMaxIterations: 7,
			wantLabel:         "agent",
		},
		{
			name: "component label applied without existing labels",
			input: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.MaxIterations = 3
				a.Meta.Labels = nil
				return a
			}(),
			wantMaxIterations: 3,
			wantLabel:         "agent",
		},
		{
			name: "existing labels preserved",
			input: func() *operator.AgentResource {
				a := validAgent()
				a.Spec.MaxIterations = 3
				a.Meta.Labels = map[string]string{"team": "alpha"}
				return a
			}(),
			wantMaxIterations: 3,
			wantLabel:         "agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MutateAgent(tt.input)
			if tt.wantNil {
				assertNilMutation(t, got == nil)
				return
			}
			assertMutatedAgent(t, tt.input, got, tt.wantMaxIterations, tt.wantLabel)
		})
	}
}

// assertNilMutation fails the test when a mutation that should return nil
// returned a non-nil resource.
func assertNilMutation(t *testing.T, gotIsNil bool) {
	t.Helper()
	if !gotIsNil {
		t.Errorf("want nil, got non-nil")
	}
}

// assertMutatedAgent checks the defaults and labels applied by MutateAgent
// and that the result is a copy of input.
func assertMutatedAgent(t *testing.T, input, got *operator.AgentResource, wantMaxIterations int, wantLabel string) {
	t.Helper()
	if got == nil {
		t.Fatal("want non-nil, got nil")
	}
	if got.Spec.MaxIterations != wantMaxIterations {
		t.Errorf("MaxIterations = %d, want %d", got.Spec.MaxIterations, wantMaxIterations)
	}
	if got.Meta.Labels[labelComponent] != wantLabel {
		t.Errorf("label %q = %q, want %q", labelComponent, got.Meta.Labels[labelComponent], wantLabel)
	}
	// Original must not be mutated.
	if input != nil && input == got {
		t.Error("MutateAgent must return a copy, not the same pointer")
	}
}

// TestMutateAgentDoesNotMutateOriginal verifies the copy-on-write contract.
func TestMutateAgentDoesNotMutateOriginal(t *testing.T) {
	original := validAgent()
	original.Spec.MaxIterations = 0
	original.Meta.Labels = nil

	_ = MutateAgent(original)

	if original.Spec.MaxIterations != 0 {
		t.Error("MutateAgent must not modify original MaxIterations")
	}
	if original.Meta.Labels != nil {
		t.Error("MutateAgent must not modify original Labels")
	}
}

// ----------------------------------------------------------------------------
// MutateTeam
// ----------------------------------------------------------------------------

func TestMutateTeam(t *testing.T) {
	tests := []struct {
		name      string
		input     *operator.TeamResource
		wantNil   bool
		wantLabel string
	}{
		{
			name:    "nil input returns nil",
			input:   nil,
			wantNil: true,
		},
		{
			name:      "standard label applied",
			input:     validTeam(),
			wantLabel: "team",
		},
		{
			name: "existing labels preserved alongside component label",
			input: func() *operator.TeamResource {
				team := validTeam()
				team.Meta.Labels = map[string]string{"env": "prod"}
				return team
			}(),
			wantLabel: "team",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MutateTeam(tt.input)
			if tt.wantNil {
				assertNilMutation(t, got == nil)
				return
			}
			assertMutatedTeam(t, tt.input, got, tt.wantLabel)
		})
	}
}

// assertMutatedTeam checks the component label applied by MutateTeam, that
// pre-existing labels are preserved, and that the result is a copy of input.
func assertMutatedTeam(t *testing.T, input, got *operator.TeamResource, wantLabel string) {
	t.Helper()
	if got == nil {
		t.Fatal("want non-nil, got nil")
	}
	if got.Meta.Labels[labelComponent] != wantLabel {
		t.Errorf("label %q = %q, want %q", labelComponent, got.Meta.Labels[labelComponent], wantLabel)
	}
	if input == nil {
		return
	}
	// Verify preserved labels.
	for k, v := range input.Meta.Labels {
		if got.Meta.Labels[k] != v {
			t.Errorf("existing label %q: got %q, want %q", k, got.Meta.Labels[k], v)
		}
	}
	// Original must not be mutated.
	if input == got {
		t.Error("MutateTeam must return a copy, not the same pointer")
	}
}

// TestMutateTeamDoesNotMutateOriginal verifies the copy-on-write contract.
func TestMutateTeamDoesNotMutateOriginal(t *testing.T) {
	original := validTeam()
	original.Meta.Labels = nil

	_ = MutateTeam(original)

	if original.Meta.Labels != nil {
		t.Error("MutateTeam must not modify original Labels")
	}
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// containsSubstring reports whether s contains the substring sub
// using a case-sensitive search, without importing strings in test code.
func containsSubstring(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Regression (FI-45): agents may not set the operator's reserved selector
// labels.
func TestValidateAgent_RejectsReservedLabels(t *testing.T) {
	for _, key := range []string{"app", "beluga-agent", "managed-by"} {
		t.Run(key, func(t *testing.T) {
			a := validAgent()
			a.Meta.Labels = map[string]string{key: "x", "team": "blue"}
			res := ValidateAgent(a)
			if res.Allowed {
				t.Fatalf("expected denial for reserved label %q", key)
			}
			if !strings.Contains(res.Reason, key) {
				t.Errorf("reason %q does not mention %q", res.Reason, key)
			}
		})
	}
	a := validAgent()
	a.Meta.Labels = map[string]string{"team": "blue", "beluga.ai/component": "agent"}
	if res := ValidateAgent(a); !res.Allowed {
		t.Errorf("non-reserved labels denied: %s", res.Reason)
	}
}

// The mutating webhook's own label must never trip the validator.
func TestValidateAgent_AcceptsMutatedAgent(t *testing.T) {
	if res := ValidateAgent(MutateAgent(validAgent())); !res.Allowed {
		t.Errorf("mutated agent denied: %s", res.Reason)
	}
}
