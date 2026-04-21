package devloop

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestGoSourceFilter_Accept(t *testing.T) {
	t.Parallel()
	f := GoSourceFilter{}
	cases := []struct {
		name string
		path string
		op   fsnotify.Op
		want bool
	}{
		{"write .go file", "/proj/main.go", fsnotify.Write, true},
		{"create .go file", "/proj/pkg/foo.go", fsnotify.Create, true},
		{"rename .go file", "/proj/main.go", fsnotify.Rename, true},
		{"chmod is ignored", "/proj/main.go", fsnotify.Chmod, false},
		{"remove is ignored", "/proj/main.go", fsnotify.Remove, false},
		{"non-.go extension", "/proj/README.md", fsnotify.Write, false},
		{"emacs scratch", "/proj/.#main.go", fsnotify.Write, false},
		{"trailing tilde", "/proj/main.go~", fsnotify.Write, false},
		{"under .git", "/proj/.git/main.go", fsnotify.Write, false},
		{"under vendor", "/proj/vendor/lib/x.go", fsnotify.Write, false},
		{"under node_modules", "/proj/node_modules/pkg/x.go", fsnotify.Write, false},
		{"nested directory ok", "/proj/agent/planner/react.go", fsnotify.Write, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := f.Accept(tc.path, tc.op); got != tc.want {
				t.Fatalf("Accept(%q, %v) = %v, want %v", tc.path, tc.op, got, tc.want)
			}
		})
	}
}
