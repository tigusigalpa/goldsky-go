package goldsky

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tigusigalpa/goldsky-go/internal/clock"
)

// ErrAPITokenRequired is returned when a REST control-plane or private
// GraphQL operation is attempted without a project API token.
var ErrAPITokenRequired = errors.New("goldsky: REST project API token is required")

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
	return newClient(apiToken, true, options...)
}

// NewDataClient creates a client for public GraphQL and Edge RPC calls without
// requiring a REST project token. REST control-plane and private GraphQL calls
// return ErrAPITokenRequired without sending a request.
func NewDataClient(options ...Option) (*Client, error) {
	return newClient("", false, options...)
}

func newClient(apiToken string, requireToken bool, options ...Option) (*Client, error) {
	apiToken = strings.TrimSpace(apiToken)
	if requireToken && apiToken == "" {
		return nil, ErrAPITokenRequired
	}

	cfg := defaultConfig()
	for _, opt := range options {
		if opt == nil {
			return nil, errors.New("goldsky: nil client option")
		}
		opt(&cfg)
	}
	if err := validateBaseURL(cfg.baseURL, "REST base URL"); err != nil {
		return nil, err
	}
	if err := validateBaseURL(cfg.edgeBaseURL, "Edge base URL"); err != nil {
		return nil, err
	}
	if cfg.httpClient == nil {
		cfg.httpClient = &http.Client{Timeout: 60 * time.Second}
	} else {
		clone := *cfg.httpClient
		cfg.httpClient = &clone
	}
	if cfg.httpTimeout != nil {
		if *cfg.httpTimeout < 0 {
			return nil, fmt.Errorf("goldsky: timeout must not be negative: %s", *cfg.httpTimeout)
		}
		cfg.httpClient.Timeout = *cfg.httpTimeout
	}
	if cfg.clock == nil {
		cfg.clock = clock.SystemClock{}
	}
	if cfg.sleeper == nil {
		cfg.sleeper = clock.SystemSleeper{}
	}
	if cfg.retry.MaxAttempts == 0 {
		cfg.retry.MaxAttempts = DefaultRetryPolicy().MaxAttempts
	} else if cfg.retry.MaxAttempts < 0 {
		return nil, fmt.Errorf("goldsky: retry max attempts must not be negative: %d", cfg.retry.MaxAttempts)
	}
	if cfg.retry.InitialBackoff < 0 || cfg.retry.MaxBackoff < 0 {
		return nil, errors.New("goldsky: retry backoff durations must not be negative")
	}
	if cfg.retry.MaxAttempts < 1 {
		cfg.retry.MaxAttempts = 1
	}
	if cfg.maxResponseBodyBytes <= 0 {
		return nil, fmt.Errorf("goldsky: max response body bytes must be positive, got %d", cfg.maxResponseBodyBytes)
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
	c.GraphQL = &GraphQLService{client: c, baseURL: DefaultGraphQLBaseURL}
	c.RPC = &RPCService{client: c, edgeAPIKey: strings.TrimSpace(cfg.edgeAPIKey), baseURL: cfg.edgeBaseURL}
	return c, nil
}

// BaseURL returns the REST control-plane base URL in use.
func (c *Client) BaseURL() string { return c.cfg.baseURL }

// UserAgent returns the User-Agent header sent on REST requests.
func (c *Client) UserAgent() string { return c.cfg.userAgent }

// SetEdgeAPIKey changes the Edge endpoint API key used by RPC calls. It is safe
// to call while other goroutines use the client. The Edge key is a separate
// secret from the REST token used by private GraphQL calls.
func (c *Client) SetEdgeAPIKey(key string) {
	c.RPC.setAPIKey(strings.TrimSpace(key))
}

func validateBaseURL(raw, label string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("goldsky: invalid %s %q", label, raw)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("goldsky: invalid %s scheme %q", label, u.Scheme)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("goldsky: %s must not contain a query or fragment", label)
	}
	return nil
}
