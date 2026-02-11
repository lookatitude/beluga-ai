package o11y

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestTokenUsage(t *testing.T) {
	// TokenUsage should not panic even without explicit InitMeter.
	ctx := context.Background()
	TokenUsage(ctx, 100, 50)
}

func TestOperationDuration(t *testing.T) {
	ctx := context.Background()
	OperationDuration(ctx, 123.45)
}

func TestCost(t *testing.T) {
	ctx := context.Background()
	Cost(ctx, 0.0042)
}

func TestCounter(t *testing.T) {
	ctx := context.Background()
	Counter(ctx, "test.counter", 5)
}

func TestHistogram(t *testing.T) {
	ctx := context.Background()
	Histogram(ctx, "test.histogram", 99.9)
}

func TestInitMeter(t *testing.T) {
	err := InitMeter("test-meter-service")
	if err != nil {
		t.Fatalf("InitMeter: %v", err)
	}

	// After init, all instrument functions should work.
	ctx := context.Background()
	TokenUsage(ctx, 10, 20)
	OperationDuration(ctx, 50.0)
	Cost(ctx, 0.001)
	Counter(ctx, "post_init.counter", 1)
	Histogram(ctx, "post_init.histogram", 42.0)
}

func TestInitMeter_Reinit(t *testing.T) {
	// First init with service name A
	err := InitMeter("service-a")
	require.NoError(t, err)

	ctx := context.Background()
	TokenUsage(ctx, 1, 1)

	// Second init with service name B - should reset instruments
	err = InitMeter("service-b")
	require.NoError(t, err)

	// All metrics should still work after reinit
	TokenUsage(ctx, 2, 2)
	OperationDuration(ctx, 10.0)
	Cost(ctx, 0.05)
	Counter(ctx, "reinit.counter", 99)
	Histogram(ctx, "reinit.histogram", 88.0)
}

func TestInitInstruments(t *testing.T) {
	// Direct call to initInstruments should succeed with default meter
	err := initInstruments()
	assert.NoError(t, err, "initInstruments should not error with default meter")
}

func TestTokenUsage_WithInMemoryReader(t *testing.T) {
	// Set up in-memory reader to verify metrics are recorded
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter = provider.Meter("github.com/lookatitude/beluga-ai/o11y")

	// Reset instruments to use new meter
	meterOnce = sync.Once{}
	meterErr = nil

	ctx := context.Background()
	TokenUsage(ctx, 42, 24)

	// Read metrics
	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err)

	// Verify at least some metrics were recorded
	assert.NotEmpty(t, rm.ScopeMetrics, "expected metrics to be recorded")
}

func TestOperationDuration_WithInMemoryReader(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter = provider.Meter("github.com/lookatitude/beluga-ai/o11y")

	meterOnce = sync.Once{}
	meterErr = nil

	ctx := context.Background()
	OperationDuration(ctx, 150.5)

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err)
	assert.NotEmpty(t, rm.ScopeMetrics)
}

func TestCost_WithInMemoryReader(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter = provider.Meter("github.com/lookatitude/beluga-ai/o11y")

	meterOnce = sync.Once{}
	meterErr = nil

	ctx := context.Background()
	Cost(ctx, 0.00123)

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err)
	assert.NotEmpty(t, rm.ScopeMetrics)
}

func TestCounter_WithInMemoryReader(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter = provider.Meter("github.com/lookatitude/beluga-ai/o11y")

	ctx := context.Background()
	Counter(ctx, "custom.counter.test", 77)

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err)
	assert.NotEmpty(t, rm.ScopeMetrics)
}

func TestHistogram_WithInMemoryReader(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter = provider.Meter("github.com/lookatitude/beluga-ai/o11y")

	ctx := context.Background()
	Histogram(ctx, "custom.histogram.test", 3.14159)

	rm := metricdata.ResourceMetrics{}
	err := reader.Collect(ctx, &rm)
	require.NoError(t, err)
	assert.NotEmpty(t, rm.ScopeMetrics)
}

