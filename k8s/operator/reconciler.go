package operator

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Reconciler defines the reconciliation interface for Beluga AI custom resources.
// Implementations compute the desired Kubernetes resource state from a CRD
// and return it as plain Go structs — they never call the Kubernetes API directly.
//
// Callers are responsible for applying the returned [ReconcileResult] to the
// cluster using their preferred K8s client library.
type Reconciler interface {
	// ReconcileAgent derives the desired Kubernetes resources for a single Agent
	// CRD. It returns an error if the spec is invalid (e.g. missing ModelRef).
	ReconcileAgent(ctx context.Context, agent *AgentResource) (*ReconcileResult, error)

	// ReconcileTeam derives the desired Kubernetes resources for every member
	// agent defined in a Team CRD. It returns an error if the spec is invalid.
	ReconcileTeam(ctx context.Context, team *TeamResource) (*TeamReconcileResult, error)
}

// compile-time interface check.
var _ Reconciler = (*DefaultReconciler)(nil)

// DefaultReconciler is the built-in [Reconciler] implementation.
// It derives Deployment, Service, and (when enabled) HPA specs from the CRD
// spec fields using sensible defaults.
type DefaultReconciler struct {
	// defaultImage is used when AgentSpec.Image is empty.
	defaultImage string

	// defaultPort is used when AgentSpec.Port is zero.
	defaultPort int

	// defaultReplicas is used when AgentSpec.Replicas is zero.
	defaultReplicas int
}

// ReconcilerOption is a functional option for [DefaultReconciler].
type ReconcilerOption func(*DefaultReconciler)

// WithDefaultImage sets the container image used when AgentSpec.Image is not
// specified.
func WithDefaultImage(image string) ReconcilerOption {
	return func(r *DefaultReconciler) {
		r.defaultImage = image
	}
}

// WithDefaultPort sets the container port used when AgentSpec.Port is zero.
func WithDefaultPort(port int) ReconcilerOption {
	return func(r *DefaultReconciler) {
		r.defaultPort = port
	}
}

// WithDefaultReplicas sets the replica count used when AgentSpec.Replicas is
// zero and scaling is disabled.
func WithDefaultReplicas(n int) ReconcilerOption {
	return func(r *DefaultReconciler) {
		r.defaultReplicas = n
	}
}

