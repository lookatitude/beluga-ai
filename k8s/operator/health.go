package operator

import (
	"net/http"

	"github.com/lookatitude/beluga-ai/v2/deploy"
)

// Probe paths served by [NewHealthHandler]. They match the liveness and
// readiness probes in the k8s/helm chart's operator Deployment.
const (
	// HealthzPath is the liveness probe path.
	HealthzPath = "/healthz"
	// ReadyzPath is the readiness probe path.
	ReadyzPath = "/readyz"
)

// NewHealthHandler returns an [http.Handler] that serves the operator's
// liveness probe at [HealthzPath] and readiness probe at [ReadyzPath], backed
// by h. Register readiness checks (informer caches synced, leader elected,
// webhook certs loaded, ...) on h with [deploy.HealthEndpoint.AddCheck]. When
// h is nil a fresh endpoint with no checks is used, so /readyz always reports
// ready. Every other path returns 404.
//
// The repository does not yet ship an operator binary; this is the wiring an
// operator main is expected to mount on the container's "http" port (the
// chart's service.targetPort, default 8080):
//
//	health := deploy.NewHealthEndpoint()
//	health.AddCheck("cache-synced", cacheSynced)
//	srv := &http.Server{
//	    Addr:              ":8080",
//	    Handler:           operator.NewHealthHandler(health),
//	    ReadHeaderTimeout: 5 * time.Second,
//	}
//
// The endpoints are unauthenticated by design (kubelet probes); restrict
// network access with a NetworkPolicy.
func NewHealthHandler(h *deploy.HealthEndpoint) http.Handler {
	if h == nil {
		h = deploy.NewHealthEndpoint()
	}
	mux := http.NewServeMux()
	mux.Handle(HealthzPath, h.Healthz())
	mux.Handle(ReadyzPath, h.Readyz())
	return mux
}
