package operator

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// ---- helpers ---------------------------------------------------------------

func agentWithModelRef(name, modelRef string) *AgentResource {
	return &AgentResource{
		APIVersion: "beluga.ai/v1",
		Kind:       "Agent",
		Meta:       ObjectMeta{Name: name, Namespace: "default"},
		Spec: AgentSpec{
			Persona:       Persona{Role: "assistant"},
			Planner:       "react",
			MaxIterations: 10,
			ModelRef:      modelRef,
		},
	}
}

func agentWithScaling(name string, scaling ScalingConfig) *AgentResource {
	a := agentWithModelRef(name, "openai-gpt4o")
	a.Spec.Scaling = scaling
	return a
}

// ---- DefaultReconciler tests -----------------------------------------------

func TestReconcileAgent_HappyPath(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithModelRef("planner", "openai-gpt4o")

	result, err := r.ReconcileAgent(context.Background(), agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Deployment
	if result.Deployment.Name != "planner" {
		t.Errorf("deployment name = %q, want %q", result.Deployment.Name, "planner")
	}
	if result.Deployment.Namespace != "default" {
		t.Errorf("deployment namespace = %q, want %q", result.Deployment.Namespace, "default")
	}
	if result.Deployment.Image != "beluga-agent:latest" {
		t.Errorf("deployment image = %q, want %q", result.Deployment.Image, "beluga-agent:latest")
	}
	if result.Deployment.Replicas != 1 {
		t.Errorf("deployment replicas = %d, want 1", result.Deployment.Replicas)
	}

	// Service
	if result.Service.Name != "planner" {
		t.Errorf("service name = %q, want %q", result.Service.Name, "planner")
	}
	if result.Service.Type != "ClusterIP" {
		t.Errorf("service type = %q, want ClusterIP", result.Service.Type)
	}

	// No HPA by default
	if result.HPA != nil {
		t.Errorf("expected nil HPA when scaling is disabled, got %+v", result.HPA)
	}
}

func TestReconcileAgent_MissingModelRef(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithModelRef("bad-agent", "")
	agent.Spec.ModelConfig = nil // explicitly nil

	_, err := r.ReconcileAgent(context.Background(), agent)
	if err == nil {
		t.Fatal("expected error for missing modelRef, got nil")
	}

	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected *core.Error, got %T: %v", err, err)
	}
	if coreErr.Code != core.ErrInvalidInput {
		t.Errorf("error code = %q, want %q", coreErr.Code, core.ErrInvalidInput)
	}
}

func TestReconcileAgent_NilAgent(t *testing.T) {
	r := NewDefaultReconciler()
	_, err := r.ReconcileAgent(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil agent, got nil")
	}
}

