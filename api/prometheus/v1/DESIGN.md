# v1 Refactor Note

`api/prometheus/v1` keeps the public API surface, docs, and caller-facing behavior in the top-level package while delegating wire-level concerns to internal subpackages.

- `internal/transport` owns HTTP execution, POST-with-GET fallback, status handling, and normalized API/HTTP error mapping.
- `internal/codec` owns the custom `jsoniter` registrations plus Prometheus-specific sample, histogram, and query-result serialization logic.
- `internal/endpoints` owns endpoint path constants and shared helpers for matcher/time query construction.
- `internal/types` holds internal wire-format structs shared by transport and codec.

Tradeoffs:

- Exported response models, enums, options, and docs stay in package `v1` so callers and `go doc` do not see internal package names.
- Rule and result unmarshalling methods remain in package `v1` because they operate directly on public types; pushing them behind aliases would leak internal packages into the public surface.
- The internal transport layer uses a small error-construction callback so it can normalize responses without depending on public `v1` error types and without introducing cycles.
