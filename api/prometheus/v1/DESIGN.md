# v1 Package Design Note

The public `api/prometheus/v1` package remains the compatibility surface: exported docs, types, option helpers, and `NewAPI` stay there so callers and `go doc` continue to see the same API.

Implementation details moved behind that facade:

- `internal/transport` owns HTTP execution, POST-with-GET fallback, response status handling, and transport-level error normalization.
- `internal/codec` owns the custom `json-iterator` registrations plus shared query/result decoding helpers.
- `internal/endpoints` centralizes endpoint paths and small request/query helper functions.
- `internal/types` holds internal-only response envelopes and option payloads shared across the internal packages.

Tradeoffs:

- Exported response and option-facing types stay in the public package to preserve documentation quality and avoid leaking internal package names into the API.
- The public package still contains thin rule unmarshaling glue because those methods belong to exported public types, but the JSON parsing work now delegates to `internal/codec`.
- The facade keeps duplication low without adding another abstraction layer on top of the existing client model.
