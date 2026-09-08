# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-08

### Fixed

- Made pagers stop after the terminal page and validate page sizes before a
  request.
- Made HTTP response status authoritative for problem details and rejected
  trailing JSON values.
- Prevented retries of non-replayable streaming deployments and rejected CR/LF
  injection in multipart filenames.
- Made timeout options order-independent without mutating a supplied HTTP
  client, escaped GraphQL path components, and validated JSON-RPC inputs and
  response IDs.
- Added the documented `definition.name` pipeline authoring field and let the
  validation endpoint report malformed names as structured findings.
- Corrected and expanded the runnable examples and README guidance.

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
