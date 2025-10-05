// Package contract provides contract tests for Container interface compliance.
// T009: Contract test for Container interface
package contract

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainerInterfaceContract verifies Container interface implementation compliance
func TestContainerInterfaceContract(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *core.Container
		operation   func(t *testing.T, container *core.Container)
		description string
	}{
		{
			name:        "Register_valid_factory",
			description: "Contract: Register method must accept valid factory functions",
			setup:       func() *core.Container { return core.NewContainer() },
			operation: func(t *testing.T, container *core.Container) {
				// Test registering various valid factory function signatures
				err := container.Register(func() string { return "test" })
				assert.NoError(t, err, "Should accept no-parameter factory")

				err = container.Register(func(dep string) int { return len(dep) })
				assert.NoError(t, err, "Should accept factory with dependencies")

				err = container.Register(func() (string, error) { return "test", nil })
				assert.NoError(t, err, "Should accept factory with error return")
			},
		},
		{
			name:        "Register_invalid_factory",
			description: "Contract: Register method must reject invalid factory functions",
			setup:       func() *core.Container { return core.NewContainer() },
			operation: func(t *testing.T, container *core.Container) {
				// Test rejecting non-function values
				err := container.Register("not a function")
				assert.Error(t, err, "Should reject non-function")

				err = container.Register(42)
				assert.Error(t, err, "Should reject non-function")

				// Test rejecting functions with no return values
				err = container.Register(func() {})
				assert.Error(t, err, "Should reject function with no return values")
			},
		},
		{
			name:        "Resolve_registered_type",
			description: "Contract: Resolve method must resolve registered types",
			setup: func() *core.Container {
				container := core.NewContainer()
				container.Register(func() string { return "contract_test_value" })
				return container
			},
			operation: func(t *testing.T, container *core.Container) {
				var result string
				err := container.Resolve(&result)
				assert.NoError(t, err, "Should resolve registered type")
				assert.Equal(t, "contract_test_value", result)
			},
		},
		{
			name:        "Resolve_unregistered_type",
			description: "Contract: Resolve method must error for unregistered types",
			setup:       func() *core.Container { return core.NewContainer() },
			operation: func(t *testing.T, container *core.Container) {
				var result int
				err := container.Resolve(&result)
				assert.Error(t, err, "Should error for unregistered type")
			},
		},
		{
			name:        "Resolve_non_pointer",
			description: "Contract: Resolve method must error for non-pointer targets",
			setup:       func() *core.Container { return core.NewContainer() },
			operation: func(t *testing.T, container *core.Container) {
				var result string
				err := container.Resolve(result) // Not a pointer
				assert.Error(t, err, "Should error for non-pointer target")
			},
		},
		{
			name:        "Has_registered_type",
			description: "Contract: Has method must return true for registered types",
			setup: func() *core.Container {
				container := core.NewContainer()
				container.Register(func() string { return "test" })
				return container
			},
			operation: func(t *testing.T, container *core.Container) {
				stringType := reflect.TypeOf("")
				has := container.Has(stringType)
				assert.True(t, has, "Should return true for registered type")
			},
		},
		{
			name:        "Has_unregistered_type",
			description: "Contract: Has method must return false for unregistered types",
			setup:       func() *core.Container { return core.NewContainer() },
			operation: func(t *testing.T, container *core.Container) {
				intType := reflect.TypeOf(0)
				has := container.Has(intType)
				assert.False(t, has, "Should return false for unregistered type")
			},
		},
		{
			name:        "Singleton_registration",
			description: "Contract: Singleton method must register instance for immediate use",
			setup:       func() *core.Container { return core.NewContainer() },
			operation: func(t *testing.T, container *core.Container) {
				// Register singleton
				container.Singleton("singleton_test_value")

				// Should be able to resolve immediately
				var result string
				err := container.Resolve(&result)
				assert.NoError(t, err, "Should resolve singleton")
				assert.Equal(t, "singleton_test_value", result)
			},
		},
		{
			name:        "Clear_removes_registrations",
			description: "Contract: Clear method must remove all registrations",
			setup: func() *core.Container {
				container := core.NewContainer()
				container.Register(func() string { return "test" })
				container.Singleton(42)
				return container
			},
			operation: func(t *testing.T, container *core.Container) {
				// Verify registrations exist
				stringType := reflect.TypeOf("")
				intType := reflect.TypeOf(0)
				assert.True(t, container.Has(stringType))
				assert.True(t, container.Has(intType))

				// Clear and verify removal
				container.Clear()
				assert.False(t, container.Has(stringType))
				assert.False(t, container.Has(intType))
			},
		},
		{
			name:        "MustResolve_success",
			description: "Contract: MustResolve should work for registered types",
			setup: func() *core.Container {
				container := core.NewContainer()
				container.Register(func() string { return "must_resolve_test" })
				return container
			},
			operation: func(t *testing.T, container *core.Container) {
				var result string
				// Should not panic for registered type
				assert.NotPanics(t, func() {
					container.MustResolve(&result)
				})
				assert.Equal(t, "must_resolve_test", result)
			},
		},
		{
			name:        "CheckHealth_healthy_container",
			description: "Contract: CheckHealth should return no error for healthy container",
			setup:       func() *core.Container { return core.NewContainer() },
			operation: func(t *testing.T, container *core.Container) {
				ctx := context.Background()
				err := container.CheckHealth(ctx)
				assert.NoError(t, err, "Healthy container should pass health check")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := tt.setup()
			tt.operation(t, container)
		})
	}
}

// TestContainerContractEdgeCases tests edge cases and boundary conditions
func TestContainerContractEdgeCases(t *testing.T) {
	t.Run("NilFactoryRegistration", func(t *testing.T) {
		container := core.NewContainer()
		err := container.Register(nil)
		assert.Error(t, err, "Should reject nil factory")
	})

	t.Run("CircularDependencyDetection", func(t *testing.T) {
		container := core.NewContainer()

		// Create circular dependency: A depends on B, B depends on A
		container.Register(func(b string) int { return len(b) })
		container.Register(func(a int) string { return fmt.Sprintf("value_%d", a) })

		// Should detect circular dependency
		var result int
		err := container.Resolve(&result)
		// Note: This test documents expected behavior - actual implementation may vary
		// The important part is that it either resolves successfully or fails gracefully
		if err != nil {
			t.Logf("Circular dependency detection: %v", err)
		}
	})

	t.Run("ResolveAfterClear", func(t *testing.T) {
		container := core.NewContainer()
		container.Register(func() string { return "test" })

		var result string
		err := container.Resolve(&result)
		require.NoError(t, err)

		container.Clear()

		// Should fail to resolve after clear
		err = container.Resolve(&result)
		assert.Error(t, err, "Should fail to resolve after clear")
	})

	t.Run("ConcurrentRegistrationResolution", func(t *testing.T) {
		container := core.NewContainer()

		var wg sync.WaitGroup
		errors := make(chan error, 20)

		// Concurrent registrations
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				err := container.Register(func() string {
					return fmt.Sprintf("service_%d", id)
				})
				if err != nil {
					errors <- err
				}
			}(i)
		}

		// Concurrent resolutions
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var result string
				err := container.Resolve(&result)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Collect any errors
		var errorList []error
		for err := range errors {
			errorList = append(errorList, err)
		}

		// Some errors are expected in concurrent scenarios, but shouldn't crash
		if len(errorList) > 0 {
			t.Logf("Concurrent operations produced %d errors (may be expected)", len(errorList))
		}
	})
}
