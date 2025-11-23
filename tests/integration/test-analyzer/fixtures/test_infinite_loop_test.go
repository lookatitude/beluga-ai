package fixtures

import "testing"

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestInfiniteLoop(t *testing.T) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			return
		default:
			// Loop body
		}
		// This is an infinite loop without exit condition
	}
}
