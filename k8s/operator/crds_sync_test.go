package operator

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// FI-13: Helm installs CRDs only from the chart's crds/ directory. The copies
// in k8s/helm/crds must stay byte-identical to the canonical k8s/crds files.
func TestHelmCRDsMatchCanonical(t *testing.T) {
	const canonical = "../crds"
	const chart = "../helm/crds"

	list := func(dir string) []string {
		t.Helper()
		matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, filepath.Base(m))
		}
		sort.Strings(names)
		return names
	}

	want := list(canonical)
	got := list(chart)
	if len(want) == 0 {
		t.Fatal("no canonical CRDs found")
	}
	if len(want) != len(got) {
		t.Fatalf("CRD sets differ: k8s/crds=%v k8s/helm/crds=%v", want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Fatalf("CRD sets differ: k8s/crds=%v k8s/helm/crds=%v", want, got)
		}
		a, err := os.ReadFile(filepath.Join(canonical, want[i])) // #nosec G304 -- fixed test directory
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(chart, got[i])) // #nosec G304 -- fixed test directory
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s differs between k8s/crds and k8s/helm/crds; run: cp k8s/crds/*.yaml k8s/helm/crds/", want[i])
		}
	}
}
