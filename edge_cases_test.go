package goldsky

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tigusigalpa/goldsky-go/internal/clock"
)

// TestPaginationContinuation verifies a pager keeps going when a page holds
// fewer than page_size items but still has a next_page_token.
func TestPaginationContinuation(t *testing.T) {
	ts := newTestServer(t)
	pages := []string{"tok1", "tok2", ""}
	idx := atomic.Int32{}
	ts.setResponder(func(w http.ResponseWriter, r *http.Request) {
		i := idx.Add(1)
		var tok any
		if int(i) < len(pages) && pages[i-1] != "" {
			tok = pages[i-1]
		} else {
			tok = nil
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":       []any{map[string]any{"name": "p"}},
			"pagination": map[string]any{"next_page_token": tok, "page_size": 50},
		})
	})
	c := newTestClient(t, ts)
	pager := c.Pipelines.NewPipelinePager(ListPipelinesOptions{PageSize: 50})
	var total int
	for i := 0; i < 10; i++ {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
		total += len(page.Data)
		if !page.HasMore() {
			break
		}
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if got := ts.requestCount(); got != 3 {
		t.Errorf("request count = %d, want 3", got)
	}
}

// TestURLEncoding verifies path segments with special characters are encoded.
func TestURLEncoding(t *testing.T) {
	ts := newTestServer(t)
	ts.setJSON(200, map[string]any{"data": []any{}, "pagination": map[string]any{"next_page_token": nil, "page_size": 0}})
	c := newTestClient(t, ts)
	// A name with a space must be percent-encoded in the path.
	if _, err := c.Subgraphs.Get(context.Background(), "my sub/graph"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	req := ts.lastRequest()
	if !strings.Contains(req.EscapedPath, "%20") && !strings.Contains(req.EscapedPath, "%2F") {
		t.Errorf("escaped path = %q, expected percent-encoding", req.EscapedPath)
	}
	if strings.Contains(req.EscapedPath, " ") {
		t.Errorf("escaped path = %q, contains raw space", req.EscapedPath)
	}
}

// TestRetryAfter verifies 429 is retried when Retry-After is present.
func TestRetryAfter(t *testing.T) {
	ts := newTestServer(t)
	var attempts atomic.Int32
	ts.setResponder(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"type":"https://api.goldsky.com/api/errors/rate-limited","title":"Rate limited","status":429}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"pagination":{"next_page_token":null,"page_size":0}}`))
	})
	c := newTestClient(t, ts, WithRetryMaxAttempts(3), WithSleeper(&clock.FakeSleeper{}))
	_, err := c.Pipelines.List(context.Background(), ListPipelinesOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

// TestNoMutationRetry verifies a POST is not retried by default on 500.
func TestNoMutationRetry(t *testing.T) {
	ts := newTestServer(t)
	var attempts atomic.Int32
	ts.setResponder(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"https://api.goldsky.com/api/errors/internal","title":"Internal","status":500}`))
	})
	c := newTestClient(t, ts, WithRetryMaxAttempts(3), WithSleeper(&clock.FakeSleeper{}))
	_, err := c.Pipelines.Create(context.Background(), CreatePipelineRequest{Definition: PipelineDefinition{Sources: map[string]any{}, Transforms: map[string]any{}, Sinks: map[string]any{}}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("mutation attempts = %d, want 1 (no retry)", got)
	}
	p := AsProblem(err)
	if p == nil || !p.IsServerError() {
		t.Fatalf("expected server error problem, got %v", err)
	}
}

// TestMutationRetryOptIn verifies a POST is retried when opted in.
func TestMutationRetryOptIn(t *testing.T) {
	ts := newTestServer(t)
	var attempts atomic.Int32
	ts.setResponder(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"type":"x","title":"unavailable","status":503}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"p","type":"t","status":"RUNNING","definition":{"sources":{},"transforms":{},"sinks":{}},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`))
	})
	c := newTestClient(t, ts, WithRetryMaxAttempts(3), WithRetryMutations(), WithSleeper(&clock.FakeSleeper{}))
	_, err := c.Pipelines.Create(context.Background(), CreatePipelineRequest{Definition: PipelineDefinition{Sources: map[string]any{}, Transforms: map[string]any{}, Sinks: map[string]any{}}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("attempts = %d, want 2", got)
	}
}

// TestProblemParsing verifies RFC 9457 problem details are parsed.
func TestProblemParsing(t *testing.T) {
	ts := newTestServer(t)
	ts.setProblem(404, ProblemDetails{
		Type:   "https://api.goldsky.com/api/errors/subgraph-not-found",
		Title:  "Subgraph not found",
		Status: 404,
		Detail: "No subgraph named 'x' exists.",
		Errors: []ValidationError{{Field: "name", Message: "missing"}},
	})
	c := newTestClient(t, ts)
	err := c.Subgraphs.DeleteDeployment(context.Background(), "x", "v1")
	p := AsProblem(err)
	if p == nil {
		t.Fatalf("expected problem, got %v", err)
	}
	if p.Type != "https://api.goldsky.com/api/errors/subgraph-not-found" {
		t.Errorf("type = %q", p.Type)
	}
	if !p.IsNotFound() {
		t.Errorf("IsNotFound = false")
	}
	if len(p.Errors) != 1 || p.Errors[0].Field != "name" {
		t.Errorf("validation errors = %+v", p.Errors)
	}
}

// TestRedaction verifies the API token never appears in error strings.
func TestRedaction(t *testing.T) {
	ts := newTestServer(t)
	ts.setResponse(500, []byte(`{"type":"x","title":"boom","status":500}`))
	c := newTestClient(t, ts)
	_, err := c.Pipelines.List(context.Background(), ListPipelinesOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "test-token") {
		t.Errorf("error leaks token: %s", err.Error())
	}
}

// TestMalformedJSON verifies malformed JSON is a transport error.
func TestMalformedJSON(t *testing.T) {
	ts := newTestServer(t)
	ts.setResponse(200, []byte("{not json"))
	c := newTestClient(t, ts)
	_, err := c.Pipelines.List(context.Background(), ListPipelinesOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if AsTransport(err) == nil {
		t.Errorf("expected TransportError, got %T: %v", err, err)
	}
}

// TestContextCancellation verifies a cancelled context aborts the request.
func TestContextCancellation(t *testing.T) {
	ts := newTestServer(t)
	ts.setResponder(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	})
	c := newTestClient(t, ts)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := c.Pipelines.List(ctx, ListPipelinesOptions{})
	if err == nil {
		t.Fatal("expected context error")
	}
}

// TestMultipartFields verifies all documented deploy fields are emitted.
func TestMultipartFields(t *testing.T) {
	ts := newTestServer(t)
	ts.setJSON(201, map[string]any{"name": "s", "version": "v1", "status": "ACTIVE", "network": "mainnet", "health": "HEALTHY", "synced": true, "graphql_endpoint": "", "private_graphql_endpoint": "", "public_endpoint_enabled": true, "private_endpoint_enabled": false, "description": nil, "deployments": []any{}})
	c := newTestClient(t, ts)
	bundle := bytes.NewReader(bytes.Repeat([]byte{1}, 32))
	_, err := c.Subgraphs.Deploy(context.Background(), "s", "v1", DeploySubgraphOptions{
		Bundle: bundle, BundleFilename: "build.zip",
		Overwrite: "0", RemoveGraft: "1", SkipGraftValidation: "1",
		StartBlock: "1000", GraftFrom: "other/v2", Description: "desc", GraphNodeShard: "shard-1",
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	body := string(ts.lastRequest().Body)
	for _, want := range []string{"bundle", "build.zip", "overwrite", "remove_graft", "skip_graft_validation", "start_block", "graft_from", "description", "graph_node_shard"} {
		if !strings.Contains(body, want) {
			t.Errorf("multipart body missing %q", want)
		}
	}
}

// TestDeployRejectsOverwriteOne verifies overwrite=1 is rejected locally.
func TestDeployRejectsOverwriteOne(t *testing.T) {
	ts := newTestServer(t)
	c := newTestClient(t, ts)
	_, err := c.Subgraphs.Deploy(context.Background(), "s", "v1", DeploySubgraphOptions{
		Bundle: bytes.NewReader([]byte{1}), BundleFilename: "build.zip", Overwrite: "1",
	})
	if err == nil || !strings.Contains(err.Error(), "overwrite=1") {
		t.Fatalf("expected overwrite=1 rejection, got %v", err)
	}
	if ts.requestCount() != 0 {
		t.Errorf("no request should have been sent")
	}
}

// TestEdge204Deletion verifies Edge deletion returns nil on 204.
func TestEdge204Deletion(t *testing.T) {
	ts := newTestServer(t)
	ts.setResponse(204, nil)
	c := newTestClient(t, ts)
	if err := c.Edge.Delete(context.Background(), "ep"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ts.lastRequest().Method != "DELETE" {
		t.Errorf("method = %q", ts.lastRequest().Method)
	}
}

// TestWebhookVerifier verifies constant-time comparison and edge cases.
func TestWebhookVerifier(t *testing.T) {
	if !VerifyWebhookSecret("s", "s") {
		t.Error("matching secrets should verify")
	}
	if VerifyWebhookSecret("s", "x") {
		t.Error("mismatched secrets should not verify")
	}
	if VerifyWebhookSecret("s", "") {
		t.Error("empty expected should not verify")
	}
	r, _ := http.NewRequest("POST", "https://example.com", nil)
	r.Header.Set(WebhookSecretHeader, "s")
	if !VerifyWebhookRequest(r, "s") {
		t.Error("request with matching header should verify")
	}
	if VerifyWebhookRequest(r, "x") {
		t.Error("request with mismatched expected should not verify")
	}
}

// TestGraphQLErrorEnvelope verifies GraphQL error envelopes are decoded.
func TestGraphQLErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"field x is required","path":["x"]}],"data":null}`))
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, newTestServer(t))
	resp, err := c.GraphQL.Query(context.Background(), srv.URL, GraphQLRequest{Query: "{ x }"}, false)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if !resp.HasErrors() {
		t.Fatal("expected errors")
	}
	if len(resp.Errors) != 1 || resp.Errors[0].Message != "field x is required" {
		t.Errorf("errors = %+v", resp.Errors)
	}
	if len(resp.Errors[0].Raw) == 0 {
		t.Error("Raw error object not retained")
	}
}

