package computeruse

import (
	"context"
)

// BrowserBackend is the interface for browser automation. Implementations
// wrap headless browsers or browser automation frameworks.
type BrowserBackend interface {
	// Navigate loads the given URL in the browser.
	Navigate(ctx context.Context, url string) error

	// Screenshot captures the current page as PNG bytes.
	Screenshot(ctx context.Context) ([]byte, error)

	// Click performs a mouse click at the given coordinates.
	Click(ctx context.Context, x, y int) error

	// Type enters text into the currently focused element.
	Type(ctx context.Context, text string) error
}

// ExtendedBrowser is an optional interface for browsers that support
// JavaScript execution and scrolling. Check via type assertion.
type ExtendedBrowser interface {
	BrowserBackend

	// Scroll scrolls the page by the given delta (positive = down).
	Scroll(ctx context.Context, x, y, delta int) error

	// ExecuteJS runs JavaScript in the browser and returns the result.
	ExecuteJS(ctx context.Context, script string) (string, error)
}
