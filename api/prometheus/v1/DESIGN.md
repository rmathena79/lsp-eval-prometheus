# v1 package design

The public [`v1`](/C:/source_gh/lsp-eval-prometheus/api/prometheus/v1/api.go) package stays as the compatibility entrypoint. It keeps `NewAPI`, the exported docs, and the public method signatures callers already use.

Internal boundaries:

- [`internal/transport`](/C:/source_gh/lsp-eval-prometheus/api/prometheus/v1/internal/transport/transport.go) owns HTTP execution, API response normalization, HTTP-to-API error mapping, and POST-with-GET fallback on `405` and `501`.
- [`internal/endpoints`](/C:/source_gh/lsp-eval-prometheus/api/prometheus/v1/internal/endpoints/endpoints.go) owns endpoint paths plus shared request and query construction helpers.
- [`internal/codec`](/C:/source_gh/lsp-eval-prometheus/api/prometheus/v1/internal/codec/codec.go) owns the custom `jsoniter` registrations and Prometheus model encoding/decoding behavior.
- [`internal/types`](/C:/source_gh/lsp-eval-prometheus/api/prometheus/v1/internal/types/types.go) owns shared response/data models and JSON unmarshaling logic used by the internal packages.

Tradeoffs:

- Exported docs and names remain in `v1` through aliases and thin forwarding methods so callers still start from the same public package.
- Public option helpers stay materialized in `v1` to avoid leaking internal implementation details into exported function signatures.
- A tiny adapter remains in `v1` for backward-compatible tests and construction, but behavior now lives in internal subpackages with no new public surface.
