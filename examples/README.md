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
The REST token and Edge key are independent: the RPC examples do not need
`GOLDSKY_API_KEY`, and public GraphQL does not need either credential.

## Examples

| # | Example | Prerequisites | Command | Behavior |
| --- | --- | --- | --- | --- |
| 1 | `01-list-pipelines` | `GOLDSKY_API_KEY` | `go run ./examples/01-list-pipelines` | Constructs a client and lists one page of pipelines. |
| 2 | `02-paginate-subgraphs` | `GOLDSKY_API_KEY` | `go run ./examples/02-paginate-subgraphs` | Walks all subgraph pages with the cancellation-aware pager. |
| 3 | `03-validate-pipeline` | `GOLDSKY_API_KEY` | `go run ./examples/03-validate-pipeline` | Validates a realistic Base USDC transfer filter with a blackhole sink, without creating it. |
| 4 | `04-create-pipeline` | `GOLDSKY_API_KEY`, `GOLDSKY_RUN_MUTATIONS=1`, `GOLDSKY_PIPELINE_NAME` | `go run ./examples/04-create-pipeline` | Creates a Base transfer dataset-to-blackhole pipeline. **Mutation**: refuses to run unless explicitly enabled. |
| 5 | `05-graphql-query` | `GOLDSKY_API_KEY`, `GOLDSKY_PROJECT_ID`, `GOLDSKY_SUBGRAPH_NAME`, `GOLDSKY_SUBGRAPH_VERSION` | `go run ./examples/05-graphql-query` | Queries `_meta.block.number` from an explicit private Subgraph version or tag. |
| 6 | `06-edge-rpc` | `GOLDSKY_EDGE_API_KEY` | `go run ./examples/06-edge-rpc` | Calls `eth_blockNumber` with a data-only client and converts the hexadecimal result to a number. |
| 7 | `07-verify-webhook` | `GOLDSKY_WEBHOOK_SECRET`; optional `GOLDSKY_WEBHOOK_ADDR` | `go run ./examples/07-verify-webhook` | Runs a bounded, graceful local webhook handler that verifies the shared secret before decoding JSON. |
| 8 | `08-handle-errors` | `GOLDSKY_API_KEY`, `GOLDSKY_PIPELINE_NAME` | `go run ./examples/08-handle-errors` | Distinguishes API problems, rate limits, missing resources, and transport failures. |
| 9 | `09-batch-edge-rpc` | `GOLDSKY_EDGE_API_KEY`; optional `GOLDSKY_CHAIN_ID` | `go run ./examples/09-batch-edge-rpc` | Fetches chain ID and latest block in one batch and checks every item for an RPC error. |

## Notes

- Examples 1-3, 5-6, 8, and 9 only read from Goldsky. Example 7 starts a local
  HTTP server and makes no Goldsky API call.
- Example 4 creates a real pipeline and may consume project resources. It is
  guarded by `GOLDSKY_RUN_MUTATIONS=1` and exits before making a request unless
  you set that value explicitly.
- No example prints a secret. Credentials are read from the environment and
  used only to authenticate requests.
- Network examples use `context.Context` with a timeout so they can be
  cancelled.
- Examples are **not** executed against Goldsky in CI; they are only compiled.

## A safe way to explore

Start with example 3. It exercises Goldsky's server-side pipeline validator but
does not create resources. Once the definition is accepted, choose a unique
`GOLDSKY_PIPELINE_NAME`, enable the mutation guard, and run example 4. The
blackhole sink intentionally discards output, making the example useful for
learning configuration without provisioning a database destination.

For GraphQL, pass a deployed version or a tag you created (for example `prod`).
The examples do not guess a `current` alias because that alias is not part of
the documented endpoint contract.

## Troubleshooting

- `ErrAPITokenRequired` means a REST or private GraphQL operation was attempted
  with `NewDataClient`; use `NewClient` with a project token.
- `Edge API key is required` means `GOLDSKY_EDGE_API_KEY` is empty. It is not
  the same value as `GOLDSKY_API_KEY`.
- A GraphQL HTTP request may succeed and still contain GraphQL errors. Check
  `resp.HasErrors()` before decoding `resp.Data`.
- A batch RPC request may contain per-item failures. Check every returned
  `RPCResponse.Error`, as example 9 does.
