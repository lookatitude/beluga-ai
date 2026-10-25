package errclass_test

import (
	"errors"
	"fmt"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/errclass"
)

func ExampleClassify() {
	err := errclass.Classify("myprovider: generate failed", 429, errors.New("slow down"))
	fmt.Println(core.IsRetryable(err))
	// Output: true
}
