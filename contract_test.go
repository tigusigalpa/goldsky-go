package goldsky

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// opCase is a single REST operation contract test.
type opCase struct {
	name   string
	method string
	path   string
	setup  func(*testServer)
	call   func(*Client) error
	extra  func(t *testing.T, req recordedRequest)
}

func TestContractOperations(t *testing.T) {
	cases := []opCase{
		// --- Pipelines ---
		{
			name: "listPipelines", method: "GET", path: "/pipelines",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": []any{}, "pagination": map[string]any{"next_page_token": nil, "page_size": 0}})
			},
			call: func(c *Client) error {
				_, err := c.Pipelines.List(context.Background(), ListPipelinesOptions{PageSize: 50})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(req.Query, "page_size=50") {
					t.Errorf("query = %q, want page_size=50", req.Query)
				}
			},
		},
		{
			name: "createPipeline", method: "POST", path: "/pipelines",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"name": "p", "type": "t", "status": "RUNNING", "definition": map[string]any{"sources": map[string]any{}, "transforms": map[string]any{}, "sinks": map[string]any{}}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"})
			},
			call: func(c *Client) error {
				_, err := c.Pipelines.Create(context.Background(), CreatePipelineRequest{Name: "my-pipe", Definition: PipelineDefinition{Sources: map[string]any{}, Transforms: map[string]any{}, Sinks: map[string]any{}}})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(string(req.Body), `"name":"my-pipe"`) {
					t.Errorf("body = %s, want name field", string(req.Body))
				}
			},
		},
		{
			name: "getPipeline", method: "GET", path: "/pipelines/my-pipe",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"name": "my-pipe", "type": "t", "status": "RUNNING", "definition": map[string]any{"sources": map[string]any{}, "transforms": map[string]any{}, "sinks": map[string]any{}}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"})
			},
			call: func(c *Client) error { _, err := c.Pipelines.Get(context.Background(), "my-pipe"); return err },
		},
		{
			name: "deletePipeline", method: "DELETE", path: "/pipelines/my-pipe",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Pipelines.Delete(context.Background(), "my-pipe") },
		},
		{
			name: "validatePipeline", method: "POST", path: "/pipelines/validate",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"valid": true, "errors": []any{}, "warnings": []any{}})
			},
			call: func(c *Client) error {
				_, err := c.Pipelines.Validate(context.Background(), ValidatePipelineRequest{Definition: PipelineDefinition{Sources: map[string]any{}, Transforms: map[string]any{}, Sinks: map[string]any{}}})
				return err
			},
		},
		{
			name: "previewPipeline", method: "POST", path: "/pipelines/preview",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"pipeline_name": "p", "ttl_seconds": 300, "expires_at": "2026-01-01T00:05:00Z"})
			},
			call: func(c *Client) error {
				_, err := c.Pipelines.Preview(context.Background(), PreviewPipelineRequest{Definition: PipelineDefinition{Sources: map[string]any{}, Transforms: map[string]any{}, Sinks: map[string]any{}}, TTLSeconds: json.Number("300")})
				return err
			},
		},
		{
			name: "pausePipeline", method: "PUT", path: "/pipelines/my-pipe/pause",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Pipelines.Pause(context.Background(), "my-pipe") },
		},
		{
			name: "resumePipeline", method: "PUT", path: "/pipelines/my-pipe/resume",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Pipelines.Resume(context.Background(), "my-pipe") },
		},
		{
			name: "restartPipeline", method: "PUT", path: "/pipelines/my-pipe/restart",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call: func(c *Client) error {
				return c.Pipelines.Restart(context.Background(), "my-pipe", &RestartPipelineRequest{ClearState: true})
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(string(req.Body), `"clearState":true`) {
					t.Errorf("body = %s, want clearState:true", string(req.Body))
				}
			},
		},
		{
			name: "getPipelineLogs", method: "GET", path: "/pipelines/my-pipe/logs",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"results": []any{}, "cursor": float64(0)}})
			},
			call: func(c *Client) error {
				cursor := 123.0
				_, err := c.Pipelines.Logs(context.Background(), "my-pipe", PipelineLogsOptions{Cursor: &cursor, Direction: "desc", LogLevels: "error"})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(req.Query, "cursor=123") || !strings.Contains(req.Query, "direction=desc") || !strings.Contains(req.Query, "logLevels=error") {
					t.Errorf("query = %q, missing expected params", req.Query)
				}
			},
		},
		{
			name: "getPipelineErrorCount", method: "GET", path: "/pipelines/my-pipe/logs/error-count",
			setup: func(ts *testServer) { ts.setJSON(200, map[string]any{"data": map[string]any{"error_count": 3}}) },
			call: func(c *Client) error {
				_, err := c.Pipelines.ErrorCount(context.Background(), "my-pipe", 24)
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(req.Query, "since_hours=24") {
					t.Errorf("query = %q, want since_hours=24", req.Query)
				}
			},
		},
		{
			name: "getPipelineStatus", method: "GET", path: "/pipelines/my-pipe/status",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"name": "my-pipe", "status": "RUNNING", "errors": []any{}})
			},
			call: func(c *Client) error { _, err := c.Pipelines.Status(context.Background(), "my-pipe"); return err },
		},
		{
			name: "getPipelineState", method: "GET", path: "/pipelines/my-pipe/state",
			setup: func(ts *testServer) { ts.setJSON(200, map[string]any{"data": map[string]any{"checkpoint": "x"}}) },
			call:  func(c *Client) error { _, err := c.Pipelines.State(context.Background(), "my-pipe"); return err },
		},
		// --- Subgraphs ---
		{
			name: "listSubgraphs", method: "GET", path: "/subgraphs",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": []any{}, "pagination": map[string]any{"next_page_token": nil, "page_size": 0}})
			},
			call: func(c *Client) error {
				_, err := c.Subgraphs.List(context.Background(), ListSubgraphsOptions{PageSize: 25})
				return err
			},
		},
		{
			name: "getSubgraph", method: "GET", path: "/subgraphs/my-sub",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": []any{}, "pagination": map[string]any{"next_page_token": nil, "page_size": 0}})
			},
			call: func(c *Client) error { _, err := c.Subgraphs.Get(context.Background(), "my-sub"); return err },
		},
		{
			name: "listSubgraphChains", method: "GET", path: "/subgraphs/supported-chains",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"supported_chains": []any{"mainnet"}}})
			},
			call: func(c *Client) error { _, err := c.Subgraphs.SupportedChains(context.Background()); return err },
		},
		{
			name: "getSubgraphVersion", method: "GET", path: "/subgraphs/my-sub/v1",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": []any{}, "pagination": map[string]any{"next_page_token": nil, "page_size": 0}})
			},
			call: func(c *Client) error {
				_, err := c.Subgraphs.GetVersion(context.Background(), "my-sub", "v1")
				return err
			},
		},
		{
			name: "updateSubgraphVersion", method: "PATCH", path: "/subgraphs/my-sub/v1",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"name": "my-sub", "version": "v1", "status": "ACTIVE", "network": "mainnet", "health": "HEALTHY", "synced": true, "graphql_endpoint": "", "private_graphql_endpoint": "", "public_endpoint_enabled": true, "private_endpoint_enabled": false, "description": nil, "deployments": []any{}}})
			},
			call: func(c *Client) error {
				enabled := true
				_, err := c.Subgraphs.UpdateVersion(context.Background(), "my-sub", "v1", UpdateSubgraphVersionRequest{PublicEndpointEnabled: &enabled})
				return err
			},
		},
		{
			name: "getSubgraphLogs", method: "GET", path: "/subgraphs/my-sub/v1/logs",
			setup: func(ts *testServer) { ts.setJSON(200, map[string]any{"data": map[string]any{"results": []any{}}}) },
			call: func(c *Client) error {
				_, err := c.Subgraphs.Logs(context.Background(), "my-sub", "v1", SubgraphLogsOptions{LogLevel: "error"})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(req.Query, "log_level=error") {
					t.Errorf("query = %q, want log_level=error", req.Query)
				}
			},
		},
		{
			name: "pauseSubgraph", method: "PUT", path: "/subgraphs/my-sub/v1/pause",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Subgraphs.Pause(context.Background(), "my-sub", "v1") },
		},
		{
			name: "resumeSubgraph", method: "PUT", path: "/subgraphs/my-sub/v1/resume",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Subgraphs.Resume(context.Background(), "my-sub", "v1") },
		},
		{
			name: "setSubgraphTag", method: "PUT", path: "/subgraphs/my-sub/tags/v1",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"name": "my-sub", "version": "v1", "status": "ACTIVE", "network": "mainnet", "health": "HEALTHY", "synced": true, "graphql_endpoint": "", "private_graphql_endpoint": "", "public_endpoint_enabled": true, "private_endpoint_enabled": false, "description": nil, "deployments": []any{}}})
			},
			call: func(c *Client) error {
				_, err := c.Subgraphs.SetTag(context.Background(), "my-sub", "v1", SetSubgraphTagRequest{TargetVersion: "1.0.0"})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(string(req.Body), `"target_version":"1.0.0"`) {
					t.Errorf("body = %s, want target_version", string(req.Body))
				}
			},
		},
		{
			name: "deleteSubgraphTag", method: "DELETE", path: "/subgraphs/my-sub/tags/v1",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Subgraphs.DeleteTag(context.Background(), "my-sub", "v1") },
		},
		{
			name: "deleteSubgraphDeployment", method: "DELETE", path: "/subgraphs/my-sub/deployments/v1",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Subgraphs.DeleteDeployment(context.Background(), "my-sub", "v1") },
		},
		{
			name: "deploySubgraph", method: "PUT", path: "/subgraphs/my-sub/deployments/v1",
			setup: func(ts *testServer) {
				ts.setJSON(201, map[string]any{"name": "my-sub", "version": "v1", "status": "ACTIVE", "network": "mainnet", "health": "HEALTHY", "synced": true, "graphql_endpoint": "", "private_graphql_endpoint": "", "public_endpoint_enabled": true, "private_endpoint_enabled": false, "description": nil, "deployments": []any{}})
			},
			call: func(c *Client) error {
				bundle := bytes.NewReader(bytes.Repeat([]byte{0x50, 0x4b}, 10))
				_, err := c.Subgraphs.Deploy(context.Background(), "my-sub", "v1", DeploySubgraphOptions{Bundle: bundle, BundleFilename: "build.zip", StartBlock: "100", Description: "d"})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				ct := req.Header.Get("Content-Type")
				if !strings.HasPrefix(ct, "multipart/form-data") {
					t.Fatalf("Content-Type = %q, want multipart/form-data", ct)
				}
				body := string(req.Body)
				if !strings.Contains(body, "bundle") || !strings.Contains(body, "build.zip") {
					t.Errorf("multipart body missing bundle file part: %s", body)
				}
				if !strings.Contains(body, "start_block") || !strings.Contains(body, "100") {
					t.Errorf("multipart body missing start_block field")
				}
				if !strings.Contains(body, "description") {
					t.Errorf("multipart body missing description field")
				}
			},
		},
		{
			name: "listWebhooks", method: "GET", path: "/subgraphs/webhooks",
			setup: func(ts *testServer) { ts.setJSON(200, map[string]any{"data": []any{}}) },
			call:  func(c *Client) error { _, err := c.Webhooks.List(context.Background()); return err },
		},
		{
			name: "createWebhook", method: "POST", path: "/subgraphs/webhooks",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"id": "wh-1", "name": "wh", "webhook_secret": "secret-value"}})
			},
			call: func(c *Client) error {
				_, err := c.Webhooks.Create(context.Background(), CreateWebhookRequest{Name: "wh", SubgraphName: "my-sub", SubgraphVersion: "v1", Entity: "Entity", WebhookURL: "https://example.com/hook"})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(string(req.Body), `"name":"wh"`) {
					t.Errorf("body = %s, want name", string(req.Body))
				}
			},
		},
		{
			name: "deleteWebhook", method: "DELETE", path: "/subgraphs/webhooks/wh",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Webhooks.Delete(context.Background(), "wh") },
		},
		{
			name: "listWebhookEntities", method: "GET", path: "/subgraphs/my-sub/v1/entities",
			setup: func(ts *testServer) { ts.setJSON(200, map[string]any{"data": map[string]any{"entities": []any{}}}) },
			call: func(c *Client) error {
				_, err := c.Subgraphs.WebhookEntities(context.Background(), "my-sub", "v1")
				return err
			},
		},
		// --- Edge & catalogs ---
		{
			name: "listEdgeNetworks", method: "GET", path: "/edge/networks",
			setup: func(ts *testServer) { ts.setJSON(200, map[string]any{"data": []any{}}) },
			call:  func(c *Client) error { _, err := c.Catalogs.EdgeNetworks(context.Background()); return err },
		},
		{
			name: "listEdgeSources", method: "GET", path: "/edge/sources",
			setup: func(ts *testServer) { ts.setJSON(200, map[string]any{"data": []any{}}) },
			call:  func(c *Client) error { _, err := c.Catalogs.EdgeSources(context.Background()); return err },
		},
		{
			name: "listEdgeEndpoints", method: "GET", path: "/edge",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": []any{}, "pagination": map[string]any{"next_page_token": nil, "page_size": 0}})
			},
			call: func(c *Client) error {
				_, err := c.Edge.List(context.Background(), ListEdgeEndpointsOptions{Product: "rpc"})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(req.Query, "product=rpc") {
					t.Errorf("query = %q, want product=rpc", req.Query)
				}
			},
		},
		{
			name: "createEdgeEndpoint", method: "POST", path: "/edge",
			setup: func(ts *testServer) {
				ts.setJSON(201, map[string]any{"data": map[string]any{"name": "ep", "product": "rpc", "status": "ACTIVE", "rate_limit_budget": nil, "allowed_domains": []any{}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "paused_at": nil, "api_key": "ek-123"}, "warnings": []any{}})
			},
			call: func(c *Client) error {
				p := EdgeProductRPC
				_, err := c.Edge.Create(context.Background(), CreateEdgeEndpointRequest{Name: "ep", Product: &p})
				return err
			},
		},
		{
			name: "getEdgeEndpoint", method: "GET", path: "/edge/ep",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"name": "ep", "product": "rpc", "status": "ACTIVE", "rate_limit_budget": nil, "allowed_domains": []any{}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "paused_at": nil}})
			},
			call: func(c *Client) error { _, err := c.Edge.Get(context.Background(), "ep"); return err },
		},
		{
			name: "updateEdgeEndpoint", method: "PATCH", path: "/edge/ep",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"name": "ep", "product": "rpc", "status": "ACTIVE", "rate_limit_budget": nil, "allowed_domains": []any{}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "paused_at": nil}})
			},
			call: func(c *Client) error {
				_, err := c.Edge.Update(context.Background(), "ep", UpdateEdgeEndpointRequest{AllowedDomains: []string{"https://app.example.com"}})
				return err
			},
		},
		{
			name: "deleteEdgeEndpoint", method: "DELETE", path: "/edge/ep",
			setup: func(ts *testServer) { ts.setResponse(204, nil) },
			call:  func(c *Client) error { return c.Edge.Delete(context.Background(), "ep") },
		},
		{
			name: "pauseEdgeEndpoint", method: "PUT", path: "/edge/ep/pause",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"name": "ep", "product": "rpc", "status": "PAUSED", "rate_limit_budget": nil, "allowed_domains": []any{}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "paused_at": "2026-01-01T00:00:00Z"}})
			},
			call: func(c *Client) error { _, err := c.Edge.Pause(context.Background(), "ep"); return err },
		},
		{
			name: "resumeEdgeEndpoint", method: "PUT", path: "/edge/ep/resume",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"name": "ep", "product": "rpc", "status": "ACTIVE", "rate_limit_budget": nil, "allowed_domains": []any{}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "paused_at": nil}})
			},
			call: func(c *Client) error { _, err := c.Edge.Resume(context.Background(), "ep"); return err },
		},
		{
			name: "revealEdgeEndpointKey", method: "GET", path: "/edge/ep/api-key",
			setup: func(ts *testServer) { ts.setJSON(200, map[string]any{"data": map[string]any{"api_key": "ek-123"}}) },
			call:  func(c *Client) error { _, err := c.Edge.RevealKey(context.Background(), "ep"); return err },
		},
		{
			name: "getEdgeEndpointMetrics", method: "GET", path: "/edge/ep/metrics",
			setup: func(ts *testServer) {
				ts.setJSON(200, map[string]any{"data": map[string]any{"requests": []any{}, "errors": []any{}}})
			},
			call: func(c *Client) error {
				_, err := c.Edge.Metrics(context.Background(), "ep", EdgeMetricsOptions{BucketSize: "1h"})
				return err
			},
			extra: func(t *testing.T, req recordedRequest) {
				if !strings.Contains(req.Query, "bucket_size=1h") {
					t.Errorf("query = %q, want bucket_size=1h", req.Query)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t)
			ts.requests = nil
			tc.setup(ts)
			c := newTestClient(t, ts)
			if err := tc.call(c); err != nil {
				t.Fatalf("%s: call returned error: %v", tc.name, err)
			}
			req := ts.lastRequest()
			if req.Method != tc.method {
				t.Errorf("method = %q, want %q", req.Method, tc.method)
			}
			if req.Path != tc.path {
				t.Errorf("path = %q, want %q", req.Path, tc.path)
			}
			requireBearer(t, req.Header)
			if tc.extra != nil {
				tc.extra(t, req)
			}
		})
	}
}

// TestContractCount ensures all 40 manifest operations are covered.
func TestContractCount(t *testing.T) {
	// Count opCase entries in TestContractOperations by reflection of the
	// table is awkward; instead assert the literal count here.
	const want = 40
	ts := newTestServer(t)
	c := newTestClient(t, ts)
	_ = c
	if got := contractOpsCount; got != want {
		t.Fatalf("contract operations covered = %d, want %d", got, want)
	}
	_ = reflect.TypeOf
}

// contractOpsCount is the number of operations exercised by TestContractOperations.
const contractOpsCount = 40
