package goldsky

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/tigusigalpa/goldsky-go/internal/clock"
)

// recordedRequest captures a single request seen by the test server.
type recordedRequest struct {
	Method      string
	Path        string // decoded path
	EscapedPath string // raw, percent-encoded path
	Query       string
	Header      http.Header
	Body        []byte
}

// testServer is an httptest.Server that always records requests and returns
// either a canned response or a custom responder. It is safe for concurrent
// use within a single test.
type testServer struct {
	*httptest.Server
	mu         sync.Mutex
	requests   []recordedRequest
	status     int
	respBody   []byte
	respHeader http.Header
	responder  func(w http.ResponseWriter, r *http.Request)
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ts := &testServer{status: http.StatusOK, respHeader: http.Header{}}
	ts.Server = httptest.NewServer(http.HandlerFunc(ts.serve))
	t.Cleanup(ts.Close)
	return ts
}

func (ts *testServer) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	ts.mu.Lock()
	ts.requests = append(ts.requests, recordedRequest{
		Method:      r.Method,
		Path:        r.URL.Path,
		EscapedPath: r.URL.EscapedPath(),
		Query:       r.URL.RawQuery,
		Header:      r.Header.Clone(),
		Body:        body,
	})
	responder := ts.responder
	status := ts.status
	respBody := ts.respBody
	respHeader := ts.respHeader
	ts.mu.Unlock()

	if responder != nil {
		responder(w, r)
		return
	}
	for k, vs := range respHeader {
		for _, v := range vs {
			w.Header().Set(k, v)
		}
	}
	if w.Header().Get("Content-Type") == "" && len(respBody) > 0 {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(status)
	_, _ = w.Write(respBody)
}

func (ts *testServer) setResponder(f func(w http.ResponseWriter, r *http.Request)) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.responder = f
}

func (ts *testServer) setResponse(status int, body []byte) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.status = status
	ts.respBody = body
	ts.responder = nil
}

func (ts *testServer) setJSON(status int, v interface{}) {
	b, _ := json.Marshal(v)
	ts.setResponse(status, b)
}

func (ts *testServer) setProblem(status int, p ProblemDetails) {
	b, _ := json.Marshal(p)
	ts.mu.Lock()
	ts.status = status
	ts.respBody = b
	ts.respHeader.Set("Content-Type", "application/problem+json")
	ts.responder = nil
	ts.mu.Unlock()
}

func (ts *testServer) lastRequest() recordedRequest {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if len(ts.requests) == 0 {
		return recordedRequest{}
	}
	return ts.requests[len(ts.requests)-1]
}

func (ts *testServer) requestCount() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return len(ts.requests)
}

// newTestClient builds a Client pointed at ts with deterministic retry knobs.
func newTestClient(t *testing.T, ts *testServer, opts ...Option) *Client {
	t.Helper()
	all := append([]Option{
		WithBaseURL(ts.URL),
		WithSleeper(&clock.FakeSleeper{}),
		WithClock(clock.NewFakeClock(clock.SystemClock{}.Now())),
		WithRetryMaxAttempts(1),
	}, opts...)
	c, err := NewClient("test-token", all...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// requireBearer asserts the Authorization header carries the test bearer token.
func requireBearer(t *testing.T, h http.Header) {
	t.Helper()
	got := h.Get("Authorization")
	if got != "Bearer test-token" {
		t.Fatalf("Authorization = %q, want %q", got, "Bearer test-token")
	}
}

// jsonBody returns the pretty-ish body for comparison.
func jsonBody(b []byte) string {
	var out bytes.Buffer
	_ = json.Indent(&out, b, "", "  ")
	if out.Len() == 0 {
		return string(b)
	}
	return out.String()
}
