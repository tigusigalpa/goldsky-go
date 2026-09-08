package goldsky

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

// EdgeProduct is the product type of an Edge endpoint.
type EdgeProduct string

const (
	EdgeProductRPC   EdgeProduct = "rpc"
	EdgeProductData  EdgeProduct = "data"
	EdgeProductBoost EdgeProduct = "boost"
)

// EdgeStatus is the lifecycle state of an Edge endpoint.
type EdgeStatus string

const (
	EdgeStatusActive EdgeStatus = "ACTIVE"
	EdgeStatusPaused EdgeStatus = "PAUSED"
)

// EdgeRateLimitBudget is a named rate-limit budget applied per API key across
// all networks. Known values are listed as constants; the wire value is
// preserved verbatim so future additions do not break decoding.
type EdgeRateLimitBudget string

const (
	EdgeTier6kUnlimitedPerIP   EdgeRateLimitBudget = "edge-tier-6krpm-total-unlimited-per-ip"
	EdgeTier60kUnlimitedPerIP  EdgeRateLimitBudget = "edge-tier-60krpm-total-unlimited-per-ip"
	EdgeTier180kUnlimitedPerIP EdgeRateLimitBudget = "edge-tier-180krpm-total-unlimited-per-ip"
	EdgeTier360kUnlimitedPerIP EdgeRateLimitBudget = "edge-tier-360krpm-total-unlimited-per-ip"
	EdgeTier600kUnlimitedPerIP EdgeRateLimitBudget = "edge-tier-600krpm-total-unlimited-per-ip"
	EdgeTier6k500PerIP         EdgeRateLimitBudget = "edge-tier-6krpm-total-500rpm-per-ip"
	EdgeTier60k500PerIP        EdgeRateLimitBudget = "edge-tier-60krpm-total-500rpm-per-ip"
	EdgeTier180k500PerIP       EdgeRateLimitBudget = "edge-tier-180krpm-total-500rpm-per-ip"
	EdgeTier360k500PerIP       EdgeRateLimitBudget = "edge-tier-360krpm-total-500rpm-per-ip"
	EdgeTier600k500PerIP       EdgeRateLimitBudget = "edge-tier-600krpm-total-500rpm-per-ip"
	EdgeTierUnlimited100PerIP  EdgeRateLimitBudget = "edge-tier-unlimited-total-100rpm-per-ip"
	EdgeTierUnlimited500PerIP  EdgeRateLimitBudget = "edge-tier-unlimited-total-500rpm-per-ip"
)

