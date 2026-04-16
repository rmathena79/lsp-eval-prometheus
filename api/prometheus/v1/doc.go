// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package v1 provides bindings to the Prometheus HTTP API v1.

# Design note — internal subpackage boundaries

The package is structured as a thin compatibility facade over four internal
subpackages.  The public surface (exported names, signatures, and behaviour)
is unchanged from the original single-file implementation.

## Package boundaries

	internal/types
	    All exported data types, string enums, constants, and option
	    machinery.  Owns the APIOptions struct and the Option function type so
	    that endpoints can use them without importing v1.  Also owns the
	    UnmarshalJSON implementations for RuleGroup, AlertingRule, and
	    RecordingRule because those methods are intimately coupled to the type
	    definitions.  No HTTP or serialisation logic lives here.
	    Dependencies: fmt, time, errors, json-iterator, prometheus/common/model.

	internal/codec
	    Registers custom json-iterator encoder/decoder functions for
	    model.SamplePair, model.SampleHistogramPair, and model.SampleStream.
	    The registration happens in the package's init() function, which runs
	    automatically when the v1 package is imported (v1 imports codec,
	    triggering init exactly once per process).  Also owns QueryResult, the
	    envelope type used to decode instant- and range-query responses.
	    Dependencies: fmt, math, strconv, unsafe, json-iterator, model.

	internal/transport
	    Owns the HTTP execution layer: the Do and DoGetFallback package-level
	    functions, the APIResponse envelope struct, and the error-mapping
	    helpers (APIError, ErrorTypeAndMsgFor).  DoGetFallback implements the
	    POST-with-GET-fallback strategy for 405/501 responses.  Functions
	    rather than methods are used so that the v1 package can keep its
	    unexported apiClientImpl struct (with an unexported "client" field)
	    while still delegating all HTTP logic here.
	    Dependencies: context, fmt, net/http, net/url, strings, json-iterator,
	    api (parent), internal/types.

	internal/endpoints
	    Defines all endpoint path constants (EPAlerts, EPQuery, …),
	    AddOptionalURLParams (applies Option values to a url.Values), and
	    FormatTime (formats a time.Time for Prometheus API parameters).
	    Dependencies: net/url, strconv, time, internal/types.

## Tradeoffs

Keeping queryResult and apiClientImpl in the v1 package (rather than moving
them to internal subpackages) was a deliberate choice to avoid breaking the
existing white-box test suite, which constructs both types via unexported
field names.  Moving them to an internal package would require exporting their
fields, changing the test code, or introducing indirection that adds noise
without benefit.

The queryResult.UnmarshalJSON method delegates to codec.QueryResult to avoid
duplicating the decode logic; the v field remains lowercase so that the test
can construct a queryResult by name without accessing v.

Type aliases (e.g. type AlertState = types.AlertState) propagate every public
name from internal/types through the v1 package without copying definitions.
Go's godoc renders aliased types as if they were defined locally, so the
public documentation is not degraded.

## Dependency graph (no cycles)

	v1 → types, codec, transport, endpoints
	transport → types
	endpoints → types
	codec → (nothing internal)
	types → (nothing internal)
*/
package v1
