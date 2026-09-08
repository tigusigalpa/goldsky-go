package goldsky

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/tigusigalpa/goldsky-go/internal/clock"
)

// DefaultBaseURL is the Goldsky REST control-plane base URL.
const DefaultBaseURL = "https://api.goldsky.com/api/v1"

// DefaultUserAgent is the default User-Agent header for REST requests.
const DefaultUserAgent = "goldsky-go/1.0.0"

// DefaultEdgeBaseURL is the Goldsky Edge RPC HTTPS JSON-RPC base URL.
const DefaultEdgeBaseURL = "https://edge.goldsky.com/standard/evm"

// DefaultGraphQLBaseURL is the Goldsky Subgraph GraphQL data-plane base URL.
const DefaultGraphQLBaseURL = "https://api.goldsky.com/api"

// RetryPolicy controls automatic retry of failed requests.
//
// By default only safe reads (GET, HEAD, OPTIONS) are retried on transport
// errors and the status codes 429, 500, 502, 503, and 504, using capped
// exponential backoff with jitter and honouring Retry-After. Mutations are not
// retried automatically because Goldsky does not document idempotency keys;
// set RetryMutations to opt in to unsafe mutation retry.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts including the first. A value
	// of 1 disables retry. Zero means use the default.
	MaxAttempts int
	// InitialBackoff is the first backoff delay. Zero means use the default.
	InitialBackoff time.Duration
	// MaxBackoff caps the backoff delay. Zero means use the default.
	MaxBackoff time.Duration
	// RetryMutations, when true, also retries non-safe methods. This is opt-in
	// and unsafe because mutations are not documented as idempotent.
	RetryMutations bool
}

// DefaultRetryPolicy returns the default retry policy: up to 3 attempts, 500 ms
// initial backoff, 30 s max backoff, no mutation retry.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     30 * time.Second,
	}
}

// config holds resolved client configuration after applying options.
type config struct {
	baseURL     string
	userAgent   string
	httpClient  *http.Client
	httpTimeout *time.Duration
	retry       RetryPolicy
	logger      *log.Logger
	clock       Clock
	sleeper     Sleeper
	edgeAPIKey  string
	edgeBaseURL string
}

// Option configures a Client.
type Option func(*config)

// Clock supplies the current time for Retry-After calculations. Most callers
// should use the system clock configured by default; the interface exists for
// deterministic tests.
type Clock interface {
	Now() time.Time
}

// Sleeper waits between retry attempts while respecting cancellation. Most
// callers should use the context-aware system sleeper configured by default.
type Sleeper interface {
	Sleep(context.Context, time.Duration) error
}

// WithBaseURL overrides the REST control-plane base URL.
func WithBaseURL(url string) Option {
	return func(c *config) { c.baseURL = url }
}

// WithHTTPClient supplies a custom *http.Client (transport, proxy, timeouts).
func WithHTTPClient(h *http.Client) Option {
	return func(c *config) { c.httpClient = h }
}

// WithTimeout sets the timeout used by the client. When combined with
// WithHTTPClient, the supplied client is shallow-cloned before its timeout is
// changed, so the caller's *http.Client is never mutated.
func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		c.httpTimeout = &d
	}
}

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *config) { c.userAgent = ua }
}

// WithRetryPolicy overrides the retry policy.
func WithRetryPolicy(p RetryPolicy) Option {
	return func(c *config) { c.retry = p }
}

// WithRetryMaxAttempts is a convenience option setting the retry attempt count.
func WithRetryMaxAttempts(n int) Option {
	return func(c *config) { c.retry.MaxAttempts = n }
}

// WithRetryMutations opts in to retrying replayable non-idempotent mutations.
// Streaming multipart deployments are never retried because their readers
// cannot be replayed safely. Unsafe.
func WithRetryMutations() Option {
	return func(c *config) { c.retry.RetryMutations = true }
}

// WithLogger sets the logger used for redacted diagnostic messages.
func WithLogger(l *log.Logger) Option {
	return func(c *config) { c.logger = l }
}

// WithClock injects a Clock for deterministic tests.
func WithClock(cl Clock) Option {
	return func(c *config) { c.clock = cl }
}

// WithSleeper injects a Sleeper for deterministic retry tests.
func WithSleeper(s Sleeper) Option {
	return func(c *config) { c.sleeper = s }
}

// WithEdgeAPIKey sets the default Edge endpoint API key used by the RPC client.
// The Edge key is a separate secret from the REST project Bearer token; it is
// never logged or included in error messages.
func WithEdgeAPIKey(key string) Option {
	return func(c *config) { c.edgeAPIKey = key }
}

// WithEdgeBaseURL overrides the Edge RPC base URL.
func WithEdgeBaseURL(url string) Option {
	return func(c *config) { c.edgeBaseURL = url }
}

func defaultConfig() config {
	return config{
		baseURL:     DefaultBaseURL,
		userAgent:   DefaultUserAgent,
		retry:       DefaultRetryPolicy(),
		logger:      log.New(ioDiscard(), "goldsky: ", 0),
		clock:       clock.SystemClock{},
		sleeper:     clock.SystemSleeper{},
		edgeBaseURL: DefaultEdgeBaseURL,
	}
}
