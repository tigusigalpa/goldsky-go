# Goldsky Go SDK

[![CI](https://github.com/tigusigalpa/goldsky-go/actions/workflows/ci.yml/badge.svg)](https://github.com/tigusigalpa/goldsky-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/goldsky-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/goldsky-go)
[![Go version](https://img.shields.io/github/go-mod/go-version/tigusigalpa/goldsky-go)](go.mod)
[![License](https://img.shields.io/github/license/tigusigalpa/goldsky-go)](LICENSE)

A Go SDK for the [Goldsky](https://goldsky.com) REST control plane and the
Subgraph GraphQL and Edge RPC data planes. Built against Goldsky REST API
**v1.2.0** (40 operations). Zero third-party runtime dependencies.

> 📚 See the [Wiki](wiki/README.md) for guides, and [examples](examples/README.md)
> for runnable programs.

## Contents

- [Installation](#installation)
- [Client setup](#client-setup)
- [REST API](#rest-api)
- [Pagination](#pagination)
- [Deploy a subgraph](#deploy-a-subgraph)
- [GraphQL](#graphql)
- [Edge RPC](#edge-rpc)
- [Webhook verification](#webhook-verification)
- [Errors](#errors)
- [Retries](#retries)
- [Security](#security)

## Installation

Go 1.22 or later.

```bash
go get github.com/tigusigalpa/goldsky-go
```

```go
import goldsky "github.com/tigusigalpa/goldsky-go"
```

## Client setup

The REST project API token and the Edge endpoint API key are **distinct
secrets**. The REST token is sent as `Authorization: Bearer <token>`; the Edge
key is sent in the Edge RPC query string.

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

The constructor performs no network calls. Create one client and reuse it.

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

Services: `Pipelines`, `Subgraphs`, `Webhooks`, `Edge`, `Catalogs`. All 40 REST
operations are covered — see [docs/api-coverage.md](docs/api-coverage.md).

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
    Bundle: f, BundleFilename: "build.zip",
})
```

## GraphQL

```go
resp, err := client.GraphQL.QueryPrivate(ctx, projectID, "my-sub", "v1", goldsky.GraphQLRequest{
    Query: "{ _meta { block { number } } }",
})
if resp.HasErrors() {
    for _, e := range resp.Errors {
        fmt.Println(e.Message)
    }
}
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
fmt.Println(block)
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

## Errors

REST failures are RFC 9457 `application/problem+json`. Branch on the stable
`type` URI, not on prose.

```go
if p := goldsky.AsProblem(err); p != nil {
    if p.IsNotFound() { /* ... */ }
    if p.IsRateLimited() {
        secs, _ := p.RetryAfter()
        fmt.Println("retry after", secs, "seconds")
    }
    fmt.Println(p.Type) // stable URI
}
```

Validation failures (400) include a field `errors` array. Transport failures
(network, malformed JSON) are `*TransportError`.

## Retries

By default only safe reads (`GET`, `HEAD`, `OPTIONS`) are retried on transport
errors and 429/500/502/503/504, with capped exponential backoff plus jitter,
honouring `Retry-After`. Mutations are not retried automatically (no documented
idempotency keys). Opt in with `goldsky.WithRetryMutations()`.

## Security

- The REST token and Edge key are never logged or included in error messages.
- Webhook secrets and Edge API keys are one-time secrets returned to the caller
  only; store them immediately.
- See [docs/security.md](docs/security.md) and [wiki/security.md](wiki/security.md).

## Documentation

- [Wiki](wiki/README.md)
- [Examples](examples/README.md)
- [API coverage](docs/api-coverage.md)
- [Goldsky REST API overview](https://docs.goldsky.com/api-reference/overview)
- [Interactive API explorer](https://api.goldsky.com/api/v1/docs)
- [Error catalogue](https://api.goldsky.com/api/errors)

## License

[MIT](LICENSE) — Copyright (c) 2026 Igor Sazonov
