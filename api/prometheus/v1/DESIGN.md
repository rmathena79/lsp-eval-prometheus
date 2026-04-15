# v1 internal layout

`api/prometheus/v1` stays as the caller-facing package: exported docs, public data types, `NewAPI`, and the `API` method set remain there so navigation and `go doc` stay familiar.

The runtime behavior is split by responsibility:

- `internal/transport` owns HTTP execution, POST-with-GET fallback, status handling, and API error normalization.
- `internal/codec` owns jsoniter registration plus the custom model encoders and decoders.
- `internal/endpoints` owns endpoint paths plus lightweight request and query helpers.
- `internal/types` owns transport-facing response envelopes and internal query decoding models.

Tradeoffs:

- Exported public data types remain in `v1` instead of becoming aliases to internal packages. That keeps the public docs stable and avoids leaking internal package names into the API surface.
- The public methods are still small wrappers because they assemble request parameters and decode public result types, but the transport and codec behavior now lives behind focused internal packages.
