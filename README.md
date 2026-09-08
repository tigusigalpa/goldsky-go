# Goldsky Golang Client/SDK/Library

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

The SDK is built against Goldsky REST API **v1.2.0** and covers all 40 documented
operations. If you prefer learning from complete programs, start with the
[runnable examples](examples/README.md).

## Contents

- [Installation](#installation)
- [Quick start](#quick-start)
- [Client setup](#client-setup)
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
- [Examples and reference](#examples-and-reference)

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
  to `NewClient`; the constructor requires it even if you only plan to use RPC.
- The Edge endpoint API key authenticates Edge JSON-RPC calls. Pass it with
  `WithEdgeAPIKey` or rotate it later with the concurrency-safe `SetEdgeAPIKey`.

The project token is sent in `Authorization: Bearer <token>`. The Edge key is
sent only to the Edge endpoint as a query parameter.

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

The constructor performs no network calls. Options are validated up front,
and a custom `http.Client` is copied before a timeout override is applied.

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
extracted. `overwrite=1` is rejected by the server.

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

Public endpoints have a documented default rate limit of 50 requests per 10
seconds; the SDK does not perform aggressive hidden retries.

## Edge RPC

HTTPS JSON-RPC 2.0 only — no WebSockets or subscriptions.

```go
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
- Do not log complete Edge endpoint URLs: the API key is part of the query.
- See [docs/security.md](docs/security.md) for the full credential and retry
  guidance.

## Examples and reference

- [Runnable examples](examples/README.md) — small programs for pipelines,
  pagination, GraphQL, Edge RPC, webhooks, and error handling.
- [API coverage](docs/api-coverage.md)
- [Security notes](docs/security.md)
- [Upgrade guide](docs/upgrading.md)
- [Goldsky REST API overview](https://docs.goldsky.com/api-reference/overview)
- [Interactive API explorer](https://api.goldsky.com/api/v1/docs)
- [Error catalogue](https://api.goldsky.com/api/errors)

## License

[MIT](LICENSE) — Copyright (c) 2026 Igor Sazonov
