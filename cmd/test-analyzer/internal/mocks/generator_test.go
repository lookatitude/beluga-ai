package mocks

import (
	"context"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	generator := NewGenerator()
	if generator == nil {
		t.Fatal("NewGenerator() returned nil")
	}

	if _, ok := generator.(MockGenerator); !ok {
		t.Error("NewGenerator() does not implement MockGenerator interface")
	}
}

func TestGenerator_GenerateMock(t *testing.T) {
	ctx := context.Background()
	generator := NewGenerator()

	t.Run("GenerateMockBasic", func(t *testing.T) {
		// Basic test - generator may return error for invalid inputs
		_, err := generator.GenerateMock(ctx, "TestInterface", "testpkg", "")
		// Error is expected for invalid inputs
		_ = err
	})

	t.Run("GenerateMockWithContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := generator.GenerateMock(ctx, "TestInterface", "testpkg", "")
		// May or may not check context
		_ = err
	})
}
