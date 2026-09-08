package goldsky

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *RPCError) Error() string {
	if e == nil {
		return "goldsky rpc: <nil error>"
	}
	return fmt.Sprintf("goldsky rpc error %d: %s", e.Code, e.Message)
}

// rpcRequest is a JSON-RPC 2.0 request.
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
	ID      int64  `json:"id"`
}

// RPCResponse is a JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCBatchCall is a single call within a batch. If Result is non-nil, the
// decoded result is written into it; use a *json.RawMessage to keep the raw
// result bytes.
type RPCBatchCall struct {
	Method string
	Params any
	Result any
}

// RPCService calls the Edge HTTPS JSON-RPC data plane.
//
// The Edge endpoint URL is
//
//	https://edge.goldsky.com/standard/evm/{chainId}?key={edgeAPIKey}
//
// The Edge API key is a separate secret from the REST project Bearer token and
// is carried in the query string; it is never logged or included in error
// messages. Goldsky documents HTTPS only; there is no WebSocket/subscription
// support.
type RPCService struct {
	client     *Client
	edgeAPIKey string
	baseURL    string
	idCounter  atomic.Int64
	mu         sync.RWMutex
}

// EndpointURL builds the Edge RPC URL for the given chain ID. The Edge API key
// is included as a query parameter.
func (s *RPCService) EndpointURL(chainID int64) string {
	q := url.Values{}
	q.Set("key", s.apiKey())
	return strings.TrimRight(s.baseURL, "/") + "/" + strconv.FormatInt(chainID, 10) + "?" + q.Encode()
}

func (s *RPCService) apiKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.edgeAPIKey
}

func (s *RPCService) setAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.edgeAPIKey = key
}

// nextRequestID returns a monotonically increasing JSON-RPC request ID.
func (s *RPCService) nextRequestID() int64 {
	return s.idCounter.Add(1)
}

// Call performs a single JSON-RPC call. If result is non-nil, the decoded
// result is written into it; pass a *json.RawMessage to keep raw bytes. A
// non-nil *RPCError is returned when the server reports a JSON-RPC error.
func (s *RPCService) Call(ctx context.Context, chainID int64, method string, params any, result any) error {
	if s.apiKey() == "" {
		return fmt.Errorf("goldsky rpc: Edge API key is required; set it with WithEdgeAPIKey or SetEdgeAPIKey")
	}
	if chainID <= 0 {
		return fmt.Errorf("goldsky rpc: chain ID must be positive, got %d", chainID)
	}
	if strings.TrimSpace(method) == "" {
		return fmt.Errorf("goldsky rpc: method is required")
	}
	req := rpcRequest{JSONRPC: "2.0", Method: method, Params: params, ID: s.nextRequestID()}
	resp, err := s.post(ctx, chainID, req)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return resp.Error
	}
	if result != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return &TransportError{Op: "rpc.Call", Err: fmt.Errorf("decode rpc result: %w", err)}
		}
	}
	return nil
}

// Batch performs a JSON-RPC batch call. Each call's Result pointer, when
// non-nil, is filled with the decoded result. The returned responses are
// matched to the calls by index. A non-nil error indicates a transport or
// decode failure; individual JSON-RPC errors are available on each response.
func (s *RPCService) Batch(ctx context.Context, chainID int64, calls []RPCBatchCall) ([]RPCResponse, error) {
	if s.apiKey() == "" {
		return nil, fmt.Errorf("goldsky rpc: Edge API key is required; set it with WithEdgeAPIKey or SetEdgeAPIKey")
	}
	if len(calls) == 0 {
		return nil, nil
	}
	if chainID <= 0 {
		return nil, fmt.Errorf("goldsky rpc: chain ID must be positive, got %d", chainID)
	}
	reqs := make([]rpcRequest, len(calls))
	for i, c := range calls {
		if strings.TrimSpace(c.Method) == "" {
			return nil, fmt.Errorf("goldsky rpc: method is required for batch call %d", i)
		}
		reqs[i] = rpcRequest{JSONRPC: "2.0", Method: c.Method, Params: c.Params, ID: s.nextRequestID()}
	}
	resps, err := s.postBatch(ctx, chainID, reqs)
	if err != nil {
		return nil, err
	}
	byID := make(map[int64]RPCResponse, len(resps))
	for _, r := range resps {
		if _, exists := byID[r.ID]; exists {
			return nil, &TransportError{Op: "rpc.Batch", Err: fmt.Errorf("duplicate response id %d", r.ID)}
		}
		byID[r.ID] = r
	}
	out := make([]RPCResponse, len(calls))
	for i, c := range calls {
		r, ok := byID[reqs[i].ID]
		if !ok {
			r = RPCResponse{JSONRPC: "2.0", ID: reqs[i].ID, Error: &RPCError{Code: -32603, Message: "no response for request id"}}
		}
		out[i] = r
		if c.Result != nil && len(r.Result) > 0 && r.Error == nil {
			if err := json.Unmarshal(r.Result, c.Result); err != nil {
				return out, &TransportError{Op: "rpc.Batch", Err: fmt.Errorf("decode rpc result: %w", err)}
			}
		}
	}
	return out, nil
}

// post sends a single JSON-RPC request and decodes the response.
func (s *RPCService) post(ctx context.Context, chainID int64, req rpcRequest) (RPCResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return RPCResponse{}, &TransportError{Op: "rpc", Err: err}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.EndpointURL(chainID), bytes.NewReader(body))
	if err != nil {
		return RPCResponse{}, &TransportError{Op: "rpc", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", s.client.cfg.userAgent)

	resp, err := s.client.cfg.httpClient.Do(httpReq)
	if err != nil {
		return RPCResponse{}, &TransportError{Op: "rpc", Err: err}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return RPCResponse{}, &TransportError{Op: "rpc", Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return RPCResponse{}, &ProblemDetails{
			Type:    "about:blank",
			Status:  resp.StatusCode,
			Headers: resp.Header.Clone(),
			RawBody: raw,
			Detail:  "Edge RPC endpoint returned non-2xx status",
		}
	}
	var out RPCResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return RPCResponse{}, &TransportError{Op: "rpc", Err: fmt.Errorf("decode rpc response: %w", err)}
	}
	if out.ID != req.ID {
		return RPCResponse{}, &TransportError{Op: "rpc", Err: fmt.Errorf("response id %d does not match request id %d", out.ID, req.ID)}
	}
	return out, nil
}

// postBatch sends a JSON-RPC batch request and decodes the responses.
func (s *RPCService) postBatch(ctx context.Context, chainID int64, reqs []rpcRequest) ([]RPCResponse, error) {
	body, err := json.Marshal(reqs)
	if err != nil {
		return nil, &TransportError{Op: "rpc", Err: err}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.EndpointURL(chainID), bytes.NewReader(body))
	if err != nil {
		return nil, &TransportError{Op: "rpc", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", s.client.cfg.userAgent)

	resp, err := s.client.cfg.httpClient.Do(httpReq)
	if err != nil {
		return nil, &TransportError{Op: "rpc", Err: err}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &TransportError{Op: "rpc", Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ProblemDetails{
			Type:    "about:blank",
			Status:  resp.StatusCode,
			Headers: resp.Header.Clone(),
			RawBody: raw,
			Detail:  "Edge RPC endpoint returned non-2xx status",
		}
	}
	var out []RPCResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, &TransportError{Op: "rpc", Err: fmt.Errorf("decode rpc batch response: %w", err)}
	}
	return out, nil
}
