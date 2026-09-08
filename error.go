package goldsky

import (
	"errors"
	"fmt"
	"net/http"
)

// ProblemDetails is an RFC 9457 (https://www.rfc-editor.org/rfc/rfc9457)
// application/problem+json failure returned by the Goldsky REST control plane.
//
// Callers must branch on Type (a stable URI) rather than Title or Detail, which
// are human-readable prose and may change. Every Type dereferences to a page in
// the Goldsky error catalogue at https://api.goldsky.com/api/errors.
type ProblemDetails struct {
	// Type is the stable problem identifier URI, e.g.
	// "https://api.goldsky.com/api/errors/subgraph-not-found". When the server
	// omits it, this is "about:blank" per RFC 9457.
	Type string `json:"type,omitempty"`
	// Title is a short, human-readable summary. Not stable; do not branch on it.
	Title string `json:"title,omitempty"`
	// Status is the HTTP status code copied into the problem body.
	Status int `json:"status,omitempty"`
	// Detail is a human-readable explanation specific to this occurrence.
	Detail string `json:"detail,omitempty"`
	// Instance is the URI identifying the specific occurrence, often the path.
	Instance string `json:"instance,omitempty"`
	// Errors is the validation field-error list present on 400 responses.
	Errors []ValidationError `json:"errors,omitempty"`

	// Headers are the response headers, useful for Retry-After and rate-limit
	// metadata. They never contain the request Authorization value.
	Headers http.Header `json:"-"`
	// RawBody is the raw response body retained for diagnostics. Problem
	// bodies do not carry secrets, but callers should still avoid logging it
	// verbatim in shared systems.
	RawBody []byte `json:"-"`
}

// ValidationError names a single offending field in a 400 validation response.
type ValidationError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// Error implements the error interface. The message never includes request
// credentials; problem bodies are server-authored and do not echo secrets.
func (e *ProblemDetails) Error() string {
	if e == nil {
		return "goldsky: <nil problem>"
	}
	status := e.Status
	if status == 0 {
		status = http.StatusOK
	}
	if e.Detail != "" {
		return fmt.Sprintf("goldsky API error %d (%s): %s", status, e.Type, e.Detail)
	}
	if e.Title != "" {
		return fmt.Sprintf("goldsky API error %d (%s): %s", status, e.Type, e.Title)
	}
	return fmt.Sprintf("goldsky API error %d (%s)", status, e.Type)
}

// Is supports errors.Is comparisons against sentinel problem types.
func (e *ProblemDetails) Is(target error) bool {
	t, ok := target.(*ProblemDetails)
	if !ok {
		return false
	}
	if t.Type != "" && e.Type != t.Type {
		return false
	}
	if t.Status != 0 && e.Status != t.Status {
		return false
	}
	return true
}

// AsProblem returns the *ProblemDetails if err is (or wraps) one, else nil.
func AsProblem(err error) *ProblemDetails {
	var p *ProblemDetails
	if errors.As(err, &p) {
		return p
	}
	return nil
}

// Classification predicates. These inspect Status because the Type URI catalogue
// is large; callers that need exact problem identity should compare Type.

// IsValidation reports a 400 validation failure.
func (e *ProblemDetails) IsValidation() bool { return e != nil && e.Status == http.StatusBadRequest }

// IsAuthentication reports a 401 authentication failure.
func (e *ProblemDetails) IsAuthentication() bool {
	return e != nil && e.Status == http.StatusUnauthorized
}

// IsSubscription reports a 402 subscription/billing failure.
func (e *ProblemDetails) IsSubscription() bool {
	return e != nil && e.Status == http.StatusPaymentRequired
}

// IsPermission reports a 403 permission or limit failure.
func (e *ProblemDetails) IsPermission() bool { return e != nil && e.Status == http.StatusForbidden }

// IsNotFound reports a 404 not-found failure.
func (e *ProblemDetails) IsNotFound() bool { return e != nil && e.Status == http.StatusNotFound }

// IsConflict reports a 409 conflict failure.
func (e *ProblemDetails) IsConflict() bool { return e != nil && e.Status == http.StatusConflict }

// IsUnprocessable reports a 422 unprocessable-entity failure (e.g. deleting a
// deployment still referenced by a tag, pipeline, or webhook).
func (e *ProblemDetails) IsUnprocessable() bool {
	return e != nil && e.Status == http.StatusUnprocessableEntity
}

// IsRateLimited reports a 429 rate-limited failure. Inspect RetryAfter for the
// server-recommended wait.
func (e *ProblemDetails) IsRateLimited() bool {
	return e != nil && e.Status == http.StatusTooManyRequests
}

// IsServerError reports a 5xx server failure.
func (e *ProblemDetails) IsServerError() bool { return e != nil && e.Status >= 500 && e.Status < 600 }

// RetryAfter returns the server-recommended wait from the Retry-After header,
// or zero if absent or unparseable.
func (e *ProblemDetails) RetryAfter() (seconds int, ok bool) {
	if e == nil || e.Headers == nil {
		return 0, false
	}
	return parseRetryAfter(e.Headers.Get("Retry-After"))
}

// TransportError describes a failure below the API contract: a network error,
// a malformed response, or an HTTP status that did not carry a problem body.
// It never includes the request Authorization header or the Edge API key.
type TransportError struct {
	// Op is a short label for the failing operation.
	Op string
	// StatusCode is the HTTP status, or zero for a network failure.
	StatusCode int
	// Err is the underlying error.
	Err error
}

// Error implements the error interface.
func (e *TransportError) Error() string {
	if e == nil {
		return "goldsky: <nil transport error>"
	}
	if e.Err != nil {
		return fmt.Sprintf("goldsky %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("goldsky %s: transport error", e.Op)
}

// Unwrap returns the underlying error.
func (e *TransportError) Unwrap() error { return e.Err }

// AsTransport returns the *TransportError if err is (or wraps) one, else nil.
func AsTransport(err error) *TransportError {
	var t *TransportError
	if errors.As(err, &t) {
		return t
	}
	return nil
}
