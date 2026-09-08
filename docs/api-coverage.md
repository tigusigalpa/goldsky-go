# API Coverage

This document maps every operation in the Goldsky REST API endpoint manifest to
its public Go method, request/options struct, response model, contract test,
and exact documentation URL.

## OpenAPI snapshot

| Field | Value |
| --- | --- |
| Source | <https://api.goldsky.com/api/v1/docs/openapi.json> |
| `openapi` | `3.1.0` |
| `info.title` | Goldsky API |
| `info.version` | `1.2.0` |
| Operation count | 40 |
| Snapshot date | 2026-09-08 |
| Local fixture | `testdata/openapi/openapi.json` |

The live OpenAPI document is the primary contract. The examined reference
version is Goldsky REST v1.2.0 with 40 operations. At build time the live spec
was fetched and diffed against the manifest below; **no delta was found** — all
40 operations, paths, and verbs match.

### Updating the snapshot safely

1. Download the live spec:
   `curl -sSL https://api.goldsky.com/api/v1/docs/openapi.json -o testdata/openapi/openapi.json`
2. Record the new `info.version` and fetch date in the table above.
3. Compare the operation list against the manifest below. If the spec changed,
   implement the live spec, update this file and the contract tests, and report
   the delta in `CHANGELOG.md`.
4. Run `go test ./...` and ensure the contract suite still passes.

## Endpoint manifest

The 40 links below are concrete operation pages in the interactive Goldsky API
reference. The companion machine-readable source is
<https://api.goldsky.com/api/v1/docs/openapi.json>.

### Turbo Pipelines

| # | Capability | Method | Path | Go method | Request struct | Response model | Test | Docs |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | List pipelines | GET | `/pipelines` | `Pipelines.List` | `ListPipelinesOptions` | `Page[Pipeline]` | `TestContractOperations/listPipelines` | <https://api.goldsky.com/api/v1/docs#tag/Pipelines/operation/listPipelines> |
| 2 | Create pipeline | POST | `/pipelines` | `Pipelines.Create` | `CreatePipelineRequest` | `Pipeline` | `TestContractOperations/createPipeline` | <https://api.goldsky.com/api/v1/docs#tag/Pipelines/operation/createPipeline> |
| 3 | Get pipeline | GET | `/pipelines/{name}` | `Pipelines.Get` | (name) | `Pipeline` | `TestContractOperations/getPipeline` | <https://api.goldsky.com/api/v1/docs#tag/Pipelines/operation/getPipeline> |
| 4 | Delete pipeline | DELETE | `/pipelines/{name}` | `Pipelines.Delete` | (name) | — | `TestContractOperations/deletePipeline` | <https://api.goldsky.com/api/v1/docs#tag/Pipelines/operation/deletePipeline> |
| 5 | Validate pipeline | POST | `/pipelines/validate` | `Pipelines.Validate` | `ValidatePipelineRequest` | `ValidatePipelineResponse` | `TestContractOperations/validatePipeline` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Authoring/operation/validatePipeline> |
| 6 | Preview pipeline | POST | `/pipelines/preview` | `Pipelines.Preview` | `PreviewPipelineRequest` | `PreviewPipelineResponse` | `TestContractOperations/previewPipeline` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Authoring/operation/previewPipeline> |
| 7 | Pause pipeline | PUT | `/pipelines/{name}/pause` | `Pipelines.Pause` | (name) | — | `TestContractOperations/pausePipeline` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Lifecycle/operation/pausePipeline> |
| 8 | Resume pipeline | PUT | `/pipelines/{name}/resume` | `Pipelines.Resume` | (name) | — | `TestContractOperations/resumePipeline` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Lifecycle/operation/resumePipeline> |
| 9 | Restart pipeline | PUT | `/pipelines/{name}/restart` | `Pipelines.Restart` | `*RestartPipelineRequest` | — | `TestContractOperations/restartPipeline` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Lifecycle/operation/restartPipeline> |
| 10 | Get pipeline logs | GET | `/pipelines/{name}/logs` | `Pipelines.Logs` | `PipelineLogsOptions` | `PipelineLogsResponse` | `TestContractOperations/getPipelineLogs` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Logs/operation/getPipelineLogs> |
| 11 | Get pipeline error count | GET | `/pipelines/{name}/logs/error-count` | `Pipelines.ErrorCount` | (name, sinceHours) | `PipelineErrorCountResponse` | `TestContractOperations/getPipelineErrorCount` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Logs/operation/getPipelineErrorCount> |
| 12 | Get pipeline status | GET | `/pipelines/{name}/status` | `Pipelines.Status` | (name) | `PipelineStatusResponse` | `TestContractOperations/getPipelineStatus` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Status/operation/getPipelineStatus> |
| 13 | Get pipeline state | GET | `/pipelines/{name}/state` | `Pipelines.State` | (name) | `PipelineStateResponse` | `TestContractOperations/getPipelineState` | <https://api.goldsky.com/api/v1/docs#tag/Pipeline%20Status/operation/getPipelineState> |

