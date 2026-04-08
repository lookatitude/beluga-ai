package learning

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestASTValidator_Validate(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		code    string
		wantErr string
	}{
		{
			name: "valid code with allowed imports",
			code: `package main

import "fmt"

func run(input string) (string, error) {
	return fmt.Sprintf("hello %s", input), nil
}
`,
		},
		{
			name: "valid code with no imports",
			code: `package main

func run(input string) (string, error) {
	return "hello", nil
}
`,
		},
		{
			name: "disallowed import",
			code: `package main

import "os/exec"

func run(input string) (string, error) {
	return "", nil
}
`,
			wantErr: `disallowed import "os/exec"`,
		},
		{
			name: "goroutine spawning",
			code: `package main

import "fmt"

func run(input string) (string, error) {
	go fmt.Println("bad")
	return "", nil
}
`,
			wantErr: "goroutine spawning",
		},
		{
			name: "unsafe package usage",
			code: `package main

import "unsafe"

func run(input string) (string, error) {
	_ = unsafe.Sizeof(0)
	return "", nil
}
`,
			wantErr: "unsafe",
		},
		{
			name: "syntax error",
			code: `package main

func run(input string (string, error) {
`,
			wantErr: "parse error",
		},
		{
			name:    "custom allowed imports",
			allowed: []string{"net/url"},
			code: `package main

import "net/url"

func run(input string) (string, error) {
	u, _ := url.Parse(input)
	return u.Host, nil
}
`,
		},
		{
			name:    "custom allowed rejects default",
			allowed: []string{"net/url"},
			code: `package main

import "fmt"

func run(input string) (string, error) {
	return fmt.Sprintf("%s", input), nil
}
`,
			wantErr: `disallowed import "fmt"`,
		},
		{
			name: "multiple imports mixed",
			code: `package main

import (
	"fmt"
	"os"
)

func run(input string) (string, error) {
	_ = os.Getenv("x")
	return fmt.Sprintf("%s", input), nil
}
`,
			wantErr: `disallowed import "os"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewASTValidator(tt.allowed)
			err := v.Validate(tt.code)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNoopExecutor(t *testing.T) {
	t.Run("returns configured response", func(t *testing.T) {
		exec := &NoopExecutor{Response: "hello world"}
		out, err := exec.Execute(context.Background(), "code", "input")
		require.NoError(t, err)
		assert.Equal(t, "hello world", out)
	})

	t.Run("returns configured error", func(t *testing.T) {
		exec := &NoopExecutor{Err: fmt.Errorf("boom")}
		_, err := exec.Execute(context.Background(), "code", "input")
		require.Error(t, err)
		assert.Equal(t, "boom", err.Error())
	})

	t.Run("returns both", func(t *testing.T) {
		exec := &NoopExecutor{Response: "partial", Err: fmt.Errorf("fail")}
		out, err := exec.Execute(context.Background(), "", "")
		assert.Equal(t, "partial", out)
		assert.Error(t, err)
	})
}
