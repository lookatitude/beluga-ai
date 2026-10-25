// Package errclass maps upstream provider failures onto typed core.Error codes
// so that retry, fallback and failover logic (core.IsRetryable) behaves
// correctly for every provider.
//
// The mapping is:
//
//	401, 403        -> core.ErrAuth          (not retryable)
//	402             -> core.ErrBudgetExhausted (not retryable)
//	404             -> core.ErrNotFound      (not retryable)
//	408             -> core.ErrTimeout       (retryable)
//	429             -> core.ErrRateLimit     (retryable)
//	other 4xx       -> core.ErrInvalidInput  (not retryable)
//	5xx             -> core.ErrProviderDown  (retryable)
//	deadline/timeout-> core.ErrTimeout       (retryable)
//	network failure -> core.ErrProviderDown  (retryable)
//	context.Canceled-> left uncoded          (not retryable)
//
// An error that already carries a *core.Error keeps its code: it is never
// re-wrapped under a different one. Every message is passed through
// httpclient.RedactError so URLs with credentials never leak.
package errclass

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/httpclient"
)

// CodeForStatus returns the core.ErrorCode for an HTTP status code. Status
// codes below 400 (which are not failures) map to core.ErrProviderDown, since
// a provider returning them as an error is behaving unexpectedly.
func CodeForStatus(status int) core.ErrorCode {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return core.ErrAuth
	case status == http.StatusPaymentRequired:
		return core.ErrBudgetExhausted
	case status == http.StatusNotFound:
		return core.ErrNotFound
	case status == http.StatusRequestTimeout:
		return core.ErrTimeout
	case status == http.StatusTooManyRequests:
		return core.ErrRateLimit
	case status >= 400 && status < 500:
		return core.ErrInvalidInput
	default:
		return core.ErrProviderDown
	}
}

// statusCoder is implemented by errors that expose an HTTP status code, e.g.
// smithy-go's *smithyhttp.ResponseError (AWS SDK v2).
type statusCoder interface {
	HTTPStatusCode() int
}

// StatusCode extracts an HTTP status code from err's chain for the error
// types known without an SDK dependency: *httpclient.APIError and anything
// implementing HTTPStatusCode() int. It returns 0 when none is found.
func StatusCode(err error) int {
	var apiErr *httpclient.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	var sc statusCoder
	if errors.As(err, &sc) {
		return sc.HTTPStatusCode()
	}
	return 0
}

// Classify wraps err with prefix (e.g. "openai: generate failed") and a
// core.ErrorCode chosen from status and the error itself. A status <= 0 means
// "unknown"; in that case StatusCode(err) is consulted. Classify returns nil
// for a nil err.
func Classify(prefix string, status int, err error) error {
	if err == nil {
		return nil
	}
	err = httpclient.RedactError(err)

	// Never re-code an error that is already typed.
	var ce *core.Error
	if errors.As(err, &ce) {
		return core.Errorf(ce.Code, "%s: %w", prefix, err)
	}

	// Cancellation is the caller's decision, not a provider fault: keep the
	// sentinel reachable and leave it non-retryable.
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w", prefix, err)
	}

	if status <= 0 {
		status = StatusCode(err)
	}
	if status > 0 {
		return core.Errorf(CodeForStatus(status), "%s: %w", prefix, err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return core.Errorf(core.ErrTimeout, "%s: %w", prefix, err)
	}
	if httpclient.IsNetworkError(err) {
		return core.Errorf(core.ErrProviderDown, "%s: %w", prefix, err)
	}

	// Unknown, status-less failure (e.g. a malformed response): report it as
	// a provider fault but do not guess that a retry will help.
	return fmt.Errorf("%s: %w", prefix, err)
}

// Wrap is core.Errorf that preserves an existing code: if any argument that
// will be wrapped with %w already carries a *core.Error, its code wins over
// code. Use it when adding context to an error that may already be typed.
func Wrap(code core.ErrorCode, format string, args ...any) *core.Error {
	e := core.Errorf(code, format, args...)
	var inner *core.Error
	if e.Err != nil && errors.As(e.Err, &inner) {
		e.Code = inner.Code
	}
	return e
}