### Subgraphs and Subgraph Webhooks

| # | Capability | Method | Path | Go method | Request struct | Response model | Test | Docs |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 14 | List subgraphs | GET | `/subgraphs` | `Subgraphs.List` | `ListSubgraphsOptions` | `Page[Subgraph]` | `TestContractOperations/listSubgraphs` | <https://api.goldsky.com/api/v1/docs#tag/Subgraphs/operation/listSubgraphs> |
| 15 | Get subgraph | GET | `/subgraphs/{name}` | `Subgraphs.Get` | (name) | `Page[Subgraph]` | `TestContractOperations/getSubgraph` | <https://api.goldsky.com/api/v1/docs#tag/Subgraphs/operation/getSubgraph> |
| 16 | List supported chains | GET | `/subgraphs/supported-chains` | `Subgraphs.SupportedChains` / `Catalogs.SupportedSubgraphChains` | — | `SubgraphChainsResponse` | `TestContractOperations/listSubgraphChains` | <https://api.goldsky.com/api/v1/docs#tag/Catalogs/operation/listSubgraphChains> |
| 17 | Get subgraph version | GET | `/subgraphs/{name}/{version}` | `Subgraphs.GetVersion` | (name, version) | `Page[Subgraph]` | `TestContractOperations/getSubgraphVersion` | <https://api.goldsky.com/api/v1/docs#tag/Subgraphs/operation/getSubgraphVersion> |
| 18 | Update version | PATCH | `/subgraphs/{name}/{version}` | `Subgraphs.UpdateVersion` | `UpdateSubgraphVersionRequest` | `Subgraph` | `TestContractOperations/updateSubgraphVersion` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Lifecycle/operation/updateSubgraphVersion> |
| 19 | Get subgraph logs | GET | `/subgraphs/{name}/{version}/logs` | `Subgraphs.Logs` | `SubgraphLogsOptions` | `SubgraphLogsResponse` | `TestContractOperations/getSubgraphLogs` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Logs/operation/getSubgraphLogs> |
| 20 | Pause subgraph | PUT | `/subgraphs/{name}/{version}/pause` | `Subgraphs.Pause` | (name, version) | — | `TestContractOperations/pauseSubgraph` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Lifecycle/operation/pauseSubgraph> |
| 21 | Resume subgraph | PUT | `/subgraphs/{name}/{version}/resume` | `Subgraphs.Resume` | (name, version) | — | `TestContractOperations/resumeSubgraph` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Lifecycle/operation/resumeSubgraph> |
| 22 | Set tag | PUT | `/subgraphs/{name}/tags/{version}` | `Subgraphs.SetTag` | `SetSubgraphTagRequest` | `Subgraph` | `TestContractOperations/setSubgraphTag` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Tags/operation/setSubgraphTag> |
| 23 | Delete tag | DELETE | `/subgraphs/{name}/tags/{version}` | `Subgraphs.DeleteTag` | (name, version) | — | `TestContractOperations/deleteSubgraphTag` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Tags/operation/deleteSubgraphTag> |
| 24 | Delete deployment | DELETE | `/subgraphs/{name}/deployments/{version}` | `Subgraphs.DeleteDeployment` | (name, version) | — | `TestContractOperations/deleteSubgraphDeployment` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Deployments/operation/deleteSubgraphDeployment> |
| 25 | Deploy subgraph | PUT | `/subgraphs/{name}/deployments/{version}` | `Subgraphs.Deploy` | `DeploySubgraphOptions` | `Subgraph` | `TestContractOperations/deploySubgraph` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Deployments/operation/deploySubgraph> |
| 26 | List webhooks | GET | `/subgraphs/webhooks` | `Webhooks.List` | — | `WebhookListResponse` | `TestContractOperations/listWebhooks` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Webhooks/operation/listWebhooks> |
| 27 | Create webhook | POST | `/subgraphs/webhooks` | `Webhooks.Create` | `CreateWebhookRequest` | `CreateWebhookResponse` | `TestContractOperations/createWebhook` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Webhooks/operation/createWebhook> |
| 28 | Delete webhook | DELETE | `/subgraphs/webhooks/{name}` | `Webhooks.Delete` | (name) | — | `TestContractOperations/deleteWebhook` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Webhooks/operation/deleteWebhook> |
| 29 | List webhook entities | GET | `/subgraphs/{name}/{version}/entities` | `Subgraphs.WebhookEntities` | (name, version) | `WebhookEntitiesResponse` | `TestContractOperations/listWebhookEntities` | <https://api.goldsky.com/api/v1/docs#tag/Subgraph%20Webhooks/operation/listWebhookEntities> |

