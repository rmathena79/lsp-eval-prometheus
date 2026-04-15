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
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

func TestTransportFallbackStatuses(t *testing.T) {
	tests := []struct {
		name           string
		postStatusCode int
		wantMethod     string
		wantErr        string
	}{
		{
			name:           "falls back on 405",
			postStatusCode: http.StatusMethodNotAllowed,
			wantMethod:     http.MethodGet,
		},
		{
			name:           "falls back on 501",
			postStatusCode: http.StatusNotImplemented,
			wantMethod:     http.MethodGet,
		},
		{
			name:           "does not fall back on 500",
			postStatusCode: http.StatusInternalServerError,
			wantMethod:     http.MethodPost,
			wantErr:        "server_error: server error: 500",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var methods []string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				methods = append(methods, req.Method)

				req.ParseForm()
				payload, err := json.Marshal(map[string]string{
					"method": req.Method,
					"query":  req.Form.Encode(),
				})
				if err != nil {
					t.Fatal(err)
				}

				body, err := json.Marshal(&apiResponse{
					Status: "success",
					Data:   payload,
				})
				if err != nil {
					t.Fatal(err)
				}

				if req.Method == http.MethodPost {
					w.WriteHeader(test.postStatusCode)
					_, _ = w.Write(body)
					return
				}

				_, _ = w.Write(body)
			}))
			defer server.Close()

			parsedURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}

			client := &apiClientImpl{client: &httpTestClient{client: *server.Client()}}
			args := url.Values{"match[]": {"up"}}
			_, body, _, err := client.DoGetFallback(context.Background(), parsedURL, args)

			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q", test.wantErr)
				}
				if err.Error() != test.wantErr {
					t.Fatalf("unexpected error: want %q, got %q", test.wantErr, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if methods[0] != http.MethodPost {
				t.Fatalf("expected first method POST, got %s", methods[0])
			}
			if methods[len(methods)-1] != test.wantMethod {
				t.Fatalf("expected final method %s, got %s", test.wantMethod, methods[len(methods)-1])
			}

			if test.wantErr == "" {
				var response map[string]string
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatal(err)
				}
				if response["method"] != test.wantMethod {
					t.Fatalf("expected response method %s, got %s", test.wantMethod, response["method"])
				}
				if response["query"] != args.Encode() {
					t.Fatalf("expected query %q, got %q", args.Encode(), response["query"])
				}
			}
		})
	}
}

func TestAPIErrorHandlingFocused(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		response    interface{}
		wantErr     *Error
		wantWarning Warnings
	}{
		{
			name:       "http error keeps detail",
			statusCode: http.StatusInternalServerError,
			response:   "backend exploded",
			wantErr: &Error{
				Type:   ErrServer,
				Msg:    "server error: 500",
				Detail: "backend exploded",
			},
		},
		{
			name:       "api envelope error wins on 200",
			statusCode: http.StatusOK,
			response: &apiResponse{
				Status:    "error",
				ErrorType: ErrTimeout,
				Error:     "query timed out",
				Data:      json.RawMessage(`"ignored"`),
				Warnings:  []string{"slow backend"},
			},
			wantErr: &Error{
				Type: ErrTimeout,
				Msg:  "query timed out",
			},
			wantWarning: Warnings{"slow backend"},
		},
		{
			name:       "unprocessable success body is normalized",
			statusCode: http.StatusUnprocessableEntity,
			response: &apiResponse{
				Status: "success",
				Data:   json.RawMessage(`"oops"`),
			},
			wantErr: &Error{
				Type: ErrBadResponse,
				Msg:  "inconsistent body for response code",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tc := &testClient{
				T:   t,
				ch:  make(chan apiClientTest, 1),
				req: &http.Request{},
			}
			tc.ch <- apiClientTest{code: test.statusCode, response: test.response}

			client := &apiClientImpl{client: tc}
			_, _, warnings, err := client.Do(context.Background(), tc.req)
			if !errors.As(err, &test.wantErr) && err == nil {
				t.Fatal("expected error, got nil")
			}
			if err == nil || err.Error() != test.wantErr.Error() {
				t.Fatalf("unexpected error: want %v, got %v", test.wantErr, err)
			}
			if test.wantErr.Detail != "" {
				apiErr := &Error{}
				if !errors.As(err, &apiErr) {
					t.Fatalf("expected *Error, got %T", err)
				}
				if apiErr.Detail != test.wantErr.Detail {
					t.Fatalf("expected detail %q, got %q", test.wantErr.Detail, apiErr.Detail)
				}
			}
			if strings.Join(warnings, ",") != strings.Join(test.wantWarning, ",") {
				t.Fatalf("unexpected warnings: want %v, got %v", test.wantWarning, warnings)
			}
		})
	}
}

func TestCustomCodecBehaviorFocused(t *testing.T) {
	stream := model.SampleStream{
		Metric: model.Metric{"__name__": "up", "job": "prometheus"},
		Values: []model.SamplePair{
			{Timestamp: 1001, Value: model.SampleValue(12.5)},
		},
	}

	data, err := json.Marshal(stream)
	if err != nil {
		t.Fatal(err)
	}

	const want = `{"metric":{"__name__":"up","job":"prometheus"},"values":[[1.001,"12.5"]]}`
	if string(data) != want {
		t.Fatalf("unexpected JSON: want %s, got %s", want, string(data))
	}

	var decoded model.SampleStream
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if len(decoded.Values) != 1 || decoded.Values[0].Timestamp != 1001 || decoded.Values[0].Value != model.SampleValue(12.5) {
		t.Fatalf("unexpected round-trip result: %#v", decoded)
	}
	if decoded.Metric.String() != stream.Metric.String() {
		t.Fatalf("unexpected metric after round-trip: want %s, got %s", stream.Metric, decoded.Metric)
	}
}
