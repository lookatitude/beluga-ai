package orchestration

import (
	"fmt"
)

// ChainedTask represents a task with dynamic chaining capabilities.
type ChainedTask struct {
	ID       string
	Execute  func(input interface{}) (output interface{}, err error)
	Next     *ChainedTask
	Fallback *ChainedTask
}

// Run executes the task and dynamically chains to the next or fallback task.
func (ct *ChainedTask) Run(input interface{}) error {
	output, err := ct.Execute(input)
	if err != nil {
		fmt.Printf("Task %s failed: %v\n", ct.ID, err)
		if ct.Fallback != nil {
			fmt.Printf("Executing fallback for task %s\n", ct.ID)
			return ct.Fallback.Run(input)
		}
		return err
	}

	fmt.Printf("Task %s succeeded with output: %v\n", ct.ID, output)
	if ct.Next != nil {
		return ct.Next.Run(output)
	}

	return nil
}