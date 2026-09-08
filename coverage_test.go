package goldsky

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestProblemDetailsHelpers(t *testing.T) {
	tests := []struct {
		status int
		check  func(*ProblemDetails) bool
	}{
		{http.StatusBadRequest, (*ProblemDetails).IsValidation},
		{http.StatusUnauthorized, (*ProblemDetails).IsAuthentication},
		{http.StatusPaymentRequired, (*ProblemDetails).IsSubscription},
		{http.StatusForbidden, (*ProblemDetails).IsPermission},
		{http.StatusNotFound, (*ProblemDetails).IsNotFound},
		{http.StatusConflict, (*ProblemDetails).IsConflict},
		{http.StatusUnprocessableEntity, (*ProblemDetails).IsUnprocessable},
		{http.StatusTooManyRequests, (*ProblemDetails).IsRateLimited},
		{http.StatusInternalServerError, (*ProblemDetails).IsServerError},
	}
	for _, tt := range tests {
		p := &ProblemDetails{Status: tt.status}
		if !tt.check(p) {
			t.Errorf("status %d was not classified", tt.status)
		}
	}
	var nilProblem *ProblemDetails
	if nilProblem.IsValidation() || nilProblem.IsAuthentication() || nilProblem.IsSubscription() ||
		nilProblem.IsPermission() || nilProblem.IsNotFound() || nilProblem.IsConflict() ||
		nilProblem.IsUnprocessable() || nilProblem.IsRateLimited() || nilProblem.IsServerError() {
		t.Fatal("nil problem must not match a classification")
	}

	errorCases := []*ProblemDetails{
		nil,
		{Type: "x", Detail: "detail"},
		{Type: "x", Title: "title"},
		{Type: "x"},
		{Type: "x", Status: 400, Detail: "detail"},
		{Type: "x", Status: 400, Title: "title"},
		{Type: "x", Status: 400},
	}
	for _, p := range errorCases {
		if got := p.Error(); got == "" {
			t.Error("empty ProblemDetails.Error result")
		}
	}

	p := &ProblemDetails{Type: "x", Status: 404}
	if !errors.Is(p, &ProblemDetails{Type: "x"}) || !errors.Is(p, &ProblemDetails{Status: 404}) {
		t.Fatal("errors.Is did not match problem type/status")
	}
	if errors.Is(p, &ProblemDetails{Type: "y"}) || errors.Is(p, errors.New("other")) {
		t.Fatal("errors.Is matched the wrong problem")
	}
	if AsProblem(errors.New("other")) != nil {
		t.Fatal("AsProblem matched a regular error")
	}

	p.Headers = http.Header{"Retry-After": []string{"12"}}
	if seconds, ok := p.RetryAfter(); !ok || seconds != 12 {
		t.Fatalf("RetryAfter = %d, %t", seconds, ok)
	}
	if seconds, ok := (*ProblemDetails)(nil).RetryAfter(); ok || seconds != 0 {
		t.Fatalf("nil RetryAfter = %d, %t", seconds, ok)
	}
}

func TestTransportAndRPCErrorHelpers(t *testing.T) {
	var nilTransport *TransportError
	if nilTransport.Error() == "" {
		t.Fatal("nil transport error string is empty")
	}
	base := errors.New("network down")
	transport := &TransportError{Op: "GET", Err: base}
	if !errors.Is(transport, base) || AsTransport(transport) != transport {
		t.Fatal("transport wrapping failed")
	}
	if (&TransportError{Op: "GET"}).Error() == "" || AsTransport(base) != nil {
		t.Fatal("transport helper result is invalid")
	}
	var nilRPC *RPCError
	if nilRPC.Error() == "" || (&RPCError{Code: -1, Message: "boom"}).Error() == "" {
		t.Fatal("RPC error string is empty")
	}
}

func TestClientOptionsAndUtilities(t *testing.T) {
	ts := newTestServer(t)
	logger := log.New(io.Discard, "", 0)
	c, err := NewClient(" token ",
		WithBaseURL(ts.URL+"/api"),
		WithUserAgent("coverage-test"),
		WithRetryPolicy(RetryPolicy{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Second}),
		WithLogger(logger),
		WithEdgeBaseURL(ts.URL+"/rpc/"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.BaseURL() != ts.URL+"/api" || c.UserAgent() != "coverage-test" {
		t.Fatalf("client configuration = %q, %q", c.BaseURL(), c.UserAgent())
	}
	if got := c.RPC.EndpointURL(1); !strings.HasPrefix(got, ts.URL+"/rpc/1?") {
		t.Fatalf("RPC URL = %q", got)
	}

	if redactURL("https://example.com/path?key=secret") != "https://example.com/path" || redactURL("plain") != "plain" {
		t.Fatal("redactURL returned an unexpected value")
	}
	if seconds, ok := parseRetryAfter("7"); !ok || seconds != 7 {
		t.Fatalf("parseRetryAfter = %d, %t", seconds, ok)
	}
	for _, value := range []string{"", "-1", "not-a-date"} {
		if _, ok := parseRetryAfter(value); ok {
			t.Fatalf("parseRetryAfter(%q) unexpectedly succeeded", value)
		}
	}
}

func TestServiceDecodeFailuresAreTransportErrors(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{"catalog chains", func(c *Client) error { _, err := c.Catalogs.SupportedSubgraphChains(context.Background()); return err }},
		{"catalog networks", func(c *Client) error { _, err := c.Catalogs.EdgeNetworks(context.Background()); return err }},
		{"catalog sources", func(c *Client) error { _, err := c.Catalogs.EdgeSources(context.Background()); return err }},
		{"pipeline create", func(c *Client) error {
			_, err := c.Pipelines.Create(context.Background(), CreatePipelineRequest{Definition: emptyPipelineDefinition()})
			return err
		}},
		{"pipeline get", func(c *Client) error { _, err := c.Pipelines.Get(context.Background(), "pipeline"); return err }},
		{"pipeline validate", func(c *Client) error {
			_, err := c.Pipelines.Validate(context.Background(), ValidatePipelineRequest{Definition: emptyPipelineDefinition()})
			return err
		}},
		{"pipeline preview", func(c *Client) error {
			_, err := c.Pipelines.Preview(context.Background(), PreviewPipelineRequest{Definition: emptyPipelineDefinition()})
			return err
		}},
		{"pipeline logs", func(c *Client) error {
			_, err := c.Pipelines.Logs(context.Background(), "pipeline", PipelineLogsOptions{})
			return err
		}},
		{"pipeline errors", func(c *Client) error {
			_, err := c.Pipelines.ErrorCount(context.Background(), "pipeline", 1)
			return err
		}},
		{"pipeline status", func(c *Client) error { _, err := c.Pipelines.Status(context.Background(), "pipeline"); return err }},
		{"subgraph list", func(c *Client) error {
			_, err := c.Subgraphs.List(context.Background(), ListSubgraphsOptions{})
			return err
		}},
		{"subgraph chains", func(c *Client) error { _, err := c.Subgraphs.SupportedChains(context.Background()); return err }},
		{"subgraph version", func(c *Client) error {
			_, err := c.Subgraphs.GetVersion(context.Background(), "subgraph", "v1")
			return err
		}},
		{"subgraph update", func(c *Client) error {
			_, err := c.Subgraphs.UpdateVersion(context.Background(), "subgraph", "v1", UpdateSubgraphVersionRequest{})
			return err
		}},
		{"subgraph logs", func(c *Client) error {
			_, err := c.Subgraphs.Logs(context.Background(), "subgraph", "v1", SubgraphLogsOptions{})
			return err
		}},
		{"subgraph tag", func(c *Client) error {
			_, err := c.Subgraphs.SetTag(context.Background(), "subgraph", "tag", SetSubgraphTagRequest{TargetVersion: "v1"})
			return err
		}},
		{"subgraph entities", func(c *Client) error {
			_, err := c.Subgraphs.WebhookEntities(context.Background(), "subgraph", "v1")
			return err
		}},
		{"edge list", func(c *Client) error {
			_, err := c.Edge.List(context.Background(), ListEdgeEndpointsOptions{})
			return err
		}},
		{"edge create", func(c *Client) error {
			_, err := c.Edge.Create(context.Background(), CreateEdgeEndpointRequest{Name: "edge"})
			return err
		}},
		{"edge get", func(c *Client) error { _, err := c.Edge.Get(context.Background(), "edge"); return err }},
		{"edge update", func(c *Client) error {
			_, err := c.Edge.Update(context.Background(), "edge", UpdateEdgeEndpointRequest{})
			return err
		}},
		{"edge pause", func(c *Client) error { _, err := c.Edge.Pause(context.Background(), "edge"); return err }},
		{"edge resume", func(c *Client) error { _, err := c.Edge.Resume(context.Background(), "edge"); return err }},
		{"edge reveal", func(c *Client) error { _, err := c.Edge.RevealKey(context.Background(), "edge"); return err }},
		{"edge metrics", func(c *Client) error {
			_, err := c.Edge.Metrics(context.Background(), "edge", EdgeMetricsOptions{})
			return err
		}},
		{"webhook list", func(c *Client) error { _, err := c.Webhooks.List(context.Background()); return err }},
		{"webhook create", func(c *Client) error {
			_, err := c.Webhooks.Create(context.Background(), CreateWebhookRequest{})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestServer(t)
			ts.setResponse(http.StatusOK, []byte("not-json"))
			err := tt.call(newTestClient(t, ts))
			if AsTransport(err) == nil {
				t.Fatalf("error = %T %v, want TransportError", err, err)
			}
		})
	}
}

func TestGraphQLConvenienceQueries(t *testing.T) {
	ts := newTestServer(t)
	ts.setResponse(http.StatusOK, []byte(`{"data":{"ok":true}}`))
	c := newTestClient(t, ts)
	c.GraphQL.baseURL = ts.URL + "/api"

	if _, err := c.GraphQL.QueryPublic(context.Background(), "project", "subgraph", "v1", GraphQLRequest{Query: "{ ok }"}); err != nil {
		t.Fatalf("QueryPublic: %v", err)
	}
	if got := ts.lastRequest().Header.Get("Authorization"); got != "" {
		t.Fatalf("public Authorization = %q", got)
	}
	if _, err := c.GraphQL.QueryPrivate(context.Background(), "project", "subgraph", "v1", GraphQLRequest{Query: "{ ok }"}); err != nil {
		t.Fatalf("QueryPrivate: %v", err)
	}
	requireBearer(t, ts.lastRequest().Header)
}

func TestEdgePager(t *testing.T) {
	ts := newTestServer(t)
	ts.setJSON(http.StatusOK, map[string]any{
		"data":       []any{},
		"pagination": map[string]any{"next_page_token": nil, "page_size": 20},
	})
	c := newTestClient(t, ts)
	pager := c.Edge.NewEdgePager(ListEdgeEndpointsOptions{Product: "rpc", PageSize: 20})
	if _, err := pager.NextPage(context.Background()); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if !strings.Contains(ts.lastRequest().Query, "product=rpc") {
		t.Fatalf("query = %q", ts.lastRequest().Query)
	}
}

func emptyPipelineDefinition() PipelineDefinition {
	return PipelineDefinition{Sources: map[string]any{}, Transforms: map[string]any{}, Sinks: map[string]any{}}
}
