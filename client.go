package goldsky

import (
	"errors"
	"strings"

	"github.com/tigusigalpa/goldsky-go/internal/clock"
)

// Client is the top-level Goldsky client. It exposes grouped service clients
// for the REST control plane and the GraphQL and Edge RPC data planes.
//
// Create one client and reuse it for the lifetime of your application. The
// REST project API token and the Edge endpoint API key are distinct secrets;
// both are kept unexported and never appear in error messages or logs.
type Client struct {
	apiToken string
	cfg      config

	// Pipelines manages Turbo Pipelines.
	Pipelines *PipelineService
	// Subgraphs manages Subgraphs and their versions, tags, and deployments.
	Subgraphs *SubgraphService
	// Webhooks manages Subgraph entity webhooks.
	Webhooks *WebhookService
	// Edge manages Edge endpoints and their lifecycle.
	Edge *EdgeService
	// Catalogs lists supported chains, networks, and Edge Data sources.
	Catalogs *CatalogService
	// GraphQL queries Subgraph GraphQL data-plane endpoints.
	GraphQL *GraphQLService
	// RPC calls the Edge HTTPS JSON-RPC data plane.
	RPC *RPCService
}

// NewClient creates a Goldsky client with the supplied REST project API token
// and options. It performs no network calls. The token is scoped to a single
// Goldsky project and is sent as a Bearer header; it is never logged.
func NewClient(apiToken string, options ...Option) (*Client, error) {
	apiToken = strings.TrimSpace(apiToken)
	if apiToken == "" {
		return nil, errors.New("goldsky: API token is required")
	}

	cfg := defaultConfig()
	for _, opt := range options {
		opt(&cfg)
	}
	if cfg.httpClient == nil {
		cfg.httpClient = defaultConfig().httpClient
	}
	if cfg.clock == nil {
		cfg.clock = clock.SystemClock{}
	}
	if cfg.sleeper == nil {
		cfg.sleeper = clock.SystemSleeper{}
	}
	if cfg.retry.MaxAttempts < 1 {
		cfg.retry.MaxAttempts = 1
	}

	c := &Client{
		apiToken: apiToken,
		cfg:      cfg,
	}
	c.Pipelines = &PipelineService{client: c}
	c.Subgraphs = &SubgraphService{client: c}
	c.Webhooks = &WebhookService{client: c}
	c.Edge = &EdgeService{client: c}
	c.Catalogs = &CatalogService{client: c}
	c.GraphQL = &GraphQLService{client: c, edgeAPIKey: cfg.edgeAPIKey, baseURL: DefaultGraphQLBaseURL}
	c.RPC = &RPCService{client: c, edgeAPIKey: cfg.edgeAPIKey, baseURL: cfg.edgeBaseURL}
	return c, nil
}

// BaseURL returns the REST control-plane base URL in use.
func (c *Client) BaseURL() string { return c.cfg.baseURL }

// UserAgent returns the User-Agent header sent on REST requests.
func (c *Client) UserAgent() string { return c.cfg.userAgent }

// SetEdgeAPIKey changes the Edge endpoint API key used by the RPC and GraphQL
// private helpers. The Edge key is a separate secret from the REST token.
func (c *Client) SetEdgeAPIKey(key string) {
	c.RPC.edgeAPIKey = key
	c.GraphQL.edgeAPIKey = key
}
