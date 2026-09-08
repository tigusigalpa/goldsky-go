# Goldsky Go SDK examples

These are intentionally small programs you can read in a minute, run as-is,
and then adapt to your application. Run commands from the repository root with
`go run ./examples/<name>`.

## Prerequisites

Set only the environment variables needed by the example you want to run.
These values are placeholders; never commit real credentials.

```bash
export GOLDSKY_API_KEY="your_project_api_token"   # REST control plane
export GOLDSKY_EDGE_API_KEY="your_edge_api_key"   # Edge RPC; separate secret
```

PowerShell:

```powershell
$env:GOLDSKY_API_KEY = "your_project_api_token"
$env:GOLDSKY_EDGE_API_KEY = "your_edge_api_key"
```

No setup or external Go dependencies are required beyond Go 1.22 or later.

## Examples

| # | Example | Prerequisites | Command | Behavior |
| --- | --- | --- | --- | --- |
| 1 | `01-list-pipelines` | `GOLDSKY_API_KEY` | `go run ./examples/01-list-pipelines` | Constructs a client and lists one page of pipelines. |
| 2 | `02-paginate-subgraphs` | `GOLDSKY_API_KEY` | `go run ./examples/02-paginate-subgraphs` | Walks all subgraph pages with the cancellation-aware pager. |
| 3 | `03-validate-pipeline` | `GOLDSKY_API_KEY` | `go run ./examples/03-validate-pipeline` | Validates a pipeline definition without creating it. |
| 4 | `04-create-pipeline` | `GOLDSKY_API_KEY`, `GOLDSKY_RUN_MUTATIONS=1`; optional `GOLDSKY_PIPELINE_NAME` | `go run ./examples/04-create-pipeline` | Creates a dataset-to-blackhole pipeline. **Mutation**: refuses to run unless explicitly enabled. |
| 5 | `05-graphql-query` | `GOLDSKY_API_KEY`, project/subgraph/version | `go run ./examples/05-graphql-query` | Queries a private Subgraph GraphQL endpoint. |
| 6 | `06-edge-rpc` | `GOLDSKY_API_KEY`, `GOLDSKY_EDGE_API_KEY` | `go run ./examples/06-edge-rpc` | Calls `eth_blockNumber` and converts the hexadecimal result to a number. |
| 7 | `07-verify-webhook` | `GOLDSKY_WEBHOOK_SECRET`; optional `GOLDSKY_WEBHOOK_ADDR` | `go run ./examples/07-verify-webhook` | Starts a local handler that verifies the webhook secret in constant time. |
| 8 | `08-handle-errors` | `GOLDSKY_API_KEY`, `GOLDSKY_PIPELINE_NAME` | `go run ./examples/08-handle-errors` | Distinguishes API problems, rate limits, missing resources, and transport failures. |

## Notes

- Examples 1-3, 5-6, and 8 only read from Goldsky. Example 7 starts a local
  HTTP server and makes no Goldsky API call.
- Example 4 creates a real pipeline and may consume project resources. It is
  guarded by `GOLDSKY_RUN_MUTATIONS=1` and exits before making a request unless
  you set that value explicitly.
- No example prints a secret. Credentials are read from the environment and
  used only to authenticate requests.
- Network examples use `context.Context` with a timeout so they can be
  cancelled.
- Examples are **not** executed against Goldsky in CI; they are only compiled.