// EdgeEndpoint is an Edge endpoint resource.
type EdgeEndpoint struct {
	Name            string      `json:"name"`
	Product         EdgeProduct `json:"product"`
	Status          EdgeStatus  `json:"status"`
	RateLimitBudget *string     `json:"rate_limit_budget"`
	AllowedDomains  []string    `json:"allowed_domains"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	PausedAt        *time.Time  `json:"paused_at"`
}

// ListEdgeEndpointsOptions filters and pages the Edge endpoint list.
type ListEdgeEndpointsOptions struct {
	Product   string
	PageSize  int
	PageToken string
}

// CreateEdgeEndpointRequest creates an Edge endpoint.
type CreateEdgeEndpointRequest struct {
	Name            string               `json:"name"`
	Product         *EdgeProduct         `json:"product,omitempty"`
	RateLimitBudget *EdgeRateLimitBudget `json:"rate_limit_budget,omitempty"`
	AllowedDomains  []string             `json:"allowed_domains,omitempty"`
}

// CreateEdgeEndpointResponse is the create response. APIKey is a one-time
// secret shown here and via the reveal endpoint only; it is never logged.
type CreateEdgeEndpointResponse struct {
	Data struct {
		EdgeEndpoint
		APIKey string `json:"api_key"`
	} `json:"data"`
	Warnings []string `json:"warnings,omitempty"`
}

// UpdateEdgeEndpointRequest updates an Edge endpoint. Domain changes are
// applied before rate-limit changes and the update is not transactional.
type UpdateEdgeEndpointRequest struct {
	RateLimitBudget *EdgeRateLimitBudget `json:"rate_limit_budget,omitempty"`
	AllowedDomains  []string             `json:"allowed_domains,omitempty"`
}

// EdgeNetwork is a supported Edge network.
type EdgeNetwork struct {
	ChainID     *json.Number `json:"chain_id"`
	Name        string       `json:"name"`
	NetworkName string       `json:"network_name"`
	ChainName   string       `json:"chain_name"`
	LogoURL     string       `json:"logo_url"`
}

// EdgeNetworksResponse lists supported Edge networks.
type EdgeNetworksResponse struct {
	Data []EdgeNetwork `json:"data"`
}

// EdgeSource is an Edge Data source.
type EdgeSource struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ProviderID   string `json:"provider_id"`
	ProviderName string `json:"provider_name"`
	LogoURL      string `json:"logo_url"`
	Endpoint     string `json:"endpoint"`
}

// EdgeSourcesResponse lists Edge Data sources.
type EdgeSourcesResponse struct {
	Data []EdgeSource `json:"data"`
}

// EdgeMetricsOptions filters Edge endpoint metrics.
type EdgeMetricsOptions struct {
	From       time.Time
	To         time.Time
	BucketSize string
}

// EdgeMetricsResponse is the Edge metrics endpoint envelope.
type EdgeMetricsResponse struct {
	Data EdgeMetricsData `json:"data"`
}

// RevealEdgeKeyResponse reveals an Edge endpoint API key.
type RevealEdgeKeyResponse struct {
	Data struct {
		APIKey string `json:"api_key"`
	} `json:"data"`
}

// EdgeService manages Edge endpoints and their lifecycle.
type EdgeService struct {
	client *Client
}

// List lists a single page of Edge endpoints. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/listEdgeEndpoints
func (s *EdgeService) List(ctx context.Context, opts ListEdgeEndpointsOptions) (Page[EdgeEndpoint], error) {
	if err := validatePageSize(opts.PageSize); err != nil {
		return Page[EdgeEndpoint]{}, err
	}
	q := make(url.Values)
	if opts.Product != "" {
		q.Set("product", opts.Product)
	}
	if opts.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(opts.PageSize))
	}
	if opts.PageToken != "" {
		q.Set("page_token", opts.PageToken)
	}
	resp, err := s.client.do(ctx, "GET", []string{"edge"}, requestOptions{query: q})
	if err != nil {
		return Page[EdgeEndpoint]{}, err
	}
	var page Page[EdgeEndpoint]
	if err := decodeJSON(resp.body, &page); err != nil {
		return Page[EdgeEndpoint]{}, &TransportError{Op: "listEdgeEndpoints", Err: err}
	}
	return page, nil
}

// EdgePager iterates Edge endpoint pages, cancellation-aware.
type EdgePager struct {
	p *listPager[EdgeEndpoint]
}

// NewEdgePager returns a pager over Edge endpoints starting at opts.PageToken.
func (s *EdgeService) NewEdgePager(opts ListEdgeEndpointsOptions) *EdgePager {
	p := &listPager[EdgeEndpoint]{
		client:   s.client,
		method:   "GET",
		segments: []string{"edge"},
		pageSize: opts.PageSize,
		token:    opts.PageToken,
		first:    true,
	}
	if opts.Product != "" {
		p.queryHook = func(q url.Values) { q.Set("product", opts.Product) }
	}
	return &EdgePager{p: p}
}

// NextPage fetches the next page.
func (p *EdgePager) NextPage(ctx context.Context) (Page[EdgeEndpoint], error) {
	return p.p.nextPage(ctx)
}

// Create creates an Edge endpoint and returns the one-time API key. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/createEdgeEndpoint
func (s *EdgeService) Create(ctx context.Context, req CreateEdgeEndpointRequest) (CreateEdgeEndpointResponse, error) {
	resp, err := s.client.do(ctx, "POST", []string{"edge"}, requestOptions{jsonBody: req})
	if err != nil {
		return CreateEdgeEndpointResponse{}, err
	}
	var out CreateEdgeEndpointResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return CreateEdgeEndpointResponse{}, &TransportError{Op: "createEdgeEndpoint", Err: err}
	}
	return out, nil
}

// Get fetches an Edge endpoint. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/getEdgeEndpoint
func (s *EdgeService) Get(ctx context.Context, name string) (EdgeEndpoint, error) {
	resp, err := s.client.do(ctx, "GET", []string{"edge", name}, requestOptions{})
	if err != nil {
		return EdgeEndpoint{}, err
	}
	var out struct {
		Data EdgeEndpoint `json:"data"`
	}
	if err := decodeJSON(resp.body, &out); err != nil {
		return EdgeEndpoint{}, &TransportError{Op: "getEdgeEndpoint", Err: err}
	}
	return out.Data, nil
}

// Update updates an Edge endpoint. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/updateEdgeEndpoint
func (s *EdgeService) Update(ctx context.Context, name string, req UpdateEdgeEndpointRequest) (EdgeEndpoint, error) {
	resp, err := s.client.do(ctx, "PATCH", []string{"edge", name}, requestOptions{jsonBody: req})
	if err != nil {
		return EdgeEndpoint{}, err
	}
	var out struct {
		Data EdgeEndpoint `json:"data"`
	}
	if err := decodeJSON(resp.body, &out); err != nil {
		return EdgeEndpoint{}, &TransportError{Op: "updateEdgeEndpoint", Err: err}
	}
	return out.Data, nil
}

// Delete deletes an Edge endpoint. Returns nil on 204. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/deleteEdgeEndpoint
func (s *EdgeService) Delete(ctx context.Context, name string) error {
	_, err := s.client.do(ctx, "DELETE", []string{"edge", name}, requestOptions{})
	return err
}

// Pause pauses an Edge endpoint. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20Lifecycle/operation/pauseEdgeEndpoint
func (s *EdgeService) Pause(ctx context.Context, name string) (EdgeEndpoint, error) {
	resp, err := s.client.do(ctx, "PUT", []string{"edge", name, "pause"}, requestOptions{})
	if err != nil {
		return EdgeEndpoint{}, err
	}
	var out struct {
		Data EdgeEndpoint `json:"data"`
	}
	if err := decodeJSON(resp.body, &out); err != nil {
		return EdgeEndpoint{}, &TransportError{Op: "pauseEdgeEndpoint", Err: err}
	}
	return out.Data, nil
}

// Resume resumes a paused Edge endpoint. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20Lifecycle/operation/resumeEdgeEndpoint
func (s *EdgeService) Resume(ctx context.Context, name string) (EdgeEndpoint, error) {
	resp, err := s.client.do(ctx, "PUT", []string{"edge", name, "resume"}, requestOptions{})
	if err != nil {
		return EdgeEndpoint{}, err
	}
	var out struct {
		Data EdgeEndpoint `json:"data"`
	}
	if err := decodeJSON(resp.body, &out); err != nil {
		return EdgeEndpoint{}, &TransportError{Op: "resumeEdgeEndpoint", Err: err}
	}
	return out.Data, nil
}

// RevealKey reveals the Edge endpoint API key. The key is a separate secret
// from the REST project token and is never logged. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20API%20Keys/operation/revealEdgeEndpointKey
func (s *EdgeService) RevealKey(ctx context.Context, name string) (RevealEdgeKeyResponse, error) {
	resp, err := s.client.do(ctx, "GET", []string{"edge", name, "api-key"}, requestOptions{})
	if err != nil {
		return RevealEdgeKeyResponse{}, err
	}
	var out RevealEdgeKeyResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return RevealEdgeKeyResponse{}, &TransportError{Op: "revealEdgeEndpointKey", Err: err}
	}
	return out, nil
}

// Metrics fetches Edge endpoint metrics. See
// https://api.goldsky.com/api/v1/docs#tag/Edge%20Metrics/operation/getEdgeEndpointMetrics
func (s *EdgeService) Metrics(ctx context.Context, name string, opts EdgeMetricsOptions) (EdgeMetricsResponse, error) {
	q := make(url.Values)
	if !opts.From.IsZero() {
		q.Set("from", opts.From.Format(time.RFC3339))
	}
	if !opts.To.IsZero() {
		q.Set("to", opts.To.Format(time.RFC3339))
	}
	if opts.BucketSize != "" {
		q.Set("bucket_size", opts.BucketSize)
	}
	resp, err := s.client.do(ctx, "GET", []string{"edge", name, "metrics"}, requestOptions{query: q})
	if err != nil {
		return EdgeMetricsResponse{}, err
	}
	var out EdgeMetricsResponse
	if err := decodeJSON(resp.body, &out); err != nil {
		return EdgeMetricsResponse{}, &TransportError{Op: "getEdgeEndpointMetrics", Err: err}
	}
	return out, nil
}
