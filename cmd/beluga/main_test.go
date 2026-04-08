package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdInit(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "myproject")

	err := cmdInit([]string{"-name", "test-project", "-dir", projDir})
	if err != nil {
		t.Fatalf("cmdInit: %v", err)
	}

	// Verify directories were created.
	for _, sub := range []string{"agents", "tools", "config"} {
		path := filepath.Join(projDir, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %s not created: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", sub)
		}
	}

	// Verify config file.
	configPath := filepath.Join(projDir, "config", "agent.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if len(data) == 0 {
		t.Error("config file is empty")
	}

	// Verify main.go.
	mainPath := filepath.Join(projDir, "main.go")
	data, err = os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if len(data) == 0 {
		t.Error("main.go is empty")
	}
}

func TestCmdInit_PathTraversal(t *testing.T) {
	err := cmdInit([]string{"-dir", "/tmp/../etc/passwd"})
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestCmdDev(t *testing.T) {
	// Just verify it doesn't error.
	err := cmdDev([]string{"-port", "9090"})
	if err != nil {
		t.Errorf("cmdDev: %v", err)
	}
}

func TestCmdDeploy(t *testing.T) {
	tests := []struct {
		target  string
		wantErr bool
	}{
		{"docker", false},
		{"compose", false},
		{"k8s", false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			err := cmdDeploy([]string{"-target", tt.target})
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
