package fixtures

import (
	"net/http"
	"testing"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestActualImplementation(t *testing.T) {
	// Unit test using actual HTTP client instead of mock
	client := http.Client{}
	_, err := client.Get("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
}