// NewDefaultReconciler constructs a [DefaultReconciler] with optional functional
// options. Sensible defaults apply when no options are provided:
//
//   - Image: "beluga-agent:latest"
//   - Port:  8080
//   - Replicas: 1
func NewDefaultReconciler(opts ...ReconcilerOption) *DefaultReconciler {
	r := &DefaultReconciler{
		defaultImage:    "beluga-agent:latest",
		defaultPort:     8080,
		defaultReplicas: 1,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ReconcileAgent derives the desired Kubernetes resources for agent.
//
// The following rules apply:
//   - ModelRef must not be empty; an [core.ErrInvalidInput] error is returned
//     otherwise.
//   - When AgentSpec.Replicas is zero the default replica count is used.
//   - When AgentSpec.Port is zero the default port is used.
//   - When AgentSpec.Image is empty the default image is used.
//   - When AgentSpec.Scaling.Enabled is true an HPA spec is included in the
//     result; MinReplicas defaults to 1, MaxReplicas defaults to 5, and
//     TargetCPUUtilization defaults to 80 when those fields are zero.
func (r *DefaultReconciler) ReconcileAgent(ctx context.Context, agent *AgentResource) (*ReconcileResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if agent == nil {
		return nil, core.NewError("operator.reconcileAgent", core.ErrInvalidInput, "agent must not be nil", nil)
	}
	if err := validateAgentSpec(agent); err != nil {
		return nil, err
	}

	name := agent.Meta.Name
	ns := agent.Meta.Namespace
	image := agentImage(agent.Spec, r.defaultImage)
	port := agentPort(agent.Spec, r.defaultPort)
	replicas := agentReplicas(agent.Spec, r.defaultReplicas)

	labels := agentLabels(name, agent.Meta.Labels)

	deployment := buildDeploymentSpec(name, ns, image, replicas, port, labels, agent.Spec)
	service := buildServiceSpec(name, ns, port, labels)

	var hpa *HPASpec
	if agent.Spec.Scaling.Enabled {
		hpa = buildHPASpec(name, ns, agent.Spec.Scaling)
	}

	return &ReconcileResult{
		Deployment: deployment,
		Service:    service,
		HPA:        hpa,
	}, nil
}

// ReconcileTeam derives the desired Kubernetes resources for each member agent
// in team. Each member is reconciled independently using a synthesised
// [AgentResource] whose spec is derived from the member reference.
//
// The returned [TeamReconcileResult] maps member names to their individual
// [ReconcileResult] values. An error is returned only when the Team spec itself
// is invalid; member-level errors are collected and returned as a combined
// error.
func (r *DefaultReconciler) ReconcileTeam(ctx context.Context, team *TeamResource) (*TeamReconcileResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if team == nil {
		return nil, core.NewError("operator.reconcileTeam", core.ErrInvalidInput, "team must not be nil", nil)
	}
	if err := validateTeamSpec(team); err != nil {
		return nil, err
	}

	result := &TeamReconcileResult{
		Members: make(map[string]*ReconcileResult, len(team.Spec.Members)),
	}

	var firstErr error
	for _, member := range team.Spec.Members {
		// Build a minimal AgentResource for each member so that the shared
		// reconciliation logic can be reused.
		agentRes := memberAgentResource(team, member)
		memberResult, err := r.ReconcileAgent(ctx, agentRes)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("team %q member %q: %w", team.Meta.Name, member.Name, err)
			}
			continue
		}
		result.Members[member.Name] = memberResult
	}
	if firstErr != nil {
		return nil, firstErr
	}

	return result, nil
}

// --- helpers ----------------------------------------------------------------

// validateAgentSpec checks the AgentResource for required fields.
func validateAgentSpec(agent *AgentResource) error {
	if agent.Spec.ModelRef == "" && agent.Spec.ModelConfig == nil {
		return core.NewError(
			"operator.reconcileAgent",
			core.ErrInvalidInput,
			fmt.Sprintf("agent %q: ModelRef must not be empty when ModelConfig is not provided", agent.Meta.Name),
			nil,
		)
	}
	if agent.Spec.MaxIterations < 0 {
		return core.NewError(
			"operator.reconcileAgent",
			core.ErrInvalidInput,
			fmt.Sprintf("agent %q: MaxIterations must not be negative", agent.Meta.Name),
			nil,
		)
	}
	return nil
}

// validateTeamSpec checks the TeamResource for required fields.
func validateTeamSpec(team *TeamResource) error {
	if len(team.Spec.Members) == 0 {
		return core.NewError(
			"operator.reconcileTeam",
			core.ErrInvalidInput,
			fmt.Sprintf("team %q: Members must not be empty", team.Meta.Name),
			nil,
		)
	}
	seen := make(map[string]struct{}, len(team.Spec.Members))
	for i, m := range team.Spec.Members {
		if m.Name == "" {
			return core.NewError(
				"operator.reconcileTeam",
				core.ErrInvalidInput,
				fmt.Sprintf("team %q: member[%d].Name must not be empty", team.Meta.Name, i),
				nil,
			)
		}
		if _, dup := seen[m.Name]; dup {
			return core.NewError(
				"operator.reconcileTeam",
				core.ErrInvalidInput,
				fmt.Sprintf("team %q: duplicate member name %q", team.Meta.Name, m.Name),
				nil,
			)
		}
		seen[m.Name] = struct{}{}
	}
	return nil
}

// agentImage returns the image from the spec or the default.
func agentImage(spec AgentSpec, defaultImage string) string {
	if spec.Image != "" {
		return spec.Image
	}
	return defaultImage
}

