package webhooks

import (
	"github.com/lookatitude/beluga-ai/v2/k8s/operator"
)

const (
	// labelComponent is the standard Beluga component label key.
	labelComponent = "beluga.ai/component"

	// componentAgent is the label value applied to Agent resources.
	componentAgent = "agent"

	// componentTeam is the label value applied to Team resources.
	componentTeam = "team"

	// defaultMaxIterations is the default value for MaxIterations when not set.
	defaultMaxIterations = 10
)

// MutateAgent applies defaults and standard labels to an AgentResource.
// It returns a shallow copy of the input with the following mutations applied:
//   - Spec.MaxIterations is set to 10 when it is zero or negative.
//   - Meta.Labels["beluga.ai/component"] is set to "agent".
//
// The original resource is not modified.
func MutateAgent(agent *operator.AgentResource) *operator.AgentResource {
	if agent == nil {
		return nil
	}

	// Shallow-copy to avoid mutating the caller's struct.
	out := *agent

	if out.Spec.MaxIterations <= 0 {
		out.Spec.MaxIterations = defaultMaxIterations
	}

	out.Meta.Labels = ensureLabel(out.Meta.Labels, labelComponent, componentAgent)

	return &out
}

// MutateTeam applies defaults and standard labels to a TeamResource.
// It returns a shallow copy of the input with the following mutations applied:
//   - Meta.Labels["beluga.ai/component"] is set to "team".
//
// The original resource is not modified.
func MutateTeam(team *operator.TeamResource) *operator.TeamResource {
	if team == nil {
		return nil
	}

	// Shallow-copy to avoid mutating the caller's struct.
	out := *team

	out.Meta.Labels = ensureLabel(out.Meta.Labels, labelComponent, componentTeam)

	return &out
}

// ensureLabel returns a copy of labels with key set to value.
// If labels is nil a new map is allocated.
func ensureLabel(labels map[string]string, key, value string) map[string]string {
	result := make(map[string]string, len(labels)+1)
	for k, v := range labels {
		result[k] = v
	}
	result[key] = value
	return result
}
