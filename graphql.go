package goldsky

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GraphQLRequest is the standard GraphQL request envelope.
type GraphQLRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

// GraphQLError is a single GraphQL error. Message is the stable field; the
// complete error object is retained in Raw for forward compatibility.
type GraphQLError struct {
	Message string          `json:"message"`
	Raw     json.RawMessage `json:"-"`
}

// UnmarshalJSON captures the message and retains the full error object.
func (e *GraphQLError) UnmarshalJSON(b []byte) error {
	e.Raw = append(e.Raw[:0], b...)
	var m struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	e.Message = m.Message
	return nil
}

// GraphQLResponse is the GraphQL data-plane response.
type GraphQLResponse struct {
	Data       json.RawMessage `json:"data,omitempty"`
	Errors     []GraphQLError  `json:"errors,omitempty"`
	Extensions json.RawMessage `json:"extensions,omitempty"`
	Status     int             `json:"-"`
	Header     http.Header     `json:"-"`
}

// HasErrors reports whether the response contains GraphQL errors.
func (r *GraphQLResponse) HasErrors() bool { return r != nil && len(r.Errors) > 0 }

// GraphQLService queries Subgraph GraphQL data-plane endpoints.
//
// Public endpoints use
//
//	https://api.goldsky.com/api/public/{project_id}/subgraphs/{subgraph_name}/{version_or_tag}/gn
//
// and have a documented default rate limit of 50 requests per 10 seconds.
// Private endpoints use https://api.goldsky.com/api/private/.../gn and require
// the project Bearer token. The SDK does not perform aggressive hidden retries
// against these endpoints.
type GraphQLService struct {
	client  *Client
	baseURL string
}

// PublicURL builds the public Subgraph GraphQL endpoint URL.
func (s *GraphQLService) PublicURL(projectID, subgraphName, versionOrTag string) string {
	return s.endpointURL("public", projectID, subgraphName, versionOrTag)
}

// PrivateURL builds the private Subgraph GraphQL endpoint URL.
func (s *GraphQLService) PrivateURL(projectID, subgraphName, versionOrTag string) string {
	return s.endpointURL("private", projectID, subgraphName, versionOrTag)
}

func (s *GraphQLService) endpointURL(scope, projectID, subgraphName, versionOrTag string) string {
	return fmt.Sprintf("%s/%s/%s/subgraphs/%s/%s/gn",
		strings.TrimRight(s.baseURL, "/"),
		scope,
		url.PathEscape(projectID),
		url.PathEscape(subgraphName),
		url.PathEscape(versionOrTag),
	)
}

// Query sends a GraphQL request to an arbitrary endpoint URL. The caller is
// responsible for using PublicURL or PrivateURL. When auth is true, the project
// Bearer token is sent; the token is never logged.
func (s *GraphQLService) Query(ctx context.Context, endpoint string, req GraphQLRequest, auth bool) (GraphQLResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return GraphQLResponse{}, &TransportError{Op: "graphql", Err: err}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return GraphQLResponse{}, &TransportError{Op: "graphql", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if auth {
		httpReq.Header.Set("Authorization", "Bearer "+s.client.apiToken)
	}
	httpReq.Header.Set("User-Agent", s.client.cfg.userAgent)

	resp, err := s.client.cfg.httpClient.Do(httpReq)
	if err != nil {
		return GraphQLResponse{}, &TransportError{Op: "graphql", Err: err}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return GraphQLResponse{}, &TransportError{Op: "graphql", Err: err}
	}
	out := GraphQLResponse{Status: resp.StatusCode, Header: resp.Header.Clone()}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Preserve a valid GraphQL error envelope when one is present, but keep
		// the HTTP failure authoritative even when a proxy returns HTML/text.
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &out)
		}
		return out, &ProblemDetails{
			Type:    "about:blank",
			Status:  resp.StatusCode,
			Headers: resp.Header.Clone(),
			RawBody: raw,
			Detail:  "GraphQL endpoint returned non-2xx status",
		}
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return out, &TransportError{Op: "graphql", Err: fmt.Errorf("decode graphql response: %w", err)}
		}
	}
	return out, nil
}

// QueryPublic queries a public Subgraph GraphQL endpoint.
func (s *GraphQLService) QueryPublic(ctx context.Context, projectID, subgraphName, versionOrTag string, req GraphQLRequest) (GraphQLResponse, error) {
	return s.Query(ctx, s.PublicURL(projectID, subgraphName, versionOrTag), req, false)
}

// QueryPrivate queries a private Subgraph GraphQL endpoint using the project
// Bearer token.
func (s *GraphQLService) QueryPrivate(ctx context.Context, projectID, subgraphName, versionOrTag string, req GraphQLRequest) (GraphQLResponse, error) {
	return s.Query(ctx, s.PrivateURL(projectID, subgraphName, versionOrTag), req, true)
}
