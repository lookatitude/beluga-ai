package core

import (
	"context"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestNewContainer(t *testing.T) {
	container := NewContainer()

	if container == nil {
		t.Fatal("NewContainer() returned nil")
	}

	// Test basic functionality
	if err := container.Register(func() string { return "test" }); err != nil {
		t.Errorf("Register() error = %v", err)
	}

	var result string
	if err := container.Resolve(&result); err != nil {
		t.Errorf("Resolve() error = %v", err)
	}

	if result != "test" {
		t.Errorf("Resolve() = %q, expected %q", result, "test")
	}
}

func TestNewContainerWithOptions(t *testing.T) {
	logger := &testLogger{}
	tracerProvider := trace.NewNoopTracerProvider()

	container := NewContainerWithOptions(
		WithLogger(logger),
		WithTracerProvider(tracerProvider),
	)

	if container == nil {
		t.Fatal("NewContainerWithOptions() returned nil")
	}

	// Verify the container has the configured components
	if impl, ok := container.(*containerImpl); ok {
		if impl.logger != logger {
			t.Error("Logger not set correctly")
		}
		if impl.tracerProvider != tracerProvider {
			t.Error("TracerProvider not set correctly")
		}
	}
}

func TestContainer_Register(t *testing.T) {
	tests := []struct {
		name        string
		factoryFunc interface{}
		wantErr     bool
	}{
		{
			name:        "valid factory function",
			factoryFunc: func() string { return "test" },
			wantErr:     false,
		},
		{
			name:        "factory with dependency",
			factoryFunc: func(s string) int { return len(s) },
			wantErr:     false,
		},
		{
			name:        "invalid factory - not a function",
			factoryFunc: "not a function",
			wantErr:     true,
		},
		{
			name:        "invalid factory - returns nothing",
			factoryFunc: func() {},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewContainer()

			err := container.Register(tt.factoryFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainer_Resolve(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(Container)
		target  interface{}
		wantErr bool
	}{
		{
			name: "resolve simple type",
			setup: func(c Container) {
				c.Register(func() string { return "test" })
			},
			target:  func() *string { var s string; return &s }(),
			wantErr: false,
		},
		{
			name: "resolve with dependency",
			setup: func(c Container) {
				c.Register(func() string { return "test" })
				c.Register(func(s string) int { return len(s) })
			},
			target:  func() *int { var i int; return &i }(),
			wantErr: false,
		},
		{
			name: "resolve unregistered type",
			setup: func(c Container) {
				// No registration
			},
			target:  func() *string { var s string; return &s }(),
			wantErr: true,
		},
		{
			name: "resolve non-pointer",
			setup: func(c Container) {
				c.Register(func() string { return "test" })
			},
			target:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewContainer()
			tt.setup(container)

			err := container.Resolve(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainer_Singleton(t *testing.T) {
	container := NewContainer()

	instance := "singleton_test"
	container.Singleton(instance)

	var result string
	if err := container.Resolve(&result); err != nil {
		t.Errorf("Resolve() error = %v", err)
	}

	if result != instance {
		t.Errorf("Resolve() = %q, expected %q", result, instance)
	}
}

func TestContainer_Has(t *testing.T) {
	container := NewContainer()

	// Test with registered factory
	container.Register(func() string { return "test" })
	if !container.Has(stringType()) {
		t.Error("Has() should return true for registered type")
	}

	// Test with singleton
	container.Singleton(42)
	if !container.Has(intType()) {
		t.Error("Has() should return true for singleton type")
	}

	// Test with unregistered type
	if container.Has(boolType()) {
		t.Error("Has() should return false for unregistered type")
	}
}

func TestContainer_Clear(t *testing.T) {
	container := NewContainer()

	container.Register(func() string { return "test" })
	container.Singleton(42)

	if !container.Has(stringType()) || !container.Has(intType()) {
		t.Error("Setup failed: types should be registered")
	}

	container.Clear()

	if container.Has(stringType()) || container.Has(intType()) {
		t.Error("Clear() should remove all registrations")
	}
}

func TestContainer_CheckHealth(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(Container)
		wantErr bool
	}{
		{
			name:    "healthy container",
			setup:   func(c Container) {},
			wantErr: false,
		},
		{
			name: "container with existing registrations",
			setup: func(c Container) {
				c.Register(func() string { return "existing" })
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := NewContainer()
			tt.setup(container)

			err := container.CheckHealth(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckHealth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuilder_WithLogger(t *testing.T) {
	logger := &testLogger{}
	builder := NewBuilder(NewContainer()).WithLogger(logger)

	if impl, ok := builder.container.(*containerImpl); ok {
		if impl.logger != logger {
			t.Error("WithLogger() did not set logger correctly")
		}
	}
}

func TestBuilder_WithTracerProvider(t *testing.T) {
	tracerProvider := trace.NewNoopTracerProvider()
	builder := NewBuilder(NewContainer()).WithTracerProvider(tracerProvider)

	if impl, ok := builder.container.(*containerImpl); ok {
		if impl.tracerProvider != tracerProvider {
			t.Error("WithTracerProvider() did not set tracer provider correctly")
		}
	}
}

// Helper functions for type reflection
func stringType() reflect.Type { return reflect.TypeOf("") }
func intType() reflect.Type    { return reflect.TypeOf(0) }
func boolType() reflect.Type   { return reflect.TypeOf(false) }

// testLogger is a simple logger implementation for testing
type testLogger struct {
	logs []string
}

func (t *testLogger) Debug(msg string, args ...interface{}) {
	t.logs = append(t.logs, "DEBUG: "+msg)
}

func (t *testLogger) Info(msg string, args ...interface{}) {
	t.logs = append(t.logs, "INFO: "+msg)
}

func (t *testLogger) Warn(msg string, args ...interface{}) {
	t.logs = append(t.logs, "WARN: "+msg)
}

func (t *testLogger) Error(msg string, args ...interface{}) {
	t.logs = append(t.logs, "ERROR: "+msg)
}

func (t *testLogger) With(args ...interface{}) Logger {
	return t
}
