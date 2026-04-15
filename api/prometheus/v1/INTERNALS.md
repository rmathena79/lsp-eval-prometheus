# v1 package boundaries

The public `api/prometheus/v1` package remains the compatibility layer for callers: exported names, signatures, docs, and wire-visible behavior stay in `v1`, while implementation details move behind internal packages.

- `internal/transport` owns HTTP execution, POST-with-GET fallback, status handling, and API response normalization.
- `internal/codec` owns the `json-iterator` registrations and the custom Prometheus model encoding/decoding behavior.
- `internal/endpoints` owns endpoint path definitions plus shared request-construction helpers.
- `internal/types` holds internal wire-envelope data structures that should not surface in the public API.

Tradeoffs:

- Public API types remain defined in `v1` so `go doc`, method signatures, and package navigation stay stable.
- The facade still contains some per-endpoint orchestration; that keeps behavior explicit and avoids introducing extra abstraction into a stable package.
