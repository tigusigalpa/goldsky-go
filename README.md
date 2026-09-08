# Goldsky Go SDK

![Goldsky API Golang Client/SDK/Library](https://i.postimg.cc/T3Gmm6ZH/goldsky-api-golang-hero.jpg)

[![CI](https://github.com/tigusigalpa/goldsky-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/goldsky-go/actions/workflows/ci.yml)
[![Tests](https://github.com/tigusigalpa/goldsky-go/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/goldsky-go/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![CodeQL](https://github.com/tigusigalpa/goldsky-go/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/goldsky-go/actions/workflows/codeql.yml)
[![Codecov](https://codecov.io/gh/tigusigalpa/goldsky-go/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/goldsky-go)
[![GitHub Release](https://img.shields.io/github/v/release/tigusigalpa/goldsky-go?style=flat-square)](https://github.com/tigusigalpa/goldsky-go/releases)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://pkg.go.dev/github.com/tigusigalpa/goldsky-go)

A small, idiomatic Go client for [Goldsky](https://goldsky.com). It covers the
REST control plane, Subgraph GraphQL endpoints, and Edge JSON-RPC without
pulling any third-party runtime dependencies into your application.

This is a community-maintained SDK, not an official Goldsky package. The public
API follows ordinary Go conventions: explicit contexts, typed request and
response models where the upstream contract is stable, and raw JSON where
Goldsky intentionally leaves a schema open.

The SDK is built against Goldsky REST API **v1.2.0** and covers all 40 documented
operations. If you prefer learning from complete programs, start with the
[runnable examples](examples/README.md).

## Contents

- [Installation](#installation)
- [Quick start](#quick-start)
- [Client setup](#client-setup)
- [Configuration](#configuration)
- [What is covered](#what-is-covered)
- [REST API](#rest-api)
- [Pagination](#pagination)
- [Deploy a subgraph](#deploy-a-subgraph)
- [GraphQL](#graphql)
- [Edge RPC](#edge-rpc)
- [Webhook verification](#webhook-verification)
- [Errors](#errors)
- [Retries](#retries)
- [Security](#security)
- [Best practices](#best-practices)
- [Known limits](#known-limits)
- [FAQ](#faq)
- [Examples and reference](#examples-and-reference)
- [Development](#development)

## Installation

Go 1.22 or later.

```bash
go get github.com/tigusigalpa/goldsky-go
```

```go
import goldsky "github.com/tigusigalpa/goldsky-go"
```

## Quick start

Create a project API token in the Goldsky dashboard, export it as
`GOLDSKY_API_KEY`, and reuse one client throughout your application:

```go
client, err := goldsky.NewClient(
    os.Getenv("GOLDSKY_API_KEY"),
    goldsky.WithTimeout(30*time.Second),
)
if err != nil {
    log.Fatal(err)
}

ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
defer cancel()

page, err := client.Pipelines.List(ctx, goldsky.ListPipelinesOptions{PageSize: 25})
if err != nil {
    log.Fatal(err)
}
for _, pipeline := range page.Data {
    fmt.Printf("%-30s %s\n", pipeline.Name, pipeline.Status)
}
```

## Client setup

Goldsky uses two different credentials:

- The project API token authenticates REST and private GraphQL calls. Pass it
  to `NewClient`.
- The Edge endpoint API key authenticates Edge JSON-RPC calls. Pass it with
  `WithEdgeAPIKey` or rotate it later with the concurrency-safe `SetEdgeAPIKey`.

The project token is sent in `Authorization: Bearer <token>`. The Edge key is
sent in `X-ERPC-Secret-Token`, so it is not embedded in the request URL.

```go
client, err := goldsky.NewClient(
    os.Getenv("GOLDSKY_API_KEY"),
    goldsky.WithEdgeAPIKey(os.Getenv("GOLDSKY_EDGE_API_KEY")),
    goldsky.WithTimeout(30*time.Second),
)
if err != nil {
    log.Fatal(err)
}
```

If your process only needs public GraphQL or Edge RPC, it does not need a REST
token:

```go
client, err := goldsky.NewDataClient(
    goldsky.WithEdgeAPIKey(os.Getenv("GOLDSKY_EDGE_API_KEY")),
)
```

Calling a REST method or `QueryPrivate` on this client returns
`ErrAPITokenRequired` locally, before any request is sent. A public GraphQL
query needs neither credential; an Edge RPC call needs only the Edge key.

The constructor performs no network calls. Options are validated up front,
and a custom `http.Client` is copied before a timeout override is applied.

## Configuration

Most applications only need a token and a timeout. The other options are useful
for tests, proxies, private gateways, and operational tuning.

| Option | Purpose |
| --- | --- |
| `WithTimeout` | Sets the end-to-end `http.Client` timeout. The default is 60 seconds. |
| `WithHTTPClient` | Supplies a custom transport, proxy, TLS policy, or redirect policy. The client value is shallow-copied. |
| `WithRetryPolicy` / `WithRetryMaxAttempts` | Changes automatic retry behavior. |
| `WithMaxResponseBodyBytes` | Caps buffered REST, GraphQL, and RPC responses. The default is 16 MiB. |
| `WithUserAgent` | Replaces the default `goldsky-go/1.1.2` user agent. |
| `WithLogger` | Receives redacted retry diagnostics. |
| `WithBaseURL` / `WithEdgeBaseURL` | Points tests or compatible gateways at another base URL. |

`WithTimeout` and `WithHTTPClient` are order-independent. The supplied
`*http.Client` is not mutated, but its `Transport` is shared by the shallow
copy, matching the standard library's normal reuse model.

## What is covered

| Service | Use it for |
| --- | --- |
| `Pipelines` | Create, validate, inspect, pause, resume, restart, and delete Turbo Pipelines; read logs, state, and status. |
| `Subgraphs` | Deploy and manage versions and tags; read indexing logs and webhook entities. |
| `Webhooks` | Create, list, and delete entity webhooks. |
| `Edge` | Manage Edge endpoints, keys, lifecycle, and metrics. |
| `Catalogs` | Discover supported subgraph chains, Edge networks, and Edge Data sources. |
| `GraphQL` | Query public or private Subgraph GraphQL endpoints. |
| `RPC` | Make single or batch HTTPS JSON-RPC 2.0 calls through Goldsky Edge. |

The exact operation-to-method mapping lives in
[docs/api-coverage.md](docs/api-coverage.md).

## REST API

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

page, err := client.Pipelines.List(ctx, goldsky.ListPipelinesOptions{PageSize: 50})
if err != nil {
    log.Fatal(err)
}
for _, p := range page.Data {
    fmt.Println(p.Name, p.Status)
}

ep, err := client.Edge.Get(ctx, "my-endpoint")
if err != nil {
    log.Fatal(err)
}
fmt.Println(ep.Name, ep.Status)
```

Path components and query values are escaped by the SDK. Every call accepts a
`context.Context`, so cancellation and deadlines work the same way as in the
standard library.

## Pagination

Pipelines, subgraphs, and Edge endpoints page with `page_size`/`page_token`.
A page can hold fewer than `page_size` items and still have a next page, so
completion is inferred from `next_page_token` alone.

```go
pager := client.Subgraphs.NewSubgraphPager(goldsky.ListSubgraphsOptions{PageSize: 100})
for {
    page, err := pager.NextPage(ctx)
    if err != nil {
        return err
    }
    for _, s := range page.Data {
        fmt.Println(s.Name, s.Version, s.Health)
    }
    if !page.HasMore() {
        break
    }
}
```

Once a pager reaches the last page, later `NextPage` calls return an empty page
without making another HTTP request. Invalid page sizes are rejected locally.

## Deploy a subgraph

`Deploy` streams the bundle as `multipart/form-data` without buffering the
whole zip in memory. The server accepts up to 50 MB compressed / 100 MB
extracted. `overwrite=1` is rejected locally before the request is sent.

```go
f, err := os.Open("build.zip")
if err != nil {
    log.Fatal(err)
}
defer f.Close()
s, err := client.Subgraphs.Deploy(ctx, "my-sub", "v1", goldsky.DeploySubgraphOptions{
    Bundle:         f,
    BundleFilename: "build.zip",
})
if err != nil {
    log.Fatal(err)
}
fmt.Println("deployed", s.Name, s.Version)
```

Streaming uploads are never retried automatically because an arbitrary
`io.Reader` cannot be replayed safely. Open a fresh file and call `Deploy` again
only after you have checked whether the first request took effect.

## GraphQL

Use `QueryPrivate` for a private endpoint and `QueryPublic` for a public one.
Both return the HTTP status and response headers alongside raw `data`,
`errors`, and `extensions` fields.

```go
resp, err := client.GraphQL.QueryPrivate(ctx, projectID, "my-sub", "v1", goldsky.GraphQLRequest{
    Query: "{ _meta { block { number } } }",
})
if err != nil {
    log.Fatal(err)
}
if resp.HasErrors() {
    for _, e := range resp.Errors {
        fmt.Println(e.Message)
    }
    return
}
fmt.Println(string(resp.Data))
```

Decode `Data` into the shape owned by your application:

```go
var data struct {
    Meta struct {
        Block struct {
            Number int `json:"number"`
        } `json:"block"`
    } `json:"_meta"`
}
if err := json.Unmarshal(resp.Data, &data); err != nil {
    return err
}
```

For public data access, construct a tokenless client and call `QueryPublic`:

```go
client, err := goldsky.NewDataClient()
if err != nil {
    log.Fatal(err)
}
resp, err := client.GraphQL.QueryPublic(ctx, projectID, "my-sub", "prod", request)
```

Public endpoints have a documented default rate limit of 50 requests per 10
seconds; the SDK does not perform aggressive hidden retries.

## Edge RPC

HTTPS JSON-RPC 2.0 only — no WebSockets or subscriptions.

```go
client, err := goldsky.NewDataClient(
    goldsky.WithEdgeAPIKey(os.Getenv("GOLDSKY_EDGE_API_KEY")),
)
if err != nil {
    log.Fatal(err)
}

var block string
if err := client.RPC.Call(ctx, 1, "eth_blockNumber", nil, &block); err != nil {
    log.Fatal(err)
}
n, err := strconv.ParseUint(strings.TrimPrefix(block, "0x"), 16, 64)
if err != nil {
    log.Fatalf("unexpected block number %q: %v", block, err)
}
fmt.Println("latest Ethereum block:", n)
```

Batch responses can arrive in any order; `Batch` matches them back to the input
calls by JSON-RPC request ID:

```go
var block, chain string
responses, err := client.RPC.Batch(ctx, 1, []goldsky.RPCBatchCall{
    {Method: "eth_blockNumber", Result: &block},
    {Method: "eth_chainId", Result: &chain},
})
if err != nil {
    log.Fatal(err)
}
for _, response := range responses {
    if response.Error != nil {
        fmt.Println("RPC error:", response.Error)
    }
}
```

A successful transport does not mean every batch item succeeded. Always inspect
each `RPCResponse.Error`. The client rejects malformed protocol envelopes,
unknown or duplicate IDs, and responses containing both `result` and `error`.

## Webhook verification

Goldsky deliveries carry the literal `goldsky-webhook-secret` header. Verify
in constant time; no HMAC format is documented.

```go
if !goldsky.VerifyWebhookRequest(r, storedSecret) {
    http.Error(w, "invalid secret", http.StatusUnauthorized)
    return
}
```

Read the request body only after verification, and return a `2xx` status once
the event has been accepted. The header contains a shared secret, not an HMAC
signature, so protect it like any other credential.

## Errors

REST failures are returned as RFC 9457 `application/problem+json`. Use
`AsProblem` to inspect them and branch on the stable `type` URI or a status
helper, not on human-readable prose:

```go
if problem := goldsky.AsProblem(err); problem != nil {
    if problem.IsNotFound() {
        fmt.Println("resource does not exist")
        return
    }
    if problem.IsRateLimited() {
        if seconds, ok := problem.RetryAfter(); ok {
            fmt.Println("retry after", seconds, "seconds")
        }
    }
    fmt.Println(problem.Type) // stable URI
}
```

Validation failures (400) include a field `errors` array. Transport failures
(network, cancellation, or malformed JSON) are `*TransportError` and support
`errors.As`. GraphQL errors remain in `GraphQLResponse.Errors`; JSON-RPC errors
are returned as `*RPCError` for single calls and per response for batches.

Non-2xx responses retain a bounded copy of the server body in
`ProblemDetails.RawBody`. Treat it as diagnostic data: avoid logging it blindly,
because an upstream proxy or application could echo sensitive request content.

## Retries

By default only safe reads (`GET`, `HEAD`, `OPTIONS`) are retried on transport
errors and 429/500/502/503/504, with capped exponential backoff plus jitter,
honouring `Retry-After`. Mutations are not retried automatically (no documented
idempotency keys). Opt in with `goldsky.WithRetryMutations()` only when your
workflow can tolerate a duplicate mutation. Streaming deployments remain
single-attempt even after opting in.

To disable retries, set the total attempt count to one:

```go
client, err := goldsky.NewClient(
    os.Getenv("GOLDSKY_API_KEY"),
    goldsky.WithRetryMaxAttempts(1),
)
```

## Security

- The REST token and Edge key are never logged or included in error messages.
- Webhook secrets and Edge API keys are one-time secrets returned to the caller
  only; store them immediately.
- Edge RPC authentication uses `X-ERPC-Secret-Token`; the SDK never adds the
  key to an endpoint URL.
- Webhook verification compares a shared secret in constant time. It does not
  prove payload integrity with an HMAC signature because Goldsky does not
  document such a signature format.
- See [docs/security.md](docs/security.md) for the full credential and retry
  guidance.

## Best practices

- Create one `Client` and reuse it. `http.Client`, the services, and Edge key
  rotation are safe for concurrent use; individual pagers are not.
- Put a deadline on every operation. A client timeout is a safety net, while a
  request context lets each caller choose an appropriate budget.
- Keep mutation retries disabled unless duplicate creates or updates are safe
  in your workflow. Check the resource after an ambiguous timeout.
- Validate a pipeline before creating it, and use a distinct lowercase name for
  each example or integration test.
- Persist one-time webhook and Edge secrets before returning success from your
  provisioning workflow.
- For webhook handlers, authenticate first, cap the request body, enqueue work,
  and respond quickly. Make processing idempotent because deliveries may be
  retried.

## Known limits

- Edge support is HTTPS JSON-RPC only. WebSockets and subscriptions are outside
  the current API.
- GraphQL `data`, pipeline definitions, and pipeline state remain raw JSON or
  `map[string]any`; their schemas belong to user-defined subgraphs and pipeline
  configurations.
- Responses are buffered in memory and limited to 16 MiB by default. Increase
  the cap explicitly only when you expect larger payloads.
- The local OpenAPI fixture is a reviewed snapshot, not code generated at build
  time. Re-check it when Goldsky publishes a new REST API version.
- Streaming subgraph deployment bodies cannot be replayed and are never retried.

## FAQ

### Do I need both API keys?

No. REST and private GraphQL need the project API token. Edge RPC needs the Edge
endpoint key. Public GraphQL needs neither. Use `NewDataClient` when no REST
control-plane access is required.

### Why did a batch RPC call return no top-level error but contain failures?

JSON-RPC batch items fail independently. The top-level error reports transport
or malformed-envelope failures; inspect `response.Error` for each item.

### Can I retry a timed-out create or update?

Only after checking whether it already succeeded. Goldsky does not document
idempotency keys, so automatic mutation retries are deliberately opt-in.

### Why is GraphQL `Data` a `json.RawMessage`?

Every subgraph has its own schema. Keeping the SDK generic avoids brittle
generated types and lets your application decode exactly the fields it owns.

## Examples and reference

- [Runnable examples](examples/README.md) — small programs for pipelines,
  pagination, GraphQL, single and batch Edge RPC, webhooks, and error handling.
- [API coverage](docs/api-coverage.md)
- [Security notes](docs/security.md)
- [Upgrade guide](docs/upgrading.md)
- [Goldsky REST API overview](https://docs.goldsky.com/api-reference/overview)
- [Interactive API explorer](https://api.goldsky.com/api/v1/docs)
- [Error catalogue](https://api.goldsky.com/api/errors)

## Development

The repository has no runtime dependencies. Before opening a change, run:

```bash
go mod tidy
go build ./...
go vet ./...
go test -race ./...
```

Contract tests cover every REST operation in the checked-in OpenAPI snapshot;
edge-case tests cover pagination, retry behavior, JSON envelopes, multipart
streaming, secret handling, and the two data-plane clients.

## License

[MIT](LICENSE) — Copyright (c) 2026 Igor Sazonov