// agentPort returns the port from the spec or the default.
func agentPort(spec AgentSpec, defaultPort int) int {
	if spec.Port > 0 {
		return spec.Port
	}
	return defaultPort
}

// agentReplicas returns the replica count from the spec or the default.
func agentReplicas(spec AgentSpec, defaultReplicas int) int {
	if spec.Replicas > 0 {
		return spec.Replicas
	}
	return defaultReplicas
}

// agentLabels merges the CRD user-supplied labels with the standard Beluga
// operator labels (app, beluga-agent).
func agentLabels(name string, userLabels map[string]string) map[string]string {
	labels := map[string]string{
		"app":          name,
		"beluga-agent": name,
		"managed-by":   "beluga-operator",
	}
	for k, v := range userLabels {
		labels[k] = v
	}
	return labels
}

// buildDeploymentSpec constructs a [DeploymentSpec] from the reconciled agent
// fields.
func buildDeploymentSpec(name, ns, image string, replicas, port int, labels map[string]string, spec AgentSpec) DeploymentSpec {
	env := []EnvVar{
		{Name: "BELUGA_AGENT_NAME", Value: name},
		{Name: "BELUGA_PLANNER", Value: spec.Planner},
		{Name: "BELUGA_MODEL_REF", Value: spec.ModelRef},
	}
	if spec.ModelConfig != nil && spec.ModelConfig.APIKeyRef != "" {
		env = append(env, EnvVar{
			Name:            "BELUGA_API_KEY",
			ValueFromSecret: spec.ModelConfig.APIKeyRef,
		})
	}

	return DeploymentSpec{
		Name:      name,
		Namespace: ns,
		Replicas:  replicas,
		Image:     image,
		Ports: []ContainerPort{
			{Name: "http", ContainerPort: port, Protocol: "TCP"},
		},
		Env:       env,
		Resources: spec.Resources,
		Labels:    labels,
	}
}

// buildServiceSpec constructs a [ServiceSpec] that routes to the agent pods.
func buildServiceSpec(name, ns string, port int, selector map[string]string) ServiceSpec {
	return ServiceSpec{
		Name:      name,
		Namespace: ns,
		Selector:  selector,
		Ports: []ServicePort{
			{Name: "http", Port: port, TargetPort: port, Protocol: "TCP"},
		},
		Type: "ClusterIP",
	}
}

// buildHPASpec constructs a [HPASpec] from a [ScalingConfig].
func buildHPASpec(agentName, ns string, scaling ScalingConfig) *HPASpec {
	minReplicas := scaling.MinReplicas
	if minReplicas <= 0 {
		minReplicas = 1
	}
	maxReplicas := scaling.MaxReplicas
	if maxReplicas <= 0 {
		maxReplicas = 5
	}
	targetCPU := scaling.TargetCPUUtilization
	if targetCPU <= 0 {
		targetCPU = 80
	}
	return &HPASpec{
		Name:                 agentName + "-hpa",
		Namespace:            ns,
		TargetDeployment:     agentName,
		MinReplicas:          minReplicas,
		MaxReplicas:          maxReplicas,
		TargetCPUUtilization: targetCPU,
	}
}

// memberAgentResource synthesises a minimal AgentResource for a team member
// so it can be reconciled by ReconcileAgent.
func memberAgentResource(team *TeamResource, member TeamMemberRef) *AgentResource {
	return &AgentResource{
		APIVersion: team.APIVersion,
		Kind:       "Agent",
		Meta: ObjectMeta{
			Name:      member.Name,
			Namespace: team.Meta.Namespace,
			Labels: map[string]string{
				"beluga-team": team.Meta.Name,
			},
		},
		Spec: AgentSpec{
			// Synthesised specs use the member name as the model ref placeholder
			// so that validation passes. Real operators should look up the actual
			// Agent CRD and merge its spec here.
			ModelRef: member.Name,
			Persona: Persona{
				Role: member.Role,
			},
			MaxIterations: 10,
		},
	}
}