func TestMetrics_CalledBeforeInit(t *testing.T) {
	// Reset to default meter (simulating package init state)
	meter = noop.NewMeterProvider().Meter("test")
	meterOnce = sync.Once{}
	meterErr = nil

	ctx := context.Background()

	// All these should succeed without panicking even with noop meter
	TokenUsage(ctx, 1, 1)
	OperationDuration(ctx, 1.0)
	Cost(ctx, 0.01)
	Counter(ctx, "before.init", 1)
	Histogram(ctx, "before.init", 1.0)
}

func TestInitMeter_MultipleServices(t *testing.T) {
	// Test multiple reinits with different service names
	serviceNames := []string{"svc-1", "svc-2", "svc-3"}
	ctx := context.Background()

	for _, name := range serviceNames {
		err := InitMeter(name)
		require.NoError(t, err, "InitMeter failed for service %s", name)

		// Verify all metrics work after each reinit
		TokenUsage(ctx, 5, 5)
		OperationDuration(ctx, 25.0)
		Cost(ctx, 0.02)
		Counter(ctx, name+".counter", 1)
		Histogram(ctx, name+".histogram", 50.0)
	}
}

func TestInitInstruments_DirectCall(t *testing.T) {
	// Direct call to initInstruments should succeed
	err := initInstruments()
	assert.NoError(t, err, "initInstruments should not error with default meter")

	// Call it multiple times to verify idempotency via sync.Once
	err = initInstruments()
	assert.NoError(t, err)
	err = initInstruments()
	assert.NoError(t, err)
}

func TestMetricFunctions_AfterSuccessfulInit(t *testing.T) {
	// Ensure InitMeter works and all metrics succeed
	err := InitMeter("metrics-test-service")
	require.NoError(t, err)

	ctx := context.Background()

	// Call each metric function multiple times
	for i := 0; i < 3; i++ {
		TokenUsage(ctx, i*10, i*5)
		OperationDuration(ctx, float64(i)*12.5)
		Cost(ctx, float64(i)*0.001)
		Counter(ctx, "loop.counter", int64(i))
		Histogram(ctx, "loop.histogram", float64(i)*3.14)
	}
}

// mockMeter returns errors for instrument creation
type mockMeter struct {
	metric.Meter
	errOnCounter   bool
	errOnHistogram bool
}

func (m *mockMeter) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	if m.errOnCounter {
		return nil, errors.New("mock counter creation error")
	}
	return noop.NewMeterProvider().Meter("noop").Int64Counter(name, options...)
}

func (m *mockMeter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	if m.errOnHistogram {
		return nil, errors.New("mock histogram creation error")
	}
	return noop.NewMeterProvider().Meter("noop").Float64Histogram(name, options...)
}

func (m *mockMeter) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	if m.errOnCounter {
		return nil, errors.New("mock float counter creation error")
	}
	return noop.NewMeterProvider().Meter("noop").Float64Counter(name, options...)
}

func TestInitInstruments_ErrorPaths(t *testing.T) {
	// Test error on Int64Counter creation
	t.Run("error on input token counter", func(t *testing.T) {
		meter = &mockMeter{errOnCounter: true}
		meterOnce = sync.Once{}
		meterErr = nil

		err := initInstruments()
		assert.Error(t, err, "expected error from Int64Counter creation")
	})

	// Test error on Float64Histogram creation
	t.Run("error on histogram creation", func(t *testing.T) {
		// Reset and use mock that succeeds on counter but fails on histogram
		meter = &mockMeter{errOnHistogram: true}
		meterOnce = sync.Once{}
		meterErr = nil

		err := initInstruments()
		// This will succeed on counters but fail on histogram
		assert.Error(t, err, "expected error from Float64Histogram creation")
	})

	// Reset to working state for subsequent tests
	meter = noop.NewMeterProvider().Meter("test")
	meterOnce = sync.Once{}
	meterErr = nil
}

