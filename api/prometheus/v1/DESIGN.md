# v1 package refactor note

`api/prometheus/v1` now acts as a compatibility facade over internal packages while preserving the public API exactly.

- `internal/transport` owns HTTP execution, POST-with-GET fallback, response envelope parsing, and error normalization. This keeps wire behavior in one place and makes fallback/error rules easy to test directly.
- `internal/codec` owns JSON decoding plus the custom `json-iterator` registrations for Prometheus model types. Keeping registration and custom marshal/unmarshal code together avoids scattering wire-format behavior across the facade.
- `internal/endpoints` owns endpoint paths plus small request/URL helpers. The facade still assembles method-specific query parameters so option handling stays close to the exported methods without introducing extra abstraction.
- `internal/types` owns API-facing data models and enum values. The public package re-exports them through aliases so callers keep the same names, method sets, and JSON behavior.

Tradeoffs:

- Public `Option` remains in `api/prometheus/v1` instead of moving into `internal/types`. That avoids leaking internal package names through the function type and keeps `WithTimeout`, `WithStats`, and related docs in the public entry point.
- A few package-private compatibility helpers remain in the public package for existing tests. They do not affect the exported API surface, but they let us keep the refactor focused instead of rewriting broad test coverage at the same time.
