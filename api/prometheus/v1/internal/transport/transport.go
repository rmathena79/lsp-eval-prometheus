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
	v1types "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

type Client interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, v1types.Warnings, error)
	DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, v1types.Warnings, error)
}

type APIResponse struct {
	Status    string            `json:"status"`
	Data      json.RawMessage   `json:"data"`
	ErrorType v1types.ErrorType `json:"errorType"`
	Error     string            `json:"error"`
	Warnings  []string          `json:"warnings,omitempty"`
}

type client struct {
	client api.Client
}

func NewClient(c api.Client) Client {
	return &client{client: c}
}

func APIError(code int) bool {
	return code == http.StatusUnprocessableEntity || code == http.StatusBadRequest
}

func ErrorTypeAndMsgFor(resp *http.Response) (v1types.ErrorType, string) {
	switch resp.StatusCode / 100 {
	case 4:
		return v1types.ErrClient, fmt.Sprintf("client error: %d", resp.StatusCode)
	case 5:
		return v1types.ErrServer, fmt.Sprintf("server error: %d", resp.StatusCode)
	}
	return v1types.ErrBadResponse, fmt.Sprintf("bad response code %d", resp.StatusCode)
}

func (h *client) URL(ep string, args map[string]string) *url.URL {
	return h.client.URL(ep, args)
}

func (h *client) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, v1types.Warnings, error) {
	resp, body, err := h.client.Do(ctx, req)
	if err != nil {
		return resp, body, nil, err
	}

	code := resp.StatusCode
	if code/100 != 2 && !APIError(code) {
		errorType, errorMsg := ErrorTypeAndMsgFor(resp)
		return resp, body, nil, &v1types.Error{
			Type:   errorType,
			Msg:    errorMsg,
			Detail: string(body),
		}
	}

	var result APIResponse
	if http.StatusNoContent != code {
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return resp, body, nil, &v1types.Error{
				Type: v1types.ErrBadResponse,
				Msg:  jsonErr.Error(),
			}
		}
	}

	if APIError(code) && result.Status == "success" {
		err = &v1types.Error{
			Type: v1types.ErrBadResponse,
			Msg:  "inconsistent body for response code",
		}
	}

	if result.Status == "error" {
		err = &v1types.Error{
			Type: result.ErrorType,
			Msg:  result.Error,
		}
	}

	return resp, []byte(result.Data), result.Warnings, err
}

func (h *client) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, v1types.Warnings, error) {
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
