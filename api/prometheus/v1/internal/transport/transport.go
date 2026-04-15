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

package transport

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

type HTTPClient interface {
	Do(context.Context, *http.Request) (*http.Response, []byte, error)
}

type Config struct {
	Unmarshal   func([]byte, interface{}) error
	APIError    func(int) bool
	StatusError func(*http.Response) (string, string)
	NewError    func(typ, msg, detail string) error
}

func Do(ctx context.Context, client HTTPClient, req *http.Request, cfg Config) (*http.Response, []byte, []string, error) {
	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return resp, body, nil, err
	}

	code := resp.StatusCode
	if code/100 != 2 && !cfg.APIError(code) {
		errorType, errorMsg := cfg.StatusError(resp)
		return resp, body, nil, cfg.NewError(errorType, errorMsg, string(body))
	}

	var result types.APIResponse
	if http.StatusNoContent != code {
		if jsonErr := cfg.Unmarshal(body, &result); jsonErr != nil {
			return resp, body, nil, cfg.NewError("bad_response", jsonErr.Error(), "")
		}
	}

	if cfg.APIError(code) && result.Status == "success" {
		err = cfg.NewError("bad_response", "inconsistent body for response code", "")
	}
	if result.Status == "error" {
		err = cfg.NewError(result.ErrorType, result.Error, "")
	}

	return resp, []byte(result.Data), result.Warnings, err
}

// DoGetFallback will attempt to do the request as-is, and on a 405 or 501 it
// will fallback to a GET request.
func DoGetFallback(ctx context.Context, client HTTPClient, u *url.URL, args url.Values, cfg Config) (*http.Response, []byte, []string, error) {
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

	resp, body, warnings, err := Do(ctx, client, req, cfg)
	if resp != nil && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented) {
		u.RawQuery = encodedArgs
		req, err = http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, nil, warnings, err
		}
		return Do(ctx, client, req, cfg)
	}
	return resp, body, warnings, err
}