func TestReconcileAgent_ScalingEnabled_DefaultValues(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithScaling("executor", ScalingConfig{
		Enabled: true,
		// MinReplicas, MaxReplicas, TargetCPUUtilization all zero → should default
	})

	result, err := r.ReconcileAgent(context.Background(), agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HPA == nil {
		t.Fatal("expected non-nil HPA when scaling is enabled")
	}
	if result.HPA.MinReplicas != 1 {
		t.Errorf("HPA.MinReplicas = %d, want 1", result.HPA.MinReplicas)
	}
	if result.HPA.MaxReplicas != 5 {
		t.Errorf("HPA.MaxReplicas = %d, want 5", result.HPA.MaxReplicas)
	}
	if result.HPA.TargetCPUUtilization != 80 {
		t.Errorf("HPA.TargetCPUUtilization = %d, want 80", result.HPA.TargetCPUUtilization)
	}
	if result.HPA.TargetDeployment != "executor" {
		t.Errorf("HPA.TargetDeployment = %q, want %q", result.HPA.TargetDeployment, "executor")
	}
	if result.HPA.Name != "executor-hpa" {
		t.Errorf("HPA.Name = %q, want %q", result.HPA.Name, "executor-hpa")
	}
}

func TestReconcileAgent_ScalingEnabled_ExplicitValues(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithScaling("worker", ScalingConfig{
		Enabled:              true,
		MinReplicas:          2,
		MaxReplicas:          10,
		TargetCPUUtilization: 60,
	})

	result, err := r.ReconcileAgent(context.Background(), agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HPA.MinReplicas != 2 {
		t.Errorf("HPA.MinReplicas = %d, want 2", result.HPA.MinReplicas)
	}
	if result.HPA.MaxReplicas != 10 {
		t.Errorf("HPA.MaxReplicas = %d, want 10", result.HPA.MaxReplicas)
	}
	if result.HPA.TargetCPUUtilization != 60 {
		t.Errorf("HPA.TargetCPUUtilization = %d, want 60", result.HPA.TargetCPUUtilization)
	}
}

func TestReconcileAgent_CustomImageAndPort(t *testing.T) {
	r := NewDefaultReconciler(
		WithDefaultImage("my-registry/beluga:v2"),
		WithDefaultPort(9090),
	)
	agent := agentWithModelRef("agent", "anthropic-claude")

	result, err := r.ReconcileAgent(context.Background(), agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deployment.Image != "my-registry/beluga:v2" {
		t.Errorf("image = %q, want %q", result.Deployment.Image, "my-registry/beluga:v2")
	}
	if len(result.Deployment.Ports) == 0 || result.Deployment.Ports[0].ContainerPort != 9090 {
		t.Errorf("port = %v, want 9090", result.Deployment.Ports)
	}
}

func TestReconcileAgent_SpecOverridesDefaults(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithModelRef("agent", "openai-gpt4o")
	agent.Spec.Image = "custom:tag"
	agent.Spec.Port = 3000
	agent.Spec.Replicas = 4

	result, err := r.ReconcileAgent(context.Background(), agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deployment.Image != "custom:tag" {
		t.Errorf("image = %q, want %q", result.Deployment.Image, "custom:tag")
	}
	if result.Deployment.Replicas != 4 {
		t.Errorf("replicas = %d, want 4", result.Deployment.Replicas)
	}
	if result.Deployment.Ports[0].ContainerPort != 3000 {
		t.Errorf("port = %d, want 3000", result.Deployment.Ports[0].ContainerPort)
	}
}

func TestReconcileAgent_EnvVarsPresent(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithModelRef("planner", "openai-gpt4o")
	agent.Spec.Planner = "react"

	result, err := r.ReconcileAgent(context.Background(), agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envMap := make(map[string]string)
	for _, e := range result.Deployment.Env {
		envMap[e.Name] = e.Value
	}
	if envMap["BELUGA_AGENT_NAME"] != "planner" {
		t.Errorf("BELUGA_AGENT_NAME = %q, want %q", envMap["BELUGA_AGENT_NAME"], "planner")
	}
	if envMap["BELUGA_PLANNER"] != "react" {
		t.Errorf("BELUGA_PLANNER = %q, want %q", envMap["BELUGA_PLANNER"], "react")
	}
	if envMap["BELUGA_MODEL_REF"] != "openai-gpt4o" {
		t.Errorf("BELUGA_MODEL_REF = %q, want %q", envMap["BELUGA_MODEL_REF"], "openai-gpt4o")
	}
}

func TestReconcileAgent_APIKeyRefInEnv(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithModelRef("agent", "openai-gpt4o")
	agent.Spec.ModelConfig = &ModelConfig{
		Provider:  "openai",
		Model:     "gpt-4o",
		APIKeyRef: "my-secret/openai-key",
	}

	result, err := r.ReconcileAgent(context.Background(), agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var found bool
	for _, e := range result.Deployment.Env {
		if e.Name == "BELUGA_API_KEY" && e.ValueFromSecret == "my-secret/openai-key" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected BELUGA_API_KEY env var with ValueFromSecret, got %+v", result.Deployment.Env)
	}
}

func TestReconcileAgent_ContextCancelled(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithModelRef("agent", "openai-gpt4o")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := r.ReconcileAgent(ctx, agent)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestReconcileAgent_ServiceSelector(t *testing.T) {
	r := NewDefaultReconciler()
	agent := agentWithModelRef("my-agent", "openai-gpt4o")

	result, err := r.ReconcileAgent(context.Background(), agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Service.Selector["app"] != "my-agent" {
		t.Errorf("service selector[app] = %q, want %q", result.Service.Selector["app"], "my-agent")
	}
	if result.Service.Selector["managed-by"] != "beluga-operator" {
		t.Errorf("service selector[managed-by] = %q, want %q", result.Service.Selector["managed-by"], "beluga-operator")
	}
}

// ---- ReconcileTeam tests ---------------------------------------------------

func makeTeam(name string, members ...TeamMemberRef) *TeamResource {
	return &TeamResource{
		APIVersion: "beluga.ai/v1",
		Kind:       "Team",
		Meta:       ObjectMeta{Name: name, Namespace: "default"},
		Spec: TeamSpec{
			Members: members,
		},
	}
}

func TestReconcileTeam_HappyPath(t *testing.T) {
	r := NewDefaultReconciler()
	team := makeTeam("my-team",
		TeamMemberRef{Name: "planner", Role: "planner"},
		TeamMemberRef{Name: "executor", Role: "executor"},
	)

	result, err := r.ReconcileTeam(context.Background(), team)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(result.Members))
	}
	for _, name := range []string{"planner", "executor"} {
		if _, ok := result.Members[name]; !ok {
			t.Errorf("expected member %q in result", name)
		}
	}
}

func TestReconcileTeam_NilTeam(t *testing.T) {
	r := NewDefaultReconciler()
	_, err := r.ReconcileTeam(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil team, got nil")
	}
}

func TestReconcileTeam_EmptyMembers(t *testing.T) {
	r := NewDefaultReconciler()
	team := makeTeam("empty-team")

	_, err := r.ReconcileTeam(context.Background(), team)
	if err == nil {
		t.Fatal("expected error for empty members, got nil")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("expected *core.Error, got %T", err)
	}
	if coreErr.Code != core.ErrInvalidInput {
		t.Errorf("code = %q, want %q", coreErr.Code, core.ErrInvalidInput)
	}
}

func TestReconcileTeam_DuplicateMemberName(t *testing.T) {
	r := NewDefaultReconciler()
	team := makeTeam("dup-team",
		TeamMemberRef{Name: "agent-a"},
		TeamMemberRef{Name: "agent-a"},
	)

	_, err := r.ReconcileTeam(context.Background(), team)
	if err == nil {
		t.Fatal("expected error for duplicate member name, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected 'duplicate' in error message, got: %v", err)
	}
}

func TestReconcileTeam_EmptyMemberName(t *testing.T) {
	r := NewDefaultReconciler()
	team := makeTeam("bad-team",
		TeamMemberRef{Name: ""},
	)

	_, err := r.ReconcileTeam(context.Background(), team)
	if err == nil {
		t.Fatal("expected error for empty member name, got nil")
	}
}

func TestReconcileTeam_ContextCancelled(t *testing.T) {
	r := NewDefaultReconciler()
	team := makeTeam("my-team", TeamMemberRef{Name: "agent-a"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.ReconcileTeam(ctx, team)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestReconcileTeam_MemberLabels(t *testing.T) {
	r := NewDefaultReconciler()
	team := makeTeam("dream-team",
		TeamMemberRef{Name: "agent-x", Role: "worker"},
	)

	result, err := r.ReconcileTeam(context.Background(), team)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	memberResult := result.Members["agent-x"]
	if memberResult == nil {
		t.Fatal("missing member result for agent-x")
	}
	if memberResult.Deployment.Labels["beluga-team"] != "dream-team" {
		t.Errorf("expected beluga-team label to be %q, got %q", "dream-team", memberResult.Deployment.Labels["beluga-team"])
	}
}

// ---- Controller tests ------------------------------------------------------

// mockStore implements ResourceStore for tests.
type mockStore struct {
	agents map[string]*AgentResource
	teams  map[string]*TeamResource
}

func (m *mockStore) GetAgent(_ context.Context, _, name string) (*AgentResource, error) {
	a, ok := m.agents[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return a, nil
}

func (m *mockStore) GetTeam(_ context.Context, _, name string) (*TeamResource, error) {
	t, ok := m.teams[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}

// mockWriter implements StatusWriter for tests.
type mockWriter struct {
	agentStatuses map[string]AgentStatus
	teamStatuses  map[string]TeamStatus
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		agentStatuses: make(map[string]AgentStatus),
		teamStatuses:  make(map[string]TeamStatus),
	}
}

func (w *mockWriter) UpdateAgentStatus(_ context.Context, _, name string, status AgentStatus) error {
	w.agentStatuses[name] = status
	return nil
}

func (w *mockWriter) UpdateTeamStatus(_ context.Context, _, name string, status TeamStatus) error {
	w.teamStatuses[name] = status
	return nil
}

func TestController_ReconcileAgent(t *testing.T) {
	store := &mockStore{
		agents: map[string]*AgentResource{
			"planner": agentWithModelRef("planner", "openai-gpt4o"),
		},
	}
	writer := newMockWriter()
	r := NewDefaultReconciler()
	ctrl := NewController(r, store, writer, WithWorkers(1))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := ctrl.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	ctrl.EnqueueAgent("default", "planner")

	// Give the worker time to process the request.
	time.Sleep(100 * time.Millisecond)
	ctrl.Stop()

	status, ok := writer.agentStatuses["planner"]
	if !ok {
		t.Fatal("expected agent status to be written")
	}
	if status.Phase != "Running" {
		t.Errorf("status.Phase = %q, want Running", status.Phase)
	}
}

func TestController_ReconcileTeam(t *testing.T) {
	store := &mockStore{
		teams: map[string]*TeamResource{
			"my-team": makeTeam("my-team",
				TeamMemberRef{Name: "agent-a"},
				TeamMemberRef{Name: "agent-b"},
			),
		},
	}
	writer := newMockWriter()
	r := NewDefaultReconciler()
	ctrl := NewController(r, store, writer, WithWorkers(1))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := ctrl.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	ctrl.EnqueueTeam("default", "my-team")

	time.Sleep(100 * time.Millisecond)
	ctrl.Stop()

	status, ok := writer.teamStatuses["my-team"]
	if !ok {
		t.Fatal("expected team status to be written")
	}
	if status.Phase != "Running" {
		t.Errorf("status.Phase = %q, want Running", status.Phase)
	}
	if status.ActiveMembers != 2 {
		t.Errorf("status.ActiveMembers = %d, want 2", status.ActiveMembers)
	}
}

func TestController_StoreNotFoundSetsFailedStatus(t *testing.T) {
	store := &mockStore{
		agents: map[string]*AgentResource{}, // empty — all lookups will fail
	}
	writer := newMockWriter()
	r := NewDefaultReconciler()
	ctrl := NewController(r, store, writer, WithWorkers(1))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := ctrl.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	ctrl.EnqueueAgent("default", "missing-agent")

	time.Sleep(100 * time.Millisecond)
	ctrl.Stop()

	// Status should NOT be written because the store returned an error before
	// reconciliation could happen.
	if _, ok := writer.agentStatuses["missing-agent"]; ok {
		t.Error("expected no status to be written for a store lookup failure")
	}
}

func TestController_StartIsIdempotent(t *testing.T) {
	store := &mockStore{agents: map[string]*AgentResource{}}
	writer := newMockWriter()
	r := NewDefaultReconciler()
	ctrl := NewController(r, store, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := ctrl.Start(ctx); err != nil {
		t.Fatalf("first Start failed: %v", err)
	}
	if err := ctrl.Start(ctx); err != nil {
		t.Fatalf("second Start failed: %v", err)
	}
	ctrl.Stop()
}

func TestController_StopIsIdempotent(t *testing.T) {
	store := &mockStore{agents: map[string]*AgentResource{}}
	writer := newMockWriter()
	r := NewDefaultReconciler()
	ctrl := NewController(r, store, writer)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := ctrl.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	ctrl.Stop()
	ctrl.Stop() // must not panic
}

// ---- Table-driven reconciliation tests ------------------------------------

func TestReconcileAgent_Table(t *testing.T) {
	tests := []struct {
		name      string
		agent     *AgentResource
		wantErr   bool
		errCode   core.ErrorCode
		checkFunc func(t *testing.T, result *ReconcileResult)
	}{
		{
			name:    "valid agent produces ClusterIP service",
			agent:   agentWithModelRef("agent-a", "openai-gpt4o"),
			wantErr: false,
			checkFunc: func(t *testing.T, r *ReconcileResult) {
				if r.Service.Type != "ClusterIP" {
					t.Errorf("service type = %q, want ClusterIP", r.Service.Type)
				}
			},
		},
		{
			name:    "missing modelRef is invalid input",
			agent:   agentWithModelRef("agent-b", ""),
			wantErr: true,
			errCode: core.ErrInvalidInput,
		},
		{
			name: "negative MaxIterations is invalid input",
			agent: func() *AgentResource {
				a := agentWithModelRef("agent-c", "openai-gpt4o")
				a.Spec.MaxIterations = -1
				return a
			}(),
			wantErr: true,
			errCode: core.ErrInvalidInput,
		},
		{
			name: "inline ModelConfig satisfies validation",
			agent: func() *AgentResource {
				a := agentWithModelRef("agent-d", "")
				a.Spec.ModelConfig = &ModelConfig{Provider: "openai", Model: "gpt-4o"}
				return a
			}(),
			wantErr: false,
			checkFunc: func(t *testing.T, r *ReconcileResult) {
				if r.Deployment.Name != "agent-d" {
					t.Errorf("deployment name = %q, want %q", r.Deployment.Name, "agent-d")
				}
			},
		},
		{
			name:  "scaling disabled → no HPA",
			agent: agentWithScaling("agent-e", ScalingConfig{Enabled: false}),
			checkFunc: func(t *testing.T, r *ReconcileResult) {
				if r.HPA != nil {
					t.Errorf("expected nil HPA, got %+v", r.HPA)
				}
			},
		},
		{
			name:  "scaling enabled → HPA with name suffix",
			agent: agentWithScaling("agent-f", ScalingConfig{Enabled: true, MaxReplicas: 3}),
			checkFunc: func(t *testing.T, r *ReconcileResult) {
				if r.HPA == nil {
					t.Fatal("expected non-nil HPA")
				}
				if r.HPA.Name != "agent-f-hpa" {
					t.Errorf("HPA name = %q, want %q", r.HPA.Name, "agent-f-hpa")
				}
				if r.HPA.MaxReplicas != 3 {
					t.Errorf("HPA.MaxReplicas = %d, want 3", r.HPA.MaxReplicas)
				}
			},
		},
	}

	rec := NewDefaultReconciler()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rec.ReconcileAgent(context.Background(), tt.agent)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errCode != "" {
					var coreErr *core.Error
					if !errors.As(err, &coreErr) {
						t.Fatalf("expected *core.Error, got %T", err)
					}
					if coreErr.Code != tt.errCode {
						t.Errorf("error code = %q, want %q", coreErr.Code, tt.errCode)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}
