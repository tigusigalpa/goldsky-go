package goldsky

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tigusigalpa/goldsky-go/internal/multipart"
)

// apiResponse is a decoded REST response. The body is fully buffered; the
// underlying connection has already been closed when do returns.
type apiResponse struct {
	statusCode int
	header     http.Header
	body       []byte
}

// requestOptions describes a REST request body and query.
type requestOptions struct {
	query     url.Values
	jsonBody  interface{}
	multipart *multipart.Body
	header    http.Header
}

// do performs a REST control-plane request against the Goldsky API, applying
// bearer authentication, retry policy, redaction, and RFC 9457 error parsing.
// path segments are URL-encoded individually and joined with "/". The REST
// project token is sent as a Bearer header and never appears in errors or logs.
func (c *Client) do(ctx context.Context, method string, segments []string, opts requestOptions) (*apiResponse, error) {
	if ctx == nil {
		return nil, &TransportError{Op: method, Err: errors.New("nil context")}
	}
	if err := ctx.Err(); err != nil {
		return nil, &TransportError{Op: method, Err: err}
	}
	if c.apiToken == "" {
		return nil, ErrAPITokenRequired
	}

	attempts := c.cfg.retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	safe := isSafeMethod(method)
	retryMutations := c.cfg.retry.RetryMutations

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		bodyReader, contentType, err := c.buildBody(opts)
		if err != nil {
			return nil, &TransportError{Op: method, Err: err}
		}

		resp, sendErr := c.sendOnce(ctx, method, segments, bodyReader, contentType, opts)
		if sendErr != nil {
			lastErr = &TransportError{Op: method, Err: sendErr}
			if (safe || retryMutations) && opts.multipart == nil && attempt < attempts {
				d := c.backoff(attempt, nil)
				c.logRetry(method, attempt, attempts, d)
				if waitErr := c.wait(ctx, d); waitErr != nil {
					return nil, &TransportError{Op: method, Err: waitErr}
				}
				continue
			}
			return nil, lastErr
		}

		body, readErr := readResponseBody(resp.Body, c.cfg.maxResponseBodyBytes)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = &TransportError{Op: method, StatusCode: resp.StatusCode, Err: readErr}
			if (safe || retryMutations) && opts.multipart == nil && attempt < attempts {
				d := c.backoff(attempt, nil)
				c.logRetry(method, attempt, attempts, d)
				if waitErr := c.wait(ctx, d); waitErr != nil {
					return nil, &TransportError{Op: method, Err: waitErr}
				}
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return &apiResponse{statusCode: resp.StatusCode, header: resp.Header, body: body}, nil
		}

		problem := parseProblem(resp, body)
		lastErr = problem

		retryable := retryableStatus(resp.StatusCode) || resp.StatusCode == 0
		canRetry := (safe || retryMutations) && opts.multipart == nil && attempt < attempts && retryable
		if !canRetry {
			return nil, problem
		}
		d := c.backoffWithRetryAfter(attempt, resp.Header)
		c.logRetry(method, attempt, attempts, d)
		if waitErr := c.wait(ctx, d); waitErr != nil {
			return nil, &TransportError{Op: method, Err: waitErr}
		}
	}
	return nil, lastErr
}

// buildBody constructs the request body reader and content type. Multipart
// bodies are streamed and therefore excluded from automatic retries.
func (c *Client) buildBody(opts requestOptions) (io.Reader, string, error) {
	if opts.multipart != nil {
		return opts.multipart, opts.multipart.ContentType(), nil
	}
	if opts.jsonBody != nil {
		buf, err := json.Marshal(opts.jsonBody)
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(buf), "application/json", nil
	}
	return nil, "", nil
}

func (c *Client) logRetry(method string, attempt, attempts int, d time.Duration) {
	if c.cfg.logger != nil {
		c.cfg.logger.Printf("retrying %s after %s (attempt %d/%d)", strings.ToUpper(method), d, attempt+1, attempts)
	}
}

// sendOnce performs a single HTTP attempt without retry logic.
func (c *Client) sendOnce(ctx context.Context, method string, segments []string, body io.Reader, contentType string, opts requestOptions) (*http.Response, error) {
	fullURL := c.buildURL(segments, opts.query)

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json, application/problem+json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("User-Agent", c.cfg.userAgent)
	for k, vs := range opts.header {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.cfg.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// buildURL joins the base URL with URL-encoded path segments and query.
func (c *Client) buildURL(segments []string, query url.Values) string {
	encoded := make([]string, 0, len(segments))
	for _, s := range segments {
		encoded = append(encoded, url.PathEscape(s))
	}
	u := c.cfg.baseURL
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	u += strings.Join(encoded, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u
}

// wait sleeps for d respecting context cancellation via the injected sleeper.
func (c *Client) wait(ctx context.Context, d time.Duration) error {
	return c.cfg.sleeper.Sleep(ctx, d)
}

// backoff computes capped exponential backoff with jitter for attempt n.
func (c *Client) backoff(attempt int, header http.Header) time.Duration {
	initial := c.cfg.retry.InitialBackoff
	if initial <= 0 {
		initial = 500 * time.Millisecond
	}
	max := c.cfg.retry.MaxBackoff
	if max <= 0 {
		max = 30 * time.Second
	}
	d := initial
	if d > max {
		d = max
	}
	for i := 1; i < attempt; i++ {
		if d >= max-d {
			d = max
			break
		}
		d *= 2
	}
	// Full jitter: randomize within [0, d].
	if d > 0 {
		jitter := time.Duration(rand.Int63n(int64(d)))
		d = jitter
	}
	return d
}

// backoffWithRetryAfter is backoff that honours the Retry-After header when it
// requests a longer wait than the computed backoff.
func (c *Client) backoffWithRetryAfter(attempt int, header http.Header) time.Duration {
	d := c.backoff(attempt, header)
	if header != nil {
		if secs, ok := parseRetryAfterAt(header.Get("Retry-After"), c.cfg.clock.Now()); ok && secs > 0 {
			ra := time.Duration(secs) * time.Second
			if ra > d {
				d = ra
			}
		}
	}
	return d
}

// parseProblem decodes an RFC 9457 problem+json body into a ProblemDetails.
// If the body is not valid problem+json, a ProblemDetails is still returned
// with the status and raw body so callers can branch on status.
func parseProblem(resp *http.Response, body []byte) *ProblemDetails {
	p := &ProblemDetails{
		Status:  resp.StatusCode,
		Headers: resp.Header.Clone(),
		RawBody: body,
	}
	if len(body) == 0 {
		if p.Type == "" {
			p.Type = "about:blank"
		}
		return p
	}
	if err := json.Unmarshal(body, p); err != nil {
		// Not a JSON problem; keep raw body and status.
		p.Type = "about:blank"
		return p
	}
	// The HTTP status line is authoritative even if a malformed server or
	// intermediary supplies a conflicting status member in the JSON body.
	p.Status = resp.StatusCode
	if p.Type == "" {
		p.Type = "about:blank"
	}
	return p
}

// decodeJSON decodes exactly one JSON value into target. Empty bodies and
// trailing values are rejected.
func decodeJSON(body []byte, target interface{}) error {
	if len(body) == 0 {
		return io.ErrUnexpectedEOF
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return fmt.Errorf("unexpected trailing data: %w", err)
	}
	return nil
}

func readResponseBody(r io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, limit))
	if err != nil {
		return nil, err
	}
	var extra [1]byte
	n, err := r.Read(extra[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if n > 0 {
		return nil, fmt.Errorf("response body exceeds %d-byte limit", limit)
	}
	return body, nil
}
