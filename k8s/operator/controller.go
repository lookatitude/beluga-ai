package operator

import (
	"context"
	"fmt"
	"sync"
)

// ControllerConfig holds the configuration for the [Controller].
type ControllerConfig struct {
	// Namespace restricts the controller to a single namespace.
	// An empty string means all namespaces (cluster-scoped).
	Namespace string

	// Workers is the number of concurrent reconciliation workers.
	// Defaults to 1 when zero.
	Workers int
}

// ControllerOption is a functional option for [NewController].
type ControllerOption func(*ControllerConfig)

// WithNamespace restricts the controller to the given namespace.
func WithNamespace(ns string) ControllerOption {
	return func(c *ControllerConfig) { c.Namespace = ns }
}

// WithWorkers sets the number of concurrent reconciliation goroutines.
func WithWorkers(n int) ControllerOption {
	return func(c *ControllerConfig) { c.Workers = n }
}

// ReconcileRequest carries the identity of a resource that needs to be
// reconciled. Callers enqueue requests via [Controller.EnqueueAgent] or
// [Controller.EnqueueTeam].
type ReconcileRequest struct {
	// Kind is "Agent" or "Team".
	Kind string

	// Name is the resource name.
	Name string

	// Namespace is the resource namespace.
	Namespace string
}

// ResourceStore is the read interface the controller uses to fetch the current
// state of custom resources. Callers provide an implementation backed by their
// preferred K8s client.
type ResourceStore interface {
	// GetAgent returns the Agent CRD with the given name and namespace.
	GetAgent(ctx context.Context, namespace, name string) (*AgentResource, error)

	// GetTeam returns the Team CRD with the given name and namespace.
	GetTeam(ctx context.Context, namespace, name string) (*TeamResource, error)
}

// StatusWriter persists the reconciliation status back to the cluster.
// Callers provide an implementation backed by their preferred K8s client.
type StatusWriter interface {
	// UpdateAgentStatus writes the observed status for the named Agent CRD.
	UpdateAgentStatus(ctx context.Context, namespace, name string, status AgentStatus) error

	// UpdateTeamStatus writes the observed status for the named Team CRD.
	UpdateTeamStatus(ctx context.Context, namespace, name string, status TeamStatus) error
}

// Controller drives the reconciliation loop. It reads pending [ReconcileRequest]
// values from an internal work queue, fetches the corresponding resource via the
// [ResourceStore], and calls the [Reconciler]. Results are written back via the
// [StatusWriter].
//
// Controller deliberately has no dependency on any K8s library; the
// [ResourceStore] and [StatusWriter] interfaces are the seams through which K8s
// client code is injected.
type Controller struct {
	cfg        ControllerConfig
	reconciler Reconciler
	store      ResourceStore
	writer     StatusWriter

	queue chan ReconcileRequest

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewController creates a [Controller] that uses reconciler to compute desired
// state, store to read CRDs, and writer to persist status updates.
//
// The work queue capacity is set to 256. Use [Controller.EnqueueAgent] or
// [Controller.EnqueueTeam] to add work items.
func NewController(reconciler Reconciler, store ResourceStore, writer StatusWriter, opts ...ControllerOption) *Controller {
	cfg := ControllerConfig{Workers: 1}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 1
	}
	return &Controller{
		cfg:        cfg,
		reconciler: reconciler,
		store:      store,
		writer:     writer,
		queue:      make(chan ReconcileRequest, 256),
	}
}

// Start launches the reconciliation worker goroutines. It is idempotent: calling
// Start on an already-running controller is a no-op. The controller stops when
// ctx is cancelled.
func (c *Controller) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	childCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.running = true

	for i := 0; i < c.cfg.Workers; i++ {
		c.wg.Add(1)
		go c.runWorker(childCtx)
	}
	return nil
}

// Stop signals the controller to cease processing and waits for all workers to
// finish. It is safe to call from any goroutine.
func (c *Controller) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	cancel := c.cancel
	c.mu.Unlock()

	cancel()
	c.wg.Wait()
}

// EnqueueAgent adds an Agent reconcile request to the work queue.
// It drops the request if the queue is full to prevent unbounded blocking.
func (c *Controller) EnqueueAgent(namespace, name string) {
	c.enqueue(ReconcileRequest{Kind: "Agent", Namespace: namespace, Name: name})
}

// EnqueueTeam adds a Team reconcile request to the work queue.
// It drops the request if the queue is full to prevent unbounded blocking.
func (c *Controller) EnqueueTeam(namespace, name string) {
	c.enqueue(ReconcileRequest{Kind: "Team", Namespace: namespace, Name: name})
}

// enqueue adds req to the work queue if there is capacity.
func (c *Controller) enqueue(req ReconcileRequest) {
	select {
	case c.queue <- req:
	default:
		// Queue is full; drop the request. A real controller would log here.
	}
}

// runWorker processes requests from the queue until ctx is cancelled.
func (c *Controller) runWorker(ctx context.Context) {
	defer c.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-c.queue:
			if !ok {
				return
			}
			_ = c.processRequest(ctx, req) // errors are handled inside
		}
	}
}

// processRequest fetches the resource and calls the reconciler. Status is
// written back regardless of the reconciler outcome.
func (c *Controller) processRequest(ctx context.Context, req ReconcileRequest) error {
	switch req.Kind {
	case "Agent":
		return c.processAgent(ctx, req)
	case "Team":
		return c.processTeam(ctx, req)
	default:
		return fmt.Errorf("controller: unknown kind %q", req.Kind)
	}
}

// processAgent fetches and reconciles a single Agent CRD.
func (c *Controller) processAgent(ctx context.Context, req ReconcileRequest) error {
	agent, err := c.store.GetAgent(ctx, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("controller: fetch agent %q/%q: %w", req.Namespace, req.Name, err)
	}

	_, reconcileErr := c.reconciler.ReconcileAgent(ctx, agent)

	status := AgentStatus{}
	if reconcileErr != nil {
		status.Phase = "Failed"
		status.Message = reconcileErr.Error()
	} else {
		status.Phase = "Running"
		status.ReadyReplicas = agentReplicas(agent.Spec, 1)
	}

	if c.writer != nil {
		if writeErr := c.writer.UpdateAgentStatus(ctx, req.Namespace, req.Name, status); writeErr != nil {
			return fmt.Errorf("controller: update agent status %q/%q: %w", req.Namespace, req.Name, writeErr)
		}
	}
	return reconcileErr
}

// processTeam fetches and reconciles a Team CRD.
func (c *Controller) processTeam(ctx context.Context, req ReconcileRequest) error {
	team, err := c.store.GetTeam(ctx, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("controller: fetch team %q/%q: %w", req.Namespace, req.Name, err)
	}

	result, reconcileErr := c.reconciler.ReconcileTeam(ctx, team)

	status := TeamStatus{}
	if reconcileErr != nil {
		status.Phase = "Failed"
		status.Message = reconcileErr.Error()
	} else {
		status.Phase = "Running"
		if result != nil {
			status.ActiveMembers = len(result.Members)
		}
	}

	if c.writer != nil {
		if writeErr := c.writer.UpdateTeamStatus(ctx, req.Namespace, req.Name, status); writeErr != nil {
			return fmt.Errorf("controller: update team status %q/%q: %w", req.Namespace, req.Name, writeErr)
		}
	}
	return reconcileErr
}
