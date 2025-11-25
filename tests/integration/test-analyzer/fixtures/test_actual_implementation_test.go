package fixtures

import (
	"net/http"
	"testing"
)

func TestActualImplementation(t *testing.T) {
	// Unit test using actual HTTP client instead of mock
	client := http.Client{}
	_, err := client.Get("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
}

