# Goldsky Go SDK - Examples

Runnable, focused examples for the `goldsky-go` SDK. Each example lives in its
own subdirectory and can be run with `go run ./examples/<name>`.

## Prerequisites

Set the environment variables each example needs. All values are placeholders;
never commit real credentials.

```bash
export GOLDSKY_API_KEY="your_project_api_token"   # REST control plane
export GOLDSKY_EDGE_API_KEY="your_edge_api_key"    # Edge RPC (separate secret)
```

Then, from the repository root:

```bash
go mod tidy
```

## Examples

| # | Example | Prerequisites | Command | Behavior |
|---|---|---|---|---|
| 1 | `01-list-pipelines` | `GOLDSKY_API_KEY` | `go run ./examples/01-list-pipelines` | Constructs a client and lists one page of pipelines. |
| 2 | `02-paginate-subgraphs` | `GOLDSKY_API_KEY` | `go run ./examples/02-paginate-subgraphs` | Walks all subgraph pages with the cancellation-aware pager. |
| 3 | `03-validate-pipeline` | `GOLDSKY_API_KEY` | `go run ./examples/03-validate-pipeline` | Validates a pipeline definition without creating it. |
| 4 | `04-create-pipeline` | `GOLDSKY_API_KEY`, `GOLDSKY_RUN_MUTATIONS=1` | `go run ./examples/04-create-pipeline` | Creates a pipeline. **Mutation**: refuses to run unless `GOLDSKY_RUN_MUTATIONS=1`. |
| 5 | `05-graphql-query` | `GOLDSKY_API_KEY`, project/subgraph/version | `go run ./examples/05-graphql-query` | Queries a private Subgraph GraphQL endpoint. |
| 6 | `06-edge-rpc` | `GOLDSKY_EDGE_API_KEY` | `go run ./examples/06-edge-rpc` | Calls `eth_blockNumber` over Edge JSON-RPC. |
| 7 | `07-verify-webhook` | none | `go run ./examples/07-verify-webhook` | Verifies a `goldsky-webhook-secret` header in constant time. |

## Notes

- Examples 1-3 and 5-7 are read-only by default.
- Example 4 is a **mutation** and is guarded by `GOLDSKY_RUN_MUTATIONS=1`; it
  exits with an actionable message otherwise.
- No example prints a secret. Credentials are read from the environment and
  used only to authenticate requests.
- Examples use `context.Context` with a timeout so they can be cancelled.
- Examples are **not** executed against Goldsky in CI; they are only compiled.
