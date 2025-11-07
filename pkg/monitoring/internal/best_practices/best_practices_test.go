package best_practices

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/logger"
	"github.com/lookatitude/beluga-ai/pkg/monitoring/internal/metrics"
	"github.com/stretchr/testify/assert"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestNewBestPracticesChecker(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("bp_test")
	mockMetrics := metrics.NewMetricsCollector()
	checker := NewBestPracticesChecker(mockLogger.(*logger.StructuredLogger), mockMetrics)

	assert.NotNil(t, checker)
	assert.NotNil(t, checker.logger)
	assert.NotNil(t, checker.metrics)
	assert.Len(t, checker.validators, 4) // concurrency, error_handling, resource_management, security
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestBestPracticesCheckerValidate(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("bp_test")
	mockMetrics := metrics.NewMetricsCollector()
	checker := NewBestPracticesChecker(mockLogger.(*logger.StructuredLogger), mockMetrics)
	ctx := context.Background()

	t.Run("validate with issues", func(t *testing.T) {
		// This should trigger some validation issues
		data := "go func() { /* uncontrolled goroutine */ }()"
		issues := checker.Validate(ctx, data, "test_component")

		assert.NotNil(t, issues)
		assert.IsType(t, []iface.ValidationIssue{}, issues)

		// Should record metrics
		metric, exists := mockMetrics.GetMetric("best_practices_checks_total", map[string]string{"component": "test_component"})
		assert.True(t, exists)
		assert.Equal(t, float64(1), metric.Value)
	})

	t.Run("validate without issues", func(t *testing.T) {
		data := "This is clean, well-structured code without obvious issues."
		issues := checker.Validate(ctx, data, "test_component")

		assert.NotNil(t, issues)
		// May still have some issues depending on the validation logic
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		assert.IsType(t, []iface.ValidationIssue{}, issues)
	})
}

func TestBestPracticesCheckerAddValidator(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("bp_test")
	mockMetrics := metrics.NewMetricsCollector()
	checker := NewBestPracticesChecker(mockLogger.(*logger.StructuredLogger), mockMetrics)

	initialCount := len(checker.validators)

	customValidator := &MockValidator{name: "custom"}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	checker.AddValidator(customValidator)

	assert.Len(t, checker.validators, initialCount+1)
	assert.Equal(t, customValidator, checker.validators[initialCount])
}

func TestConcurrencyValidator(t *testing.T) {
	validator := &ConcurrencyValidator{}

	t.Run("no concurrency issues", func(t *testing.T) {
		data := "This is normal code without goroutine issues."
		issues := validator.Validate(context.Background(), data)
		assert.Empty(t, issues)
	})

	t.Run("uncontrolled goroutine", func(t *testing.T) {
		data := `go func() {
			// This is an uncontrolled goroutine
			time.Sleep(time.Second)
		}()`
		issues := validator.Validate(context.Background(), data)
		if assert.NotEmpty(t, issues, "Expected at least one issue for uncontrolled goroutine") {
			assert.Equal(t, "concurrency", issues[0].Validator)
			assert.Contains(t, strings.ToLower(issues[0].Issue), "concurrency")
		}
	})

	t.Run("mutex in defer", func(t *testing.T) {
		data := `
		var mu sync.Mutex
		defer mu.Unlock() // This can cause deadlocks
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		`
		issues := validator.Validate(context.Background(), data)
		if assert.NotEmpty(t, issues, "Expected at least one issue for mutex in defer") {
			assert.Equal(t, "concurrency", issues[0].Validator)
		}
	})
}

func TestErrorHandlingValidator(t *testing.T) {
	validator := &ErrorHandlingValidator{}

	t.Run("no error handling issues", func(t *testing.T) {
		data := "This is normal code without error handling patterns."
		issues := validator.Validate(context.Background(), data)
		assert.Empty(t, issues)
	})

	t.Run("error not checked", func(t *testing.T) {
		data := `
		result, err := someFunction()
		// Error not checked!
		return result
		`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "error_handling", issues[0].Validator)
		assert.Contains(t, strings.ToLower(issues[0].Issue), "error")
	})

	t.Run("panic usage", func(t *testing.T) {
		data := `
		if somethingWrong {
			panic("This is bad in production")
		}
		`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "error_handling", issues[0].Validator)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	})

	t.Run("log.Fatal usage", func(t *testing.T) {
		data := `log.Fatal("This exits the program")`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "error_handling", issues[0].Validator)
	})
}