func TestMetricFunctions_WithInitError(t *testing.T) {
	// Set up meter that errors on instrument creation
	meter = &mockMeter{errOnCounter: true}
	meterOnce = sync.Once{}
	meterErr = nil

	ctx := context.Background()

	// These should not panic even when initInstruments fails
	TokenUsage(ctx, 1, 1)
	OperationDuration(ctx, 1.0)
	Cost(ctx, 0.01)

	// Reset for next test
	meter = noop.NewMeterProvider().Meter("test")
	meterOnce = sync.Once{}
	meterErr = nil
}

func TestCounter_WithMeterError(t *testing.T) {
	// Use mock meter that fails on Int64Counter
	meter = &mockMeter{errOnCounter: true}

	ctx := context.Background()

	// Should not panic when meter.Int64Counter returns error
	Counter(ctx, "failing.counter", 42)

	// Reset
	meter = noop.NewMeterProvider().Meter("test")
}

func TestHistogram_WithMeterError(t *testing.T) {
	// Use mock meter that fails on Float64Histogram
	meter = &mockMeter{errOnHistogram: true}

	ctx := context.Background()

	// Should not panic when meter.Float64Histogram returns error
	Histogram(ctx, "failing.histogram", 99.9)

	// Reset
	meter = noop.NewMeterProvider().Meter("test")
}

// errorMeter is a mock meter that returns errors for instrument creation.
type errorMeter struct {
	metric.Meter
	errorOnCounter   bool
	errorOnHistogram bool
}

func (m *errorMeter) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	if m.errorOnCounter {
		return nil, errors.New("mock counter creation error")
	}
	return noop.NewMeterProvider().Meter("test").Int64Counter(name, options...)
}

func (m *errorMeter) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	if m.errorOnCounter {
		return nil, errors.New("mock float counter creation error")
	}
	return noop.NewMeterProvider().Meter("test").Float64Counter(name, options...)
}

func (m *errorMeter) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	if m.errorOnHistogram {
		return nil, errors.New("mock histogram creation error")
	}
	return noop.NewMeterProvider().Meter("test").Float64Histogram(name, options...)
}

func TestInitInstruments_ErrorOnInputCounter(t *testing.T) {
	// Set meter to error meter that fails on first Int64Counter call
	originalMeter := meter
	originalMeterErr := meterErr
	defer func() {
		meter = originalMeter
		meterErr = originalMeterErr
		meterOnce = sync.Once{}
	}()

	meter = &errorMeter{errorOnCounter: true}
	meterOnce = sync.Once{}
	meterErr = nil

	err := initInstruments()
	assert.Error(t, err, "initInstruments should return error when input counter creation fails")
	assert.Contains(t, err.Error(), "counter creation error")
}

func TestInitInstruments_ErrorOnHistogram(t *testing.T) {
	// Set meter to error meter that fails on Float64Histogram call
	originalMeter := meter
	originalMeterErr := meterErr
	defer func() {
		meter = originalMeter
		meterErr = originalMeterErr
		meterOnce = sync.Once{}
	}()

	meter = &errorMeter{errorOnHistogram: true}
	meterOnce = sync.Once{}
	meterErr = nil

	err := initInstruments()
	assert.Error(t, err, "initInstruments should return error when histogram creation fails")
	assert.Contains(t, err.Error(), "histogram creation error")
}

func TestTokenUsage_WithInitError(t *testing.T) {
	// Set meterErr to simulate failed initialization
	originalMeter := meter
	originalMeterErr := meterErr
	defer func() {
		meter = originalMeter
		meterErr = originalMeterErr
		meterOnce = sync.Once{}
	}()

	meter = &errorMeter{errorOnCounter: true}
	meterOnce = sync.Once{}
	meterErr = nil

	ctx := context.Background()
	// Should not panic even with init error
	TokenUsage(ctx, 10, 20)
}

