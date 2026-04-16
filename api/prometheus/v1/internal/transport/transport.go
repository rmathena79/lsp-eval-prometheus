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

// Package transport implements HTTP execution for the Prometheus v1 API
// client. It owns the POST-with-GET-fallback strategy, HTTP status code
// mapping, and API-level error normalisation.
//
// The two exported functions, Do and DoGetFallback, are free functions that
// accept an api.Client so that the v1 package can keep its unexported
// apiClientImpl struct (with its unexported field) while still delegating all
// HTTP logic here.
package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

// APIResponse is the envelope returned by every Prometheus v1 API endpoint.
type APIResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType types.ErrorType `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  types.Warnings  `json:"warnings,omitempty"`
}

// APIError reports whether the HTTP status code indicates an API-level error
// that still carries a structured JSON body.
func APIError(code int) bool {
	// These are the codes that Prometheus sends when it returns an error.
	return code == http.StatusUnprocessableEntity || code == http.StatusBadRequest
}

// ErrorTypeAndMsgFor maps an HTTP response status code to an ErrorType and a
// human-readable message.
func ErrorTypeAndMsgFor(resp *http.Response) (types.ErrorType, string) {
	switch resp.StatusCode / 100 {
	case 4:
		return types.ErrClient, fmt.Sprintf("client error: %d", resp.StatusCode)
	case 5:
		return types.ErrServer, fmt.Sprintf("server error: %d", resp.StatusCode)
	}
	return types.ErrBadResponse, fmt.Sprintf("bad response code %d", resp.StatusCode)
}

// Do executes req using client and processes the API response envelope.
// It maps HTTP and API-level errors to *types.Error and extracts the
// data payload and any warnings.
func Do(client api.Client, ctx context.Context, req *http.Request) (*http.Response, []byte, types.Warnings, error) {
	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return resp, body, nil, err
	}

	code := resp.StatusCode

	if code/100 != 2 && !APIError(code) {
		errorType, errorMsg := ErrorTypeAndMsgFor(resp)
		return resp, body, nil, &types.Error{
			Type:   errorType,
			Msg:    errorMsg,
			Detail: string(body),
		}
	}

	var result APIResponse

	if http.StatusNoContent != code {
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return resp, body, nil, &types.Error{
				Type: types.ErrBadResponse,
				Msg:  jsonErr.Error(),
			}
		}
	}

	if APIError(code) && result.Status == "success" {
		err = &types.Error{
			Type: types.ErrBadResponse,
			Msg:  "inconsistent body for response code",
		}
	}

	if result.Status == "error" {
		err = &types.Error{
			Type: result.ErrorType,
			Msg:  result.Error,
		}
	}

	return resp, []byte(result.Data), result.Warnings, err
}

// DoGetFallback attempts the request as a POST. On a 405 (Method Not Allowed)
// or 501 (Not Implemented) response it retries as a GET, encoding the
// parameters in the URL query string instead of the request body.
//
// The Idempotency-Key header trick follows Go's net/http Transport documentation:
// a zero-length slice marks the request as idempotent so the transport will
// retry on network errors without actually sending a header on the wire.
func DoGetFallback(client api.Client, ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, types.Warnings, error) {
	encodedArgs := args.Encode()
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(encodedArgs))
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Following comment originates from https://pkg.go.dev/net/http#Transport
	// Transport only retries a request upon encountering a network error if the request is
	// idempotent and either has no body or has its Request.GetBody defined. HTTP requests
	// are considered idempotent if they have HTTP methods GET, HEAD, OPTIONS, or TRACE; or
	// if their Header map contains an "Idempotency-Key" or "X-Idempotency-Key" entry. If the
	// idempotency key value is a zero-length slice, the request is treated as idempotent but
	// the header is not sent on the wire.
	req.Header["Idempotency-Key"] = nil

	resp, body, warnings, err := Do(client, ctx, req)
	if resp != nil && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented) {
		u.RawQuery = encodedArgs
		req, err = http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, nil, warnings, err
		}
		return Do(client, ctx, req)
	}
	return resp, body, warnings, err
}
