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
	"fmt"
	"net/http"
	"net/url"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/prometheus/client_golang/api"
	internaltypes "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

type Client interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, internaltypes.Warnings, error)
	DoGetFallback(context.Context, *url.URL, url.Values) (*http.Response, []byte, internaltypes.Warnings, error)
}

type APIClient struct {
	client api.Client
}

func NewClient(client api.Client) Client {
	return &APIClient{client: client}
}

func (h *APIClient) URL(ep string, args map[string]string) *url.URL {
	return h.client.URL(ep, args)
}

func (h *APIClient) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, internaltypes.Warnings, error) {
	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return resp, body, nil, err
	}

	code := resp.StatusCode
	if code/100 != 2 && !apiError(code) {
		errorType, errorMsg := errorTypeAndMsgFor(resp)
		return resp, body, nil, &internaltypes.Error{
			Type:   errorType,
			Msg:    errorMsg,
			Detail: string(body),
		}
	}

	var result internaltypes.APIResponse
	if http.StatusNoContent != code {
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return resp, body, nil, &internaltypes.Error{
				Type: internaltypes.ErrBadResponse,
				Msg:  jsonErr.Error(),
			}
		}
	}

	if apiError(code) && result.Status == "success" {
		err = &internaltypes.Error{
			Type: internaltypes.ErrBadResponse,
			Msg:  "inconsistent body for response code",
		}
	}

	if result.Status == "error" {
		err = &internaltypes.Error{
			Type: result.ErrorType,
			Msg:  result.Error,
		}
	}

	return resp, []byte(result.Data), result.Warnings, err
}

func (h *APIClient) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, internaltypes.Warnings, error) {
	encodedArgs := args.Encode()
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(encodedArgs))
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func apiError(code int) bool {
	return code == http.StatusUnprocessableEntity || code == http.StatusBadRequest
}

func errorTypeAndMsgFor(resp *http.Response) (internaltypes.ErrorType, string) {
	switch resp.StatusCode / 100 {
	case 4:
		return internaltypes.ErrClient, fmt.Sprintf("client error: %d", resp.StatusCode)
	case 5:
		return internaltypes.ErrServer, fmt.Sprintf("server error: %d", resp.StatusCode)
	default:
		return internaltypes.ErrBadResponse, fmt.Sprintf("bad response code %d", resp.StatusCode)
	}
}
