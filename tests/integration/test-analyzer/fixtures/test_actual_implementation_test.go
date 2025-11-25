package fixtures

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestActualImplementation(t *testing.T) {
	// Unit test using actual HTTP client instead of mock
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
}