func TestResourceManagementValidator(t *testing.T) {
	validator := &ResourceManagementValidator{}

	t.Run("no resource issues", func(t *testing.T) {
		data := "This is normal code without resource management issues."
		issues := validator.Validate(context.Background(), data)
		assert.Empty(t, issues)
	})

	t.Run("file operations", func(t *testing.T) {
		data := `
		file, err := os.Open("test.txt")
		if err != nil {
			return err
		}
		// File not closed!
		`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "resource_management", issues[0].Validator)
		assert.Contains(t, strings.ToLower(issues[0].Issue), "resource")
	})

	t.Run("HTTP requests", func(t *testing.T) {
		data := `resp, err := http.Get("http://example.com")`
		issues := validator.Validate(context.Background(), data)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		assert.NotEmpty(t, issues)
		assert.Equal(t, "resource_management", issues[0].Validator)
	})

	t.Run("database operations", func(t *testing.T) {
		data := `db.Query("SELECT * FROM users")`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "resource_management", issues[0].Validator)
	})
}

func TestSecurityValidator(t *testing.T) {
	validator := &SecurityValidator{}

	t.Run("no security issues", func(t *testing.T) {
		data := "This is normal code without obvious security issues."
		issues := validator.Validate(context.Background(), data)
		assert.Empty(t, issues)
	})

	t.Run("password in string", func(t *testing.T) {
		data := `password := "secret123"`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "security", issues[0].Validator)
		assert.Contains(t, strings.ToLower(issues[0].Issue), "security")
	})

	t.Run("SQL injection risk", func(t *testing.T) {
		data := `query := "SELECT * FROM users WHERE id = " + userID`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "security", issues[0].Validator)
	})

	t.Run("code injection", func(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		data := `eval(userInput)`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "security", issues[0].Validator)
	})

	t.Run("command injection", func(t *testing.T) {
		data := `exec.Command("bash", "-c", userCommand)`
		issues := validator.Validate(context.Background(), data)
		assert.NotEmpty(t, issues)
		assert.Equal(t, "security", issues[0].Validator)
	})
}

func TestValidatorNames(t *testing.T) {
	validators := []iface.Validator{
		&ConcurrencyValidator{},
		&ErrorHandlingValidator{},
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		&ResourceManagementValidator{},
		&SecurityValidator{},
	}

	expectedNames := []string{
		"concurrency",
		"error_handling",
		"resource_management",
		"security",
	}

	for i, validator := range validators {
		assert.Equal(t, expectedNames[i], validator.Name())
	}
}

func TestPerformanceMonitor(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("perf_test")
	mockMetrics := metrics.NewMetricsCollector()
	monitor := NewPerformanceMonitor(mockLogger.(*logger.StructuredLogger), mockMetrics)

	t.Run("monitor operation", func(t *testing.T) {
		start := time.Now()
		err := monitor.MonitorOperation(context.Background(), "test_operation", func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, duration >= 10*time.Millisecond)

		// Check if metrics were recorded
		// The metric is recorded with labels, so we need to match them
		metric, exists := mockMetrics.GetMetric("operation_duration", map[string]string{
			"operation": "test_operation",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		})
		assert.True(t, exists)
		if exists {
			assert.True(t, metric.Value >= 0.01) // At least 10ms
		}
	})

	t.Run("monitor operation with error", func(t *testing.T) {
		testErr := assert.AnError
		err := monitor.MonitorOperation(context.Background(), "failing_operation", func() error {
			return testErr
		})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

		assert.Error(t, err)
		assert.Equal(t, testErr, err)
	})
}

func TestPerformanceMonitorGoroutines(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("perf_test")
	mockMetrics := metrics.NewMetricsCollector()
	monitor := NewPerformanceMonitor(mockLogger.(*logger.StructuredLogger), mockMetrics)

	// This is a simple test since runtime.NumGoroutine() can be unpredictable
	monitor.MonitorGoroutines(context.Background())

	// Check if metrics were recorded
	metric, exists := mockMetrics.GetMetric("goroutines_total", nil)
	assert.True(t, exists)
	assert.True(t, metric.Value >= 1) // At least the main goroutine
}