func TestOperationDuration_WithInitError(t *testing.T) {
	originalMeter := meter
	originalMeterErr := meterErr
	defer func() {
		meter = originalMeter
		meterErr = originalMeterErr
		meterOnce = sync.Once{}
	}()

	meter = &errorMeter{errorOnHistogram: true}
	meterOnce = sync.Once{}
	meterErr = nil

	ctx := context.Background()
	// Should not panic even with init error
	OperationDuration(ctx, 50.0)
}

func TestCost_WithInitError(t *testing.T) {
	originalMeter := meter
	originalMeterErr := meterErr
	defer func() {
		meter = originalMeter
		meterErr = originalMeterErr
		meterOnce = sync.Once{}
	}()

	meter = &errorMeter{errorOnCounter: true}
	meterOnce = sync.Once{}
	meterErr = nil

	ctx := context.Background()
	// Should not panic even with init error
	Cost(ctx, 0.01)
}

func TestCounter_WithCreationError(t *testing.T) {
	originalMeter := meter
	defer func() {
		meter = originalMeter
	}()

	meter = &errorMeter{errorOnCounter: true}
	ctx := context.Background()

	// Should not panic even when counter creation fails
	Counter(ctx, "test.counter", 10)
}

func TestHistogram_WithCreationError(t *testing.T) {
	originalMeter := meter
	defer func() {
		meter = originalMeter
	}()

	meter = &errorMeter{errorOnHistogram: true}
	ctx := context.Background()

	// Should not panic even when histogram creation fails
	Histogram(ctx, "test.histogram", 42.0)
}

func TestInitInstruments_ErrorOnOutputCounter(t *testing.T) {
	// Use a meter that succeeds for first Int64Counter but fails on second
	originalMeter := meter
	originalMeterErr := meterErr
	defer func() {
		meter = originalMeter
		meterErr = originalMeterErr
		meterOnce = sync.Once{}
	}()

	// Create a meter that succeeds on first call, fails on second
	callCount := 0
	meter = &mockMeterWithCallCount{
		callCountPtr: &callCount,
	}
	meterOnce = sync.Once{}
	meterErr = nil

	err := initInstruments()
	assert.Error(t, err, "initInstruments should return error when output counter creation fails")
}

func TestInitInstruments_ErrorOnCostGauge(t *testing.T) {
	// Create a custom meter that fails only on Float64Counter (cost gauge)
	originalMeter := meter
	originalMeterErr := meterErr
	defer func() {
		meter = originalMeter
		meterErr = originalMeterErr
		meterOnce = sync.Once{}
	}()

	// Create meter that succeeds for Int64Counter and Float64Histogram but fails on Float64Counter
	meter = &mockMeterForCostError{}
	meterOnce = sync.Once{}
	meterErr = nil

	err := initInstruments()
	assert.Error(t, err, "initInstruments should return error when cost gauge creation fails")
}

// mockMeterWithCallCount tracks Int64Counter calls and fails on second call
type mockMeterWithCallCount struct {
	metric.Meter
	callCountPtr *int
}

func (m *mockMeterWithCallCount) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	*m.callCountPtr++
	if *m.callCountPtr == 2 {
		return nil, errors.New("mock output counter creation error")
	}
	return noop.NewMeterProvider().Meter("test").Int64Counter(name, options...)
}

func (m *mockMeterWithCallCount) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return noop.NewMeterProvider().Meter("test").Float64Histogram(name, options...)
}

func (m *mockMeterWithCallCount) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	return noop.NewMeterProvider().Meter("test").Float64Counter(name, options...)
}

// mockMeterForCostError succeeds on Int64Counter and Float64Histogram but fails on Float64Counter
type mockMeterForCostError struct {
	metric.Meter
}

func (m *mockMeterForCostError) Int64Counter(name string, options ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return noop.NewMeterProvider().Meter("test").Int64Counter(name, options...)
}

func (m *mockMeterForCostError) Float64Histogram(name string, options ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return noop.NewMeterProvider().Meter("test").Float64Histogram(name, options...)
}

func (m *mockMeterForCostError) Float64Counter(name string, options ...metric.Float64CounterOption) (metric.Float64Counter, error) {
	return nil, errors.New("mock cost gauge creation error")
}