// TestEdgeRPCSingle verifies a single JSON-RPC call decodes the result.
func TestEdgeRPCSingle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.String(), "key=edge-key") {
			t.Errorf("url missing key: %s", redactURL(r.URL.String()))
		}
		body, _ := io.ReadAll(r.Body)
		var req rpcRequest
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + jsonInt(req.ID) + `,"result":"0x1234"}`))
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, newTestServer(t), WithEdgeAPIKey("edge-key"))
	c.RPC.baseURL = srv.URL
	var result string
	if err := c.RPC.Call(context.Background(), 1, "eth_blockNumber", nil, &result); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result != "0x1234" {
		t.Errorf("result = %q", result)
	}
}

// TestEdgeRPCBatch verifies a batch call returns ordered responses.
func TestEdgeRPCBatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqs []rpcRequest
		_ = json.Unmarshal(body, &reqs)
		out := make([]map[string]any, len(reqs))
		for i, req := range reqs {
			out[i] = map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": req.Method}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, newTestServer(t), WithEdgeAPIKey("edge-key"))
	c.RPC.baseURL = srv.URL
	var r1, r2 string
	resps, err := c.RPC.Batch(context.Background(), 1, []RPCBatchCall{
		{Method: "eth_blockNumber", Result: &r1},
		{Method: "eth_chainId", Result: &r2},
	})
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}
	if len(resps) != 2 {
		t.Fatalf("resps = %d", len(resps))
	}
	if r1 != "eth_blockNumber" || r2 != "eth_chainId" {
		t.Errorf("results = %q, %q", r1, r2)
	}
}

// TestEdgeRPCError verifies a JSON-RPC error is returned as *RPCError.
func TestEdgeRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req rpcRequest
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + jsonInt(req.ID) + `,"error":{"code":-32601,"message":"method not found"}}`))
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, newTestServer(t), WithEdgeAPIKey("edge-key"))
	c.RPC.baseURL = srv.URL
	err := c.RPC.Call(context.Background(), 1, "nope", nil, nil)
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("expected *RPCError, got %v", err)
	}
	if rpcErr.Code != -32601 {
		t.Errorf("code = %d", rpcErr.Code)
	}
}

// TestNetworkTimeout verifies transport errors are wrapped.
func TestNetworkTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient("test-token", WithBaseURL(srv.URL), WithTimeout(20*time.Millisecond), WithRetryMaxAttempts(1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Pipelines.List(context.Background(), ListPipelinesOptions{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if AsTransport(err) == nil {
		t.Errorf("expected TransportError, got %T", err)
	}
}

// TestNewClientRequiresToken verifies a token is required.
func TestNewClientRequiresToken(t *testing.T) {
	if _, err := NewClient(""); err == nil {
		t.Fatal("expected error for empty token")
	}
}

// TestGraphQLURLs verifies public/private URL construction.
func TestGraphQLURLs(t *testing.T) {
	c := newTestClient(t, newTestServer(t))
	pub := c.GraphQL.PublicURL("proj", "sub", "v1")
	priv := c.GraphQL.PrivateURL("proj", "sub", "v1")
	if !strings.HasSuffix(pub, "/api/public/proj/subgraphs/sub/v1/gn") {
		t.Errorf("public url = %q", pub)
	}
	if !strings.HasSuffix(priv, "/api/private/proj/subgraphs/sub/v1/gn") {
		t.Errorf("private url = %q", priv)
	}
}

// TestEdgeEndpointURL verifies the Edge RPC URL includes the chain ID and key.
func TestEdgeEndpointURL(t *testing.T) {
	c := newTestClient(t, newTestServer(t), WithEdgeAPIKey("ek"))
	u := c.RPC.EndpointURL(137)
	if !strings.Contains(u, "/evm/137?") || !strings.Contains(u, "key=ek") {
		t.Errorf("endpoint url = %q", u)
	}
}

// jsonInt renders an int64 as a JSON number string.
func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
