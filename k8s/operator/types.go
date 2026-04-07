package operator

// ObjectMeta holds common Kubernetes-style metadata for a custom resource.
// It mirrors the subset of k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta
// that the operator needs without importing the K8s library.
type ObjectMeta struct {
	// Name is the name of the resource. Must be unique within a namespace.
	Name string `json:"name"`

	// Namespace is the namespace the resource belongs to.
	Namespace string `json:"namespace"`

	// Labels is a map of string key-value pairs attached to the resource.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is a map of string key-value pairs for non-identifying metadata.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Persona defines the agent's identity and conversational behaviour.
type Persona struct {
	// Role is the agent's functional title, e.g. "Planner" or "Executor".
	Role string `json:"role"`

	// Goal is the high-level objective the agent pursues.
	Goal string `json:"goal"`

	// Backstory provides context that shapes the agent's responses.
	Backstory string `json:"backstory,omitempty"`
}

// ModelConfig describes how the agent communicates with its language model.
type ModelConfig struct {
	// Provider is the LLM provider name, e.g. "openai" or "anthropic".
	Provider string `json:"provider"`

	// Model is the model identifier, e.g. "gpt-4o" or "claude-3-5-sonnet".
	Model string `json:"model"`

	// Temperature controls the sampling randomness. Range [0, 2].
	Temperature float64 `json:"temperature,omitempty"`

	// MaxTokens caps the number of tokens in each response.
	MaxTokens int `json:"maxTokens,omitempty"`

	// APIKeyRef names the Kubernetes Secret key that holds the provider API key.
	// Format: "<secret-name>/<key>".
	APIKeyRef string `json:"apiKeyRef,omitempty"`
}

// ResourceRequirements specifies compute resource requests and limits.
type ResourceRequirements struct {
	// CPU is the CPU request/limit in Kubernetes quantity notation, e.g. "500m".
	CPU string `json:"cpu,omitempty"`

	// Memory is the memory request/limit in Kubernetes quantity notation, e.g. "256Mi".
	Memory string `json:"memory,omitempty"`
}

// Resources groups the resource requests and limits for a container.
type Resources struct {
	// Requests are the minimum resources required.
	Requests ResourceRequirements `json:"requests,omitempty"`

	// Limits are the maximum resources the container may consume.
	Limits ResourceRequirements `json:"limits,omitempty"`
}

// ScalingConfig controls Horizontal Pod Autoscaler behaviour.
type ScalingConfig struct {
	// Enabled activates the HPA for this agent.
	Enabled bool `json:"enabled"`

	// MinReplicas is the minimum number of pod replicas.
	// Defaults to 1 when zero.
	MinReplicas int `json:"minReplicas,omitempty"`

	// MaxReplicas is the maximum number of pod replicas.
	MaxReplicas int `json:"maxReplicas,omitempty"`

	// TargetCPUUtilization is the average CPU utilization percentage that
	// triggers scaling.
	TargetCPUUtilization int `json:"targetCPUUtilization,omitempty"`
}

// AgentSpec is the Go representation of the Agent CRD spec.
type AgentSpec struct {
	// Persona defines the agent's role, goal, and backstory.
	Persona Persona `json:"persona"`

	// Planner is the planner implementation name, e.g. "react" or "openai-functions".
	Planner string `json:"planner"`

	// MaxIterations caps the number of reasoning iterations per invocation.
	// Must be > 0.
	MaxIterations int `json:"maxIterations"`

	// ModelRef references a ModelConfig by name (key into a ConfigMap or Secret).
	// Must not be empty.
	ModelRef string `json:"modelRef"`

	// ModelConfig is the inline model configuration. Takes precedence over
	// ModelRef when both are set.
	ModelConfig *ModelConfig `json:"modelConfig,omitempty"`

	// ToolRefs lists the names of Tool CRDs this agent may invoke.
	ToolRefs []string `json:"toolRefs,omitempty"`

	// Replicas is the desired number of pod replicas. Defaults to 1.
	Replicas int `json:"replicas,omitempty"`

	// Resources specifies compute resource requests and limits for the pod.
	Resources Resources `json:"resources,omitempty"`

	// Scaling configures the HPA for this agent.
	Scaling ScalingConfig `json:"scaling,omitempty"`

	// Port is the container port the agent's HTTP server listens on.
	// Defaults to 8080.
	Port int `json:"port,omitempty"`

	// Image is the container image to deploy, e.g. "beluga-agent:latest".
	// Defaults to "beluga-agent:latest".
	Image string `json:"image,omitempty"`
}

// AgentStatus reflects the observed state of an Agent CRD.
type AgentStatus struct {
	// ReadyReplicas is the number of pods currently in the Ready state.
	ReadyReplicas int `json:"readyReplicas"`

	// Phase is a short, human-readable summary of the agent lifecycle state.
	// Typical values: "Pending", "Running", "Failed".
	Phase string `json:"phase,omitempty"`

	// Message provides additional human-readable information about the current state.
	Message string `json:"message,omitempty"`
}

// AgentResource is the top-level Go representation of the Agent custom resource.
// It mirrors the structure of a Kubernetes object without importing K8s types.
type AgentResource struct {
	// APIVersion identifies the versioned schema, e.g. "beluga.ai/v1".
	APIVersion string `json:"apiVersion"`

	// Kind is always "Agent".
	Kind string `json:"kind"`

	// Meta holds the resource's name, namespace, labels, and annotations.
	Meta ObjectMeta `json:"metadata"`

	// Spec is the desired state declared by the user.
	Spec AgentSpec `json:"spec"`

	// Status is the last observed state written by the controller.
	Status AgentStatus `json:"status,omitempty"`
}

// TeamMemberRef references an Agent CRD by name within the same namespace.
type TeamMemberRef struct {
	// Name is the Agent CRD name.
	Name string `json:"name"`

	// Role describes the member's function within the team, e.g. "planner".
	Role string `json:"role,omitempty"`
}

// TeamSpec is the Go representation of the Team CRD spec.
type TeamSpec struct {
	// Members lists the agent CRDs that form this team.
	Members []TeamMemberRef `json:"members"`

	// Supervisor is the optional name of the agent that orchestrates the team.
	Supervisor string `json:"supervisor,omitempty"`

	// MaxConcurrent limits how many member agents may run simultaneously.
	// Zero means unlimited.
	MaxConcurrent int `json:"maxConcurrent,omitempty"`

	// SharedToolRefs lists Tool CRDs available to all members.
	SharedToolRefs []string `json:"sharedToolRefs,omitempty"`
}

// TeamStatus reflects the observed state of a Team CRD.
type TeamStatus struct {
	// ActiveMembers is the number of member agents currently running.
	ActiveMembers int `json:"activeMembers"`

	// Phase is a short summary of the team's lifecycle state.
	Phase string `json:"phase,omitempty"`

	// Message provides additional human-readable information.
	Message string `json:"message,omitempty"`
}

// TeamResource is the top-level Go representation of the Team custom resource.
type TeamResource struct {
	// APIVersion identifies the versioned schema, e.g. "beluga.ai/v1".
	APIVersion string `json:"apiVersion"`

	// Kind is always "Team".
	Kind string `json:"kind"`

	// Meta holds the resource's name, namespace, labels, and annotations.
	Meta ObjectMeta `json:"metadata"`

	// Spec is the desired state declared by the user.
	Spec TeamSpec `json:"spec"`

	// Status is the last observed state written by the controller.
	Status TeamStatus `json:"status,omitempty"`
}

// --- Desired-state spec types (no K8s library imports) ---

// EnvVar represents a container environment variable.
type EnvVar struct {
	// Name is the environment variable name.
	Name string `json:"name"`

	// Value is the literal value. Mutually exclusive with ValueFromSecret.
	Value string `json:"value,omitempty"`

	// ValueFromSecret references a Kubernetes Secret key.
	// Format: "<secret-name>/<key>".
	ValueFromSecret string `json:"valueFromSecret,omitempty"`
}

// ContainerPort describes a port exposed by the container.
type ContainerPort struct {
	// Name is an optional label for the port.
	Name string `json:"name,omitempty"`

	// ContainerPort is the port number.
	ContainerPort int `json:"containerPort"`

	// Protocol is the network protocol. Defaults to "TCP".
	Protocol string `json:"protocol,omitempty"`
}

// DeploymentSpec is the desired-state specification for a Kubernetes Deployment,
// expressed without importing any K8s types.
type DeploymentSpec struct {
	// Name is the Deployment resource name.
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// Replicas is the desired number of pod replicas.
	Replicas int `json:"replicas"`

	// Image is the container image reference.
	Image string `json:"image"`

	// Ports lists the container ports to expose.
	Ports []ContainerPort `json:"ports,omitempty"`

	// Env lists the environment variables to inject.
	Env []EnvVar `json:"env,omitempty"`

	// Resources specifies compute resource requests and limits.
	Resources Resources `json:"resources,omitempty"`

	// Labels are applied to both the Deployment and its pod template.
	Labels map[string]string `json:"labels,omitempty"`
}

// ServicePort describes a port exposed by a Kubernetes Service.
type ServicePort struct {
	// Name is an optional label for the port.
	Name string `json:"name,omitempty"`

	// Port is the port number exposed by the Service.
	Port int `json:"port"`

	// TargetPort is the port on the container to forward traffic to.
	TargetPort int `json:"targetPort"`

	// Protocol is the network protocol. Defaults to "TCP".
	Protocol string `json:"protocol,omitempty"`
}

// ServiceSpec is the desired-state specification for a Kubernetes Service,
// expressed without importing any K8s types.
type ServiceSpec struct {
	// Name is the Service resource name.
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// Selector maps to the pods the service routes to.
	Selector map[string]string `json:"selector"`

	// Ports lists the ports exposed by the Service.
	Ports []ServicePort `json:"ports,omitempty"`

	// Type is the Service type. Defaults to "ClusterIP".
	Type string `json:"type,omitempty"`
}

// HPASpec is the desired-state specification for a Kubernetes HorizontalPodAutoscaler,
// expressed without importing any K8s types.
type HPASpec struct {
	// Name is the HPA resource name.
	Name string `json:"name"`

	// Namespace is the target namespace.
	Namespace string `json:"namespace"`

	// TargetDeployment is the name of the Deployment managed by this HPA.
	TargetDeployment string `json:"targetDeployment"`

	// MinReplicas is the minimum number of replicas.
	MinReplicas int `json:"minReplicas"`

	// MaxReplicas is the maximum number of replicas.
	MaxReplicas int `json:"maxReplicas"`

	// TargetCPUUtilization is the average CPU utilization percentage threshold.
	TargetCPUUtilization int `json:"targetCPUUtilization"`
}

// ReconcileResult describes the complete desired Kubernetes state produced by
// reconciling a single Agent or Team member.
type ReconcileResult struct {
	// Deployment is the desired Deployment specification.
	Deployment DeploymentSpec `json:"deployment"`

	// Service is the desired Service specification.
	Service ServiceSpec `json:"service"`

	// HPA is the desired HorizontalPodAutoscaler specification.
	// Nil when autoscaling is not enabled.
	HPA *HPASpec `json:"hpa,omitempty"`
}

// TeamReconcileResult aggregates the per-member reconciliation results for a
// Team CRD.
type TeamReconcileResult struct {
	// Members maps each member agent name to its ReconcileResult.
	Members map[string]*ReconcileResult `json:"members"`
}
