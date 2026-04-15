// Copyright 2026 The Prometheus Authors
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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	json "github.com/json-iterator/go"

	"github.com/prometheus/client_golang/api"
	v1types "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

type roundTripClient struct {
	do func(context.Context, *http.Request) (*http.Response, []byte, error)
}

func (c roundTripClient) URL(ep string, args map[string]string) *url.URL {
	return nil
}

func (c roundTripClient) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	return c.do(ctx, req)
}

func TestDoAPIErrorHandling(t *testing.T) {
	tests := []struct {
		name             string
		code             int
		response         interface{}
		expectedBody     string
		expectedWarnings v1types.Warnings
		expectedErr      *v1types.Error
	}{
		{
			name: "api error body maps through",
			code: http.StatusUnprocessableEntity,
			response: &APIResponse{
				Status:    "error",
				Data:      json.RawMessage(`null`),
				ErrorType: v1types.ErrBadData,
				Error:     "failed",
			},
			expectedErr: &v1types.Error{
				Type: v1types.ErrBadData,
				Msg:  "failed",
			},
		},
		{
			name:     "server error normalizes http failure",
			code:     http.StatusInternalServerError,
			response: "boom",
			expectedErr: &v1types.Error{
				Type:   v1types.ErrServer,
				Msg:    "server error: 500",
				Detail: "boom",
			},
		},
		{
			name: "success status on api error code is bad response",
			code: http.StatusBadRequest,
			response: &APIResponse{
				Status: "success",
				Data:   json.RawMessage(`"value"`),
			},
			expectedErr: &v1types.Error{
				Type: v1types.ErrBadResponse,
				Msg:  "inconsistent body for response code",
			},
		},
		{
			name: "warnings survive api error envelopes",
			code: http.StatusOK,
			response: &APIResponse{
				Status:    "error",
				Data:      json.RawMessage(`"value"`),
				ErrorType: v1types.ErrTimeout,
				Error:     "timed out",
				Warnings:  []string{"warn"},
			},
			expectedWarnings: v1types.Warnings{"warn"},
			expectedErr: &v1types.Error{
				Type: v1types.ErrTimeout,
				Msg:  "timed out",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(roundTripClient{
				do: func(context.Context, *http.Request) (*http.Response, []byte, error) {
					var body []byte
					switch v := test.response.(type) {
					case string:
						body = []byte(v)
					default:
						var err error
						body, err = json.Marshal(v)
						if err != nil {
							t.Fatal(err)
						}
					}
					return &http.Response{StatusCode: test.code}, body, nil
				},
			})

			_, body, warnings, err := client.Do(context.Background(), &http.Request{})
			if !reflect.DeepEqual(test.expectedWarnings, warnings) {
				t.Fatalf("unexpected warnings: want %v, got %v", test.expectedWarnings, warnings)
			}
			if test.expectedBody != "" && string(body) != test.expectedBody {
				t.Fatalf("unexpected body: want %q, got %q", test.expectedBody, string(body))
			}
			if test.expectedErr == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got none")
			}
			gotErr, ok := err.(*v1types.Error)
			if !ok {
				t.Fatalf("expected *types.Error, got %T", err)
			}
			if !reflect.DeepEqual(test.expectedErr, gotErr) {
				t.Fatalf("unexpected error: want %#v, got %#v", test.expectedErr, gotErr)
			}
		})
	}
}

type httpClientAdapter struct {
	client *http.Client
}

func (c *httpClientAdapter) URL(ep string, args map[string]string) *url.URL {
	return nil
}

func (c *httpClientAdapter) Do(_ context.Context, req *http.Request) (*http.Response, []byte, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return resp, body, err
}

func TestDoGetFallbackUsesGetFor405And501(t *testing.T) {
	values := url.Values{"a": []string{"1", "2"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		payload, err := json.Marshal(map[string]string{
			"method": req.Method,
			"form":   req.Form.Encode(),
		})
		if err != nil {
			t.Fatal(err)
		}
		body, err := json.Marshal(&APIResponse{Status: "success", Data: payload})
		if err != nil {
			t.Fatal(err)
		}
		if req.Method == http.MethodPost && (req.URL.Path == "/405" || req.URL.Path == "/501") {
			if req.URL.Path == "/405" {
				http.Error(w, string(body), http.StatusMethodNotAllowed)
				return
			}
			http.Error(w, string(body), http.StatusNotImplemented)
			return
		}
		w.Write(body)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(&httpClientAdapter{client: server.Client()})
	for _, path := range []string{"", "/405", "/501"} {
		t.Run(path, func(t *testing.T) {
			u := *baseURL
			u.Path = path
			_, body, _, err := client.DoGetFallback(context.Background(), &u, values)
			if err != nil {
				t.Fatalf("DoGetFallback failed: %v", err)
			}
			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			wantMethod := http.MethodPost
			if path == "/405" || path == "/501" {
				wantMethod = http.MethodGet
			}
			if payload["method"] != wantMethod {
				t.Fatalf("unexpected method: want %s, got %s", wantMethod, payload["method"])
			}
			if payload["form"] != values.Encode() {
				t.Fatalf("unexpected form: want %s, got %s", values.Encode(), payload["form"])
			}
		})
	}
}

var _ api.Client = roundTripClient{}
