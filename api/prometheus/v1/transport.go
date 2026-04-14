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

package v1

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/prometheus/client_golang/api"
)

// Warnings is an array of non critical errors
type Warnings []string

// apiClient wraps a regular client and processes successful API responses.
// Successful also includes responses that errored at the API level.
type apiClient interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, Warnings, error)
	DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error)
}

type apiClientImpl struct {
	client api.Client
}

type apiResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType ErrorType       `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings,omitempty"`
}

func apiError(code int) bool {
	// These are the codes that Prometheus sends when it returns an error.
	return code == http.StatusUnprocessableEntity || code == http.StatusBadRequest
}

func errorTypeAndMsgFor(resp *http.Response) (ErrorType, string) {
	switch resp.StatusCode / 100 {
	case 4:
		return ErrClient, fmt.Sprintf("client error: %d", resp.StatusCode)
	case 5:
		return ErrServer, fmt.Sprintf("server error: %d", resp.StatusCode)
	}
	return ErrBadResponse, fmt.Sprintf("bad response code %d", resp.StatusCode)
}

func (h *apiClientImpl) URL(ep string, args map[string]string) *url.URL {
	return h.client.URL(ep, args)
}

func (h *apiClientImpl) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, Warnings, error) {
	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return resp, body, nil, err
	}

	code := resp.StatusCode

	if code/100 != 2 && !apiError(code) {
		errorType, errorMsg := errorTypeAndMsgFor(resp)
		return resp, body, nil, &Error{
			Type:   errorType,
			Msg:    errorMsg,
			Detail: string(body),
		}
	}

	var result apiResponse

	if http.StatusNoContent != code {
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return resp, body, nil, &Error{
				Type: ErrBadResponse,
				Msg:  jsonErr.Error(),
			}
		}
	}

	if apiError(code) && result.Status == "success" {
		err = &Error{
			Type: ErrBadResponse,
			Msg:  "inconsistent body for response code",
		}
	}

	if result.Status == "error" {
		err = &Error{
			Type: result.ErrorType,
			Msg:  result.Error,
		}
	}

	return resp, []byte(result.Data), result.Warnings, err
}

// DoGetFallback will attempt to do the request as-is, and on a 405 or 501 it
// will fallback to a GET request.
func (h *apiClientImpl) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error) {
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

	resp, body, warnings, err := h.Do(ctx, req)
	if resp != nil && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented) {
		u.RawQuery = encodedArgs
		req, err = http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, nil, warnings, err
		}
		return h.Do(ctx, req)
	}
	return resp, body, warnings, err
}