### Edge endpoints and catalogs

| # | Capability | Method | Path | Go method | Request struct | Response model | Test | Docs |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 30 | List Edge networks | GET | `/edge/networks` | `Catalogs.EdgeNetworks` | — | `EdgeNetworksResponse` | `TestContractOperations/listEdgeNetworks` | <https://api.goldsky.com/api/v1/docs#tag/Catalogs/operation/listEdgeNetworks> |
| 31 | List Edge sources | GET | `/edge/sources` | `Catalogs.EdgeSources` | — | `EdgeSourcesResponse` | `TestContractOperations/listEdgeSources` | <https://api.goldsky.com/api/v1/docs#tag/Catalogs/operation/listEdgeSources> |
| 32 | List Edge endpoints | GET | `/edge` | `Edge.List` | `ListEdgeEndpointsOptions` | `Page[EdgeEndpoint]` | `TestContractOperations/listEdgeEndpoints` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/listEdgeEndpoints> |
| 33 | Create Edge endpoint | POST | `/edge` | `Edge.Create` | `CreateEdgeEndpointRequest` | `CreateEdgeEndpointResponse` | `TestContractOperations/createEdgeEndpoint` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/createEdgeEndpoint> |
| 34 | Get Edge endpoint | GET | `/edge/{name}` | `Edge.Get` | (name) | `EdgeEndpoint` | `TestContractOperations/getEdgeEndpoint` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/getEdgeEndpoint> |
| 35 | Update Edge endpoint | PATCH | `/edge/{name}` | `Edge.Update` | `UpdateEdgeEndpointRequest` | `EdgeEndpoint` | `TestContractOperations/updateEdgeEndpoint` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/updateEdgeEndpoint> |
| 36 | Delete Edge endpoint | DELETE | `/edge/{name}` | `Edge.Delete` | (name) | — | `TestContractOperations/deleteEdgeEndpoint` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20Endpoints/operation/deleteEdgeEndpoint> |
| 37 | Pause Edge endpoint | PUT | `/edge/{name}/pause` | `Edge.Pause` | (name) | `EdgeEndpoint` | `TestContractOperations/pauseEdgeEndpoint` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20Lifecycle/operation/pauseEdgeEndpoint> |
| 38 | Resume Edge endpoint | PUT | `/edge/{name}/resume` | `Edge.Resume` | (name) | `EdgeEndpoint` | `TestContractOperations/resumeEdgeEndpoint` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20Lifecycle/operation/resumeEdgeEndpoint> |
| 39 | Reveal Edge API key | GET | `/edge/{name}/api-key` | `Edge.RevealKey` | (name) | `RevealEdgeKeyResponse` | `TestContractOperations/revealEdgeEndpointKey` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20API%20Keys/operation/revealEdgeEndpointKey> |
| 40 | Get Edge metrics | GET | `/edge/{name}/metrics` | `Edge.Metrics` | `EdgeMetricsOptions` | `EdgeMetricsResponse` | `TestContractOperations/getEdgeEndpointMetrics` | <https://api.goldsky.com/api/v1/docs#tag/Edge%20Metrics/operation/getEdgeEndpointMetrics> |

## Data planes (outside the REST manifest)

| Capability | Go method | Docs |
| --- | --- | --- |
| GraphQL Subgraph query (public/private) | `GraphQL.Query`, `GraphQL.QueryPublic`, `GraphQL.QueryPrivate` | <https://docs.goldsky.com/subgraphs/graphql-endpoints> |
| Edge JSON-RPC (single/batch) | `RPC.Call`, `RPC.Batch` | <https://docs.goldsky.com/edge-rpc/quickstart> |
| Webhook secret verification | `VerifyWebhookSecret`, `VerifyWebhookRequest` | <https://docs.goldsky.com/subgraphs/webhooks> |

## Pagination

`Pipelines.List`, `Subgraphs.List`, and `Edge.List` return a `Page[T]` with a
`Pagination.NextPageToken`. Use `Pipelines.NewPipelinePager`,
`Subgraphs.NewSubgraphPager`, or `Edge.NewEdgePager` for full cancellation-aware
iteration. A page can hold fewer than `page_size` items and still have a next
page, so completion is inferred from `next_page_token` alone.
