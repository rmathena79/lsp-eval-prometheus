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

// Package transport provides HTTP execution helpers for the Prometheus v1 API
// client. It is responsible for the POST-with-GET-fallback protocol used by
// query endpoints: a POST is attempted first; if the server responds with
// 405 (Method Not Allowed) or 501 (Not Implemented), the request is retried
// as a GET with the same parameters encoded in the query string.
//
// The package operates purely at the HTTP level and returns raw response bytes.
// Callers are responsible for parsing the Prometheus API response envelope.
package transport

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/prometheus/client_golang/api"
)

// DoGetFallback sends a POST request with form-encoded args to u. If the
// server responds with 405 (Method Not Allowed) or 501 (Not Implemented),
// the request is retried as a GET with args encoded in the query string.
//
// The raw response and body bytes are returned; callers must parse the
// Prometheus API envelope themselves.
//
// An Idempotency-Key header with a nil value is set on the POST to opt into
// transport-level retries on network errors without sending the header on the
// wire. See https://pkg.go.dev/net/http#Transport for details.
func DoGetFallback(ctx context.Context, client api.Client, u *url.URL, args url.Values) (*http.Response, []byte, error) {
	encodedArgs := args.Encode()
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(encodedArgs))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Setting Idempotency-Key to nil marks the request as idempotent without
	// transmitting the header, enabling transport-level retries on network errors.
	req.Header["Idempotency-Key"] = nil

	resp, body, err := client.Do(ctx, req)
	if resp != nil && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented) {
		u.RawQuery = encodedArgs
		req, err = http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, nil, err
		}
		return client.Do(ctx, req)
	}
	return resp, body, err
}