func TestDeadlockDetector(t *testing.T) {
	mockLogger := logger.NewStructuredLogger("deadlock_test")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	detector := NewDeadlockDetector(mockLogger.(*logger.StructuredLogger), time.Minute)

	assert.NotNil(t, detector)
	assert.NotNil(t, detector.lastActivity)
	assert.Equal(t, time.Minute, detector.checkInterval)

	t.Run("record activity", func(t *testing.T) {
		detector.RecordActivity("component1")
		assert.Contains(t, detector.lastActivity, "component1")
		assert.NotZero(t, detector.lastActivity["component1"])
	})

	t.Run("multiple components", func(t *testing.T) {
		detector.RecordActivity("component2")
		detector.RecordActivity("component3")

		assert.Contains(t, detector.lastActivity, "component2")
		assert.Contains(t, detector.lastActivity, "component3")
		assert.Len(t, detector.lastActivity, 3) // component1, component2, component3
	})
}

func TestContainsPattern(t *testing.T) {
	tests := []struct {
		text     string
		pattern  string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "goodbye", false},
		{"", "pattern", false},
		{"text", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.text+"_"+tt.pattern, func(t *testing.T) {
			result := containsPattern(tt.text, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
}

// Mock validator for testing
type MockValidator struct {
	name string
}

func (mv *MockValidator) Name() string {
	return mv.name
}

func (mv *MockValidator) Validate(ctx context.Context, data interface{}) []iface.ValidationIssue {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	return []iface.ValidationIssue{
		{
			Validator:  mv.name,
			Issue:      "Mock validation issue",
			Severity:   "low",
			Suggestion: "This is a mock suggestion",
		},
	}
}

// Benchmark tests
func BenchmarkBestPracticesChecker_Validate(b *testing.B) {
	mockLogger := logger.NewStructuredLogger("bench_test")
	mockMetrics := metrics.NewMetricsCollector()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	checker := NewBestPracticesChecker(mockLogger.(*logger.StructuredLogger), mockMetrics)
	ctx := context.Background()

	data := "This is sample code that might contain various patterns for validation."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		issues := checker.Validate(ctx, data, "bench_component")
		_ = issues
	}
}

func BenchmarkConcurrencyValidator_Validate(b *testing.B) {
	validator := &ConcurrencyValidator{}
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	data := `go func() {
		defer mu.Unlock() // Potential deadlock
		time.Sleep(time.Second)
	}()`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		issues := validator.Validate(ctx, data)
		_ = issues
	}
}

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func BenchmarkErrorHandlingValidator_Validate(b *testing.B) {
	validator := &ErrorHandlingValidator{}
	ctx := context.Background()

	data := `result, err := someFunction()
	if err != nil {
		panic("This is bad")
	}
	log.Fatal("Also bad")`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		issues := validator.Validate(ctx, data)
		_ = issues
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
}

func BenchmarkResourceManagementValidator_Validate(b *testing.B) {
	validator := &ResourceManagementValidator{}
	ctx := context.Background()

	data := `file, _ := os.Open("test.txt")
	resp, _ := http.Get("http://example.com")
	db.Query("SELECT * FROM users")`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		issues := validator.Validate(ctx, data)
		_ = issues
	}
}

func BenchmarkSecurityValidator_Validate(b *testing.B) {
	validator := &SecurityValidator{}
	ctx := context.Background()

	data := `password := "secret"
	query := "SELECT * FROM users WHERE id = " + userID
	eval(userInput)
	exec.Command("bash", userCommand)`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		issues := validator.Validate(ctx, data)
		_ = issues
	}
}

func BenchmarkPerformanceMonitor_MonitorOperation(b *testing.B) {
	mockLogger := logger.NewStructuredLogger("bench_test")
	mockMetrics := metrics.NewMetricsCollector()
	monitor := NewPerformanceMonitor(mockLogger.(*logger.StructuredLogger), mockMetrics)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := monitor.MonitorOperation(ctx, "bench_operation", func() error {
			time.Sleep(time.Microsecond)
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
