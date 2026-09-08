package goldsky

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ioDiscard returns a writer that discards all output.
func ioDiscard() io.Writer { return io.Discard }

// parseRetryAfter parses an HTTP Retry-After header value, which may be either
// a delta-seconds integer or an HTTP-date. It returns the wait duration.
// Unparseable values return (0, false).
func parseRetryAfter(value string) (seconds int, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if n, err := strconv.Atoi(value); err == nil {
		if n < 0 {
			return 0, false
		}
		return n, true
	}
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0, true
		}
		return int(d.Round(time.Second) / time.Second), true
	}
	return 0, false
}

// redactURL removes the query string from a URL string for safe logging. The
// Edge RPC API key is carried in the query, so it must never be logged.
func redactURL(raw string) string {
	if i := strings.IndexByte(raw, '?'); i >= 0 {
		return raw[:i]
	}
	return raw
}

// boolStr renders a bool as "0" or "1" for multipart form fields.
func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// isSafeMethod reports whether method is a retry-safe idempotent read.
func isSafeMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// retryableStatus reports whether status is one of the retryable codes.
func retryableStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}
