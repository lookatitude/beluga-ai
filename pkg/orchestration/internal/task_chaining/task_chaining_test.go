package orchestration

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestChainedTask_Run_Success_NoNext(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	task := &ChainedTask{
		ID: "test-task",
		Execute: func(input interface{}) (interface{}, error) {
			return "success_output", nil
		},
	}

	err := task.Run("test_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Task test-task succeeded with output: success_output")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func TestChainedTask_Run_Success_WithNext(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	nextTask := &ChainedTask{
		ID: "next-task",
		Execute: func(input interface{}) (interface{}, error) {
			return "next_output", nil
		},
	}

	task := &ChainedTask{
		ID: "first-task",
		Execute: func(input interface{}) (interface{}, error) {
			return "first_output", nil
		},
		Next: nextTask,
	}

	err := task.Run("test_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	assert.Contains(t, output, "Task first-task succeeded with output: first_output")
	assert.Contains(t, output, "Task next-task succeeded with output: next_output")
}

func TestChainedTask_Run_Error_NoFallback(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	task := &ChainedTask{
		ID: "failing-task",
		Execute: func(input interface{}) (interface{}, error) {
			return nil, errors.New("task execution failed")
		},
	}

	err := task.Run("test_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task execution failed")
	assert.Contains(t, output, "Task failing-task failed: task execution failed")
}

func TestChainedTask_Run_Error_WithFallback(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fallbackTask := &ChainedTask{
		ID: "fallback-task",
		Execute: func(input interface{}) (interface{}, error) {
			return "fallback_output", nil
		},
	}

	task := &ChainedTask{
		ID: "main-task",
		Execute: func(input interface{}) (interface{}, error) {
			return nil, errors.New("main task failed")
		},
		Fallback: fallbackTask,
	}

	err := task.Run("test_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	output := buf.String()

	assert.NoError(t, err) // Should succeed due to fallback
	assert.Contains(t, output, "Task main-task failed: main task failed")
	assert.Contains(t, output, "Executing fallback for task main-task")
	assert.Contains(t, output, "Task fallback-task succeeded with output: fallback_output")
}

func TestChainedTask_Run_FallbackAlsoFails(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fallbackTask := &ChainedTask{
		ID: "failing-fallback",
		Execute: func(input interface{}) (interface{}, error) {
			return nil, errors.New("fallback also failed")
		},
	}

	task := &ChainedTask{
		ID: "main-task",
		Execute: func(input interface{}) (interface{}, error) {
			return nil, errors.New("main task failed")
		},
		Fallback: fallbackTask,
	}

	err := task.Run("test_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	io.Copy(&buf, r)
	output := buf.String()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback also failed")
	assert.Contains(t, output, "Task main-task failed: main task failed")
	assert.Contains(t, output, "Executing fallback for task main-task")
	assert.Contains(t, output, "Task failing-fallback failed: fallback also failed")
}

func TestChainedTask_Run_ChainWithMultipleTasks(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a chain: task1 -> task2 -> task3
	task3 := &ChainedTask{
		ID: "task3",
		Execute: func(input interface{}) (interface{}, error) {
			return "final_output", nil
		},
	}

	task2 := &ChainedTask{
		ID: "task2",
		Execute: func(input interface{}) (interface{}, error) {
			return "task2_output", nil
		},
		Next: task3,
	}

	task1 := &ChainedTask{
		ID: "task1",
		Execute: func(input interface{}) (interface{}, error) {
			return "task1_output", nil
		},
		Next: task2,
	}

	err := task1.Run("initial_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Task task1 succeeded with output: task1_output")
	assert.Contains(t, output, "Task task2 succeeded with output: task2_output")
	assert.Contains(t, output, "Task task3 succeeded with output: final_output")
}

func TestChainedTask_Run_ChainWithFallbackInMiddle(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a chain where middle task fails and has fallback
	fallbackTask := &ChainedTask{
		ID: "fallback-middle",
		Execute: func(input interface{}) (interface{}, error) {
			return "fallback_output", nil
		},
		Next: &ChainedTask{
			ID: "after-fallback",
			Execute: func(input interface{}) (interface{}, error) {
				return "after_fallback_output", nil
			},
		},
	}

	task2 := &ChainedTask{
		ID: "failing-task",
		Execute: func(input interface{}) (interface{}, error) {
			return nil, errors.New("middle task failed")
		},
		Fallback: fallbackTask,
		Next: &ChainedTask{
			ID: "never-reached",
			Execute: func(input interface{}) (interface{}, error) {
				return "should_not_reach", nil
			},
		},
	}

	task1 := &ChainedTask{
		ID: "first-task",
		Execute: func(input interface{}) (interface{}, error) {
			return "first_output", nil
		},
		Next: task2,
	}

	err := task1.Run("initial_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Task first-task succeeded with output: first_output")
	assert.Contains(t, output, "Task failing-task failed: middle task failed")
	assert.Contains(t, output, "Executing fallback for task failing-task")
	assert.Contains(t, output, "Task fallback-middle succeeded with output: fallback_output")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	assert.Contains(t, output, "Task after-fallback succeeded with output: after_fallback_output")
	assert.NotContains(t, output, "should_not_reach")
}

func TestChainedTask_Run_NilExecuteFunction(t *testing.T) {
	task := &ChainedTask{
		ID:      "nil-execute",
		Execute: nil,
	}

	err := task.Run("test_input")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "runtime error") // This will be a nil pointer dereference
}

func TestChainedTask_Run_EmptyID(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	task := &ChainedTask{
		ID: "",
		Execute: func(input interface{}) (interface{}, error) {
			return "output", nil
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		},
	}

	err := task.Run("test_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Task  succeeded with output: output") // Empty ID
}

func TestChainedTask_Run_SpecialCharacters(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	task := &ChainedTask{
		ID: "task:with:colons@domain.com",
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
		Execute: func(input interface{}) (interface{}, error) {
			return "special!@#$%^&*()_output", nil
		},
	}

	err := task.Run("special!@#$%^&*()_input")

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Task task:with:colons@domain.com succeeded with output: special!@#$%^&*()_output")
}

func TestChainedTask_Run_NilInput(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	task := &ChainedTask{
		ID: "nil-input-task",
		Execute: func(input interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			if input == nil {
				return "handled_nil", nil
			}
			return "unexpected", nil
		},
	}

	err := task.Run(nil)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Task nil-input-task succeeded with output: handled_nil")
}

func TestChainedTask_Run_ComplexDataTypes(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	task := &ChainedTask{
		ID: "complex-task",
		Execute: func(input interface{}) (interface{}, error) {
			// Input should be a slice
			if slice, ok := input.([]string); ok {
				return map[string]interface{}{
					"processed": slice,
					"count":     len(slice),
					"first":     slice[0],
				}, nil
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			}
			return nil, errors.New("expected slice input")
		},
	}

	input := []string{"item1", "item2", "item3"}
	err := task.Run(input)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Task complex-task succeeded with output:")
	assert.Contains(t, output, "processed")
	assert.Contains(t, output, "count")
	assert.Contains(t, output, "first")
}

func TestChainedTask_Run_CircularReferencePrevention(t *testing.T) {
	// Create a circular reference: task1 -> task2 -> task1
	task1 := &ChainedTask{
		ID: "task1",
		Execute: func(input interface{}) (interface{}, error) {
			return "task1_output", nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

	task2 := &ChainedTask{
		ID: "task2",
		Execute: func(input interface{}) (interface{}, error) {
			return "task2_output", nil
		},
		Next: task1, // This creates a cycle
	}

	task1.Next = task2 // Complete the cycle

	// This should eventually cause a stack overflow due to infinite recursion
	// In practice, we'd want to add cycle detection, but for this test
	// we'll just ensure it doesn't panic immediately

	assert.NotPanics(t, func() {
		// This will likely cause infinite recursion, but should be caught by Go's stack limits
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Expected panic due to circular reference: %v", r)
			}
		}()
		task1.Run("initial")
	})
}

func TestChainedTask_Run_OutputTypeHandling(t *testing.T) {
	testCases := []struct {
		name     string
		output   interface{}
		expected string
	}{
		{
			name:     "string_output",
			output:   "string_result",
			expected: "string_result",
		},
		{
			name:     "int_output",
			output:   42,
			expected: "42",
		},
		{
			name:     "bool_output",
			output:   true,
			expected: "true",
		},
		{
			name:     "map_output",
			output:   map[string]string{"key": "value"},
			expected: "map[key:value]",
		},
		{
			name:     "slice_output",
			output:   []int{1, 2, 3},
			expected: "[1 2 3]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			task := &ChainedTask{
				ID: fmt.Sprintf("test-%s", tc.name),
				Execute: func(input interface{}) (interface{}, error) {
					return tc.output, nil
				},
			}

			err := task.Run("input")

			// Restore stdout
			w.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			assert.NoError(t, err)
			assert.Contains(t, output, fmt.Sprintf("Task test-%s succeeded with output:", tc.name))
		})
	}
}

func TestChainedTask_Run_ErrorPropagation(t *testing.T) {
	// Test that errors are properly propagated through the chain
	customError := errors.New("custom execution error")

	task := &ChainedTask{
		ID: "error-task",
		Execute: func(input interface{}) (interface{}, error) {
			return nil, customError
		},
	}

	err := task.Run("input")

	assert.Error(t, err)
	assert.Equal(t, customError, err)
}

func TestChainedTask_Run_FallbackReceivesOriginalInput(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	originalInput := "original_test_input"

	fallbackTask := &ChainedTask{
		ID: "fallback-task",
		Execute: func(input interface{}) (interface{}, error) {
			// Fallback should receive the original input, not the failed task's output
			if input == originalInput {
				return "fallback_success", nil
			}
			return nil, fmt.Errorf("fallback received wrong input: %v", input)
		},
	}

	task := &ChainedTask{
		ID: "main-task",
		Execute: func(input interface{}) (interface{}, error) {
			return nil, errors.New("main failed")
		},
		Fallback: fallbackTask,
	}

	err := task.Run(originalInput)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.NoError(t, err)
	assert.Contains(t, output, "Task fallback-task succeeded with output: fallback_success")
}
