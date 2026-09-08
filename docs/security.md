# Security

## Two distinct secrets

The Goldsky SDK handles two separate secrets. Never confuse them.

| Secret | Used by | Where it goes | How it is obtained |
| --- | --- | --- | --- |
| REST project API token | REST control plane (`Client`) | `Authorization: Bearer <token>` header | Goldsky dashboard project settings |
| Edge endpoint API key | Edge RPC (`RPC`) | `X-ERPC-Secret-Token` header | `Edge.Create` (one-time) or `Edge.RevealKey` |

The REST token is scoped to a single project and is never part of a REST path.
The Edge key is per-endpoint. Goldsky also documents a `?secret=` query form,
but the SDK deliberately uses the header so the value does not enter URLs,
browser history, proxy request lines, or ordinary access logs.

## What the SDK never logs or leaks

- The REST API token is never included in error messages, `Error()` strings,
  problem details, or logs.
- The Edge API key is never included in URLs, error messages, RPC error strings,
  or SDK logs.
- Webhook create secrets and Edge create API keys are returned to the caller
  but never written to logs by the SDK.

## One-time secrets

- A webhook delivery secret is returned **once** in `CreateWebhookResponse`. It
  is not available from any other endpoint. Store it immediately.
- An Edge endpoint API key is returned in `CreateEdgeEndpointResponse` and can
  later be revealed again via `Edge.RevealKey`. Treat it as a secret regardless.

## Webhook verification

Goldsky deliveries carry the literal `goldsky-webhook-secret` header. Verify
it with `VerifyWebhookSecret` / `VerifyWebhookRequest`, which uses
`crypto/subtle.ConstantTimeCompare`. Goldsky does not document an HMAC signature
format, so the SDK performs a direct constant-time comparison only. An empty
expected secret is always rejected to prevent accidental acceptance of unset
secrets.

## Retry safety

Mutations are not retried by default because Goldsky does not document
idempotency keys. Automatic retry applies only to safe reads (`GET`, `HEAD`,
`OPTIONS`) on transport errors and 429/500/502/503/504. Opt in to mutation
retry with `WithRetryMutations()` only when you understand the risk. Streaming
subgraph deployments are never retried because their readers cannot be replayed
safely.

## Bounded diagnostic bodies

REST, GraphQL, and Edge RPC responses are buffered and limited to 16 MiB by
default. Use `WithMaxResponseBodyBytes` only when a legitimate response needs a
different cap. Non-2xx REST responses retain the bounded body in
`ProblemDetails.RawBody`; do not log it blindly because upstream services may
echo request data.

## RBAC

Read operations need the Viewer role; writes need Editor. See
<https://docs.goldsky.com/rbac>.

## Reporting vulnerabilities

Report security issues responsibly. Do not open a public issue for a suspected
vulnerability.
