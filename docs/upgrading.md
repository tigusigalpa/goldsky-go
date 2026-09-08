# Upgrading

## From versions prior to 1.0.0

There is no prior released version; 1.0.0 is the initial public release built
against Goldsky REST API v1.2.0.

## When the OpenAPI spec changes

Goldsky may evolve the REST API. The SDK treats the live OpenAPI document as
authoritative. To upgrade:

1. Re-run the snapshot procedure in [api-coverage.md](api-coverage.md).
2. If operations were added or changed, update the relevant service methods,
   models, contract tests, and this coverage map.
3. Models preserve unknown enum values verbatim, so a new server enum value
   does not break decoding; you only need to add a new constant if you want a
   named symbol for it.
4. Pipeline `definition`, source/transform/sink maps, and pipeline `state` are
   intentionally open (`map[string]any` / `json.RawMessage`) and do not require
   SDK changes when Goldsky adds nested fields.

## Breaking changes policy

The SDK follows semantic versioning. Breaking changes to public method
signatures or struct fields require a major version bump. Adding new methods,
options, or model fields is considered non-breaking.
