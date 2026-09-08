# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-08

### Added

- Initial release of `goldsky-go`, built against Goldsky REST API v1.2.0.
- REST control-plane services covering all 40 operations: Pipelines, Subgraphs,
  Webhooks, Edge, and Catalogs.
- Subgraph GraphQL data-plane client (public/private endpoints).
- Edge HTTPS JSON-RPC 2.0 data-plane client (single and batch).
- Webhook secret verification helper (constant-time).
- RFC 9457 `application/problem+json` error parsing with status predicates.
- Retry policy defaulting to safe reads only, honouring `Retry-After`, with
  opt-in mutation retry.
- Cancellation-aware pagers for pipelines, subgraphs, and Edge endpoints.
- Streaming `multipart/form-data` subgraph deployment.
- Table-driven contract tests for all 40 operations plus edge-case tests.
- Seven runnable examples, a wiki documentation set, and API coverage map.
