package httpclient

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// redacted is the placeholder that replaces credential values.
const redacted = "REDACTED"

// sensitiveParamExact lists query-parameter names (lower-cased) whose values
// are always treated as credentials.
var sensitiveParamExact = map[string]bool{
	"key":       true,
	"auth":      true,
	"sig":       true,
	"code":      true,
	"pass":      true,
	"pwd":       true,
	"session":   true,
	"sessionid": true,
}

// sensitiveParamSubstrings marks a query-parameter name as a credential when
// its lower-cased form contains any of these fragments (api_key, apikey,
// x-goog-api-key, access_token, client_secret, signature, password, ...).
var sensitiveParamSubstrings = []string{
	"apikey", "api_key", "api-key", "token", "secret", "password", "passwd",
	"signature", "credential", "authorization",
}

var (
	// queryParamRe matches one query parameter: a separator, a name and a value.
	queryParamRe = regexp.MustCompile(`([?&;])([^=&;#\s"'?]+)=([^&;#\s"']*)`)
	// userinfoRe matches the password component of URL userinfo.
	userinfoRe = regexp.MustCompile(`(://[^/\s:@"']+):([^@/\s"']+)@`)
)

// isSensitiveParam reports whether a query-parameter name carries a credential.
func isSensitiveParam(name string) bool {
	n := strings.ToLower(name)
	if sensitiveParamExact[n] {
		return true
	}
	for _, s := range sensitiveParamSubstrings {
		if strings.Contains(n, s) {
			return true
		}
	}
	return false
}

// RedactURL replaces credential-bearing query-parameter values (key=, api_key=,
// token=, access_token=, signature=, ...) and any userinfo password in s with
// "REDACTED". It works on a bare URL and on free text that embeds URLs, such
// as an error message, and preserves parameter order.
func RedactURL(s string) string {
	if s == "" {
		return s
	}
	out := queryParamRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := queryParamRe.FindStringSubmatch(m)
		if len(sub) != 4 || !isSensitiveParam(sub[2]) || sub[3] == "" {
			return m
		}
		return sub[1] + sub[2] + "=" + redacted
	})
	return userinfoRe.ReplaceAllString(out, "${1}:"+redacted+"@")
}

// redactedError carries a redacted message while keeping the original chain
// reachable for errors.Is / errors.As.
type redactedError struct {
	msg string
	err error
}

func (e *redactedError) Error() string { return e.msg }
func (e *redactedError) Unwrap() error { return e.err }

// RedactError scrubs credentials out of err. Every *url.Error in the chain has
// its URL field redacted in place (so callers that errors.As into *url.Error
// also see the redacted URL), and if the rendered message still contains a
// credential the error is wrapped so that Error() returns the redacted text.
// errors.Is / errors.As traversal is preserved. A nil err returns nil.
func RedactError(err error) error {
	if err == nil {
		return nil
	}
	redactURLErrors(err)
	msg := err.Error()
	if clean := RedactURL(msg); clean != msg {
		return &redactedError{msg: clean, err: err}
	}
	return err
}

// redactURLErrors walks the error tree and redacts the URL of every *url.Error.
func redactURLErrors(err error) {
	for err != nil {
		if ue, ok := err.(*url.Error); ok { //nolint:errorlint // deliberate per-node type check while walking the chain
			ue.URL = RedactURL(ue.URL)
		}
		switch x := err.(type) { //nolint:errorlint // walking the chain manually
		case interface{ Unwrap() []error }:
			for _, e := range x.Unwrap() {
				redactURLErrors(e)
			}
			return
		default:
			err = errors.Unwrap(err)
		}
	}
}
