package operator

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/deploy"
)

// FI-13: the Helm chart probes /healthz and /readyz on the operator's http
// port; NewHealthHandler is the wiring an operator binary mounts there.
func TestNewHealthHandler(t *testing.T) {
	failing := deploy.NewHealthEndpoint()
	failing.AddCheck("cache-synced", func(context.Context) error { return errors.New("not synced") })

	passing := deploy.NewHealthEndpoint()
	passing.AddCheck("cache-synced", func(context.Context) error { return nil })

	tests := []struct {
		name   string
		h      *deploy.HealthEndpoint
		method string
		path   string
		want   int
	}{
		{"healthz nil endpoint", nil, http.MethodGet, HealthzPath, http.StatusOK},
		{"readyz nil endpoint", nil, http.MethodGet, ReadyzPath, http.StatusOK},
		{"healthz ok even if not ready", failing, http.MethodGet, HealthzPath, http.StatusOK},
		{"readyz failing check", failing, http.MethodGet, ReadyzPath, http.StatusServiceUnavailable},
		{"readyz passing check", passing, http.MethodGet, ReadyzPath, http.StatusOK},
		{"head allowed", passing, http.MethodHead, ReadyzPath, http.StatusOK},
		{"post rejected", passing, http.MethodPost, HealthzPath, http.StatusMethodNotAllowed},
		{"unknown path", passing, http.MethodGet, "/metrics", http.StatusNotFound},
		{"subpath not matched", passing, http.MethodGet, HealthzPath + "/x", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			NewHealthHandler(tt.h).ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil))
			if rec.Code != tt.want {
				t.Errorf("%s %s = %d, want %d", tt.method, tt.path, rec.Code, tt.want)
			}
		})
	}
}

func TestHealthPathsMatchChartProbes(t *testing.T) {
	if HealthzPath != "/healthz" || ReadyzPath != "/readyz" {
		t.Errorf("paths changed (%q, %q); update k8s/helm/templates/deployment.yaml probes", HealthzPath, ReadyzPath)
	}
}
