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
	"testing"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

func TestDoGetFallbackRegression(t *testing.T) {
	values := url.Values{"query": []string{"up"}, "match[]": []string{"up"}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		respBody, _ := json.Marshal(map[string]string{
			"method": req.Method,
			"query":  req.Form.Encode(),
		})
		body, _ := json.Marshal(&apiResponse{
			Status: "success",
			Data:   respBody,
		})

		switch req.URL.Path {
		case "/fallback-405":
			if req.Method == http.MethodPost {
				http.Error(w, string(body), http.StatusMethodNotAllowed)
				return
			}
		case "/fallback-501":
			if req.Method == http.MethodPost {
				http.Error(w, string(body), http.StatusNotImplemented)
				return
			}
		}

		_, _ = w.Write(body)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	api := &apiClientImpl{client: &httpTestClient{client: *server.Client()}}

	for _, test := range []struct {
		name   string
		path   string
		method string
	}{
		{name: "post stays post", path: "", method: http.MethodPost},
		{name: "405 falls back", path: "/fallback-405", method: http.MethodGet},
		{name: "501 falls back", path: "/fallback-501", method: http.MethodGet},
	} {
		t.Run(test.name, func(t *testing.T) {
			u := *baseURL
			u.Path = test.path

			_, body, _, err := api.DoGetFallback(context.Background(), &u, values)
			if err != nil {
				t.Fatalf("DoGetFallback returned error: %v", err)
			}

			var got struct {
				Method string `json:"method"`
				Query  string `json:"query"`
			}
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if got.Method != test.method {
				t.Fatalf("expected method %s, got %s", test.method, got.Method)
			}
			if got.Query != values.Encode() {
				t.Fatalf("expected values %q, got %q", values.Encode(), got.Query)
			}
		})
	}
}

func TestAPIClientDoErrorRegression(t *testing.T) {
	req := &http.Request{}
	tc := &testClient{
		T:   t,
		ch:  make(chan apiClientTest, 1),
		req: req,
	}
	client := &apiClientImpl{client: tc}

	t.Run("http status maps to server error", func(t *testing.T) {
		tc.ch <- apiClientTest{
			code:     http.StatusInternalServerError,
			response: "boom",
		}

		_, _, _, err := client.Do(context.Background(), req)
		apiErr := &Error{}
		if err == nil || !errors.As(err, &apiErr) {
			t.Fatalf("expected v1.Error, got %v", err)
		}
		if apiErr.Type != ErrServer || apiErr.Msg != "server error: 500" || apiErr.Detail != "boom" {
			t.Fatalf("unexpected error: %#v", apiErr)
		}
	})

	t.Run("api envelope error stays api error", func(t *testing.T) {
		tc.ch <- apiClientTest{
			code: http.StatusUnprocessableEntity,
			response: &apiResponse{
				Status:    "error",
				Data:      json.RawMessage(`null`),
				ErrorType: ErrBadData,
				Error:     "invalid matcher",
			},
		}

		_, _, _, err := client.Do(context.Background(), req)
		apiErr := &Error{}
		if err == nil || !errors.As(err, &apiErr) {
			t.Fatalf("expected v1.Error, got %v", err)
		}
		if apiErr.Type != ErrBadData || apiErr.Msg != "invalid matcher" || apiErr.Detail != "" {
			t.Fatalf("unexpected error: %#v", apiErr)
		}
	})
}

func TestCustomCodecRegistrationRegression(t *testing.T) {
	point := model.SamplePair{
		Timestamp: 1234,
		Value:     model.SampleValue(5.5),
	}

	b, err := json.Marshal(point)
	if err != nil {
		t.Fatalf("marshal sample pair: %v", err)
	}
	if string(b) != `[1.234,"5.5"]` {
		t.Fatalf("unexpected sample pair json: %s", string(b))
	}

	var got model.SamplePair
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal sample pair: %v", err)
	}
	if got != point {
		t.Fatalf("roundtrip mismatch: want %#v, got %#v", point, got)
	}
}
