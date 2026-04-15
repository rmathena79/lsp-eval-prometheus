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

type APIClient interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, []string, error)
	DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, []string, error)
}

type Error struct {
	Type   string
	Msg    string
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Msg)
}

func NewAPIClient(client api.Client) APIClient {
	return &apiClient{client: client}
}

type apiClient struct {
	client api.Client
}

func (c *apiClient) URL(ep string, args map[string]string) *url.URL {
	return c.client.URL(ep, args)
}

func (c *apiClient) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, []string, error) {
	resp, body, err := c.client.Do(ctx, req)
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

	var result internaltypes.APIResponse
	if code != http.StatusNoContent {
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return resp, body, nil, &Error{
				Type: "bad_response",
				Msg:  jsonErr.Error(),
			}
		}
	}

	if apiError(code) && result.Status == "success" {
		err = &Error{
			Type: "bad_response",
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

func (c *apiClient) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, []string, error) {
	encodedArgs := args.Encode()
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(encodedArgs))
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header["Idempotency-Key"] = nil

	resp, body, warnings, err := c.Do(ctx, req)
	if resp != nil && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented) {
		getURL := *u
		getURL.RawQuery = encodedArgs
		req, err = http.NewRequest(http.MethodGet, getURL.String(), nil)
		if err != nil {
			return nil, nil, warnings, err
		}
		return c.Do(ctx, req)
	}
	return resp, body, warnings, err
}

func apiError(code int) bool {
	return code == http.StatusUnprocessableEntity || code == http.StatusBadRequest
}

func errorTypeAndMsgFor(resp *http.Response) (string, string) {
	switch resp.StatusCode / 100 {
	case 4:
		return "client_error", fmt.Sprintf("client error: %d", resp.StatusCode)
	case 5:
		return "server_error", fmt.Sprintf("server error: %d", resp.StatusCode)
	}
	return "bad_response", fmt.Sprintf("bad response code %d", resp.StatusCode)
}
