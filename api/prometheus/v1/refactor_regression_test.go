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

func TestTransportFallback501Regression(t *testing.T) {
	args := url.Values{"match[]": {"up", "process_start_time_seconds"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		if req.Method == http.MethodPost {
			http.Error(w, `{"status":"success","data":{"method":"POST"}}`, http.StatusNotImplemented)
			return
		}
		body, err := json.Marshal(&apiResponse{
			Status: "success",
			Data:   json.RawMessage(`{"method":"` + req.Method + `","values":"` + req.Form.Encode() + `"}`),
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Write(body)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := &apiClientImpl{client: &httpTestClient{client: *server.Client()}}

	_, body, _, err := client.DoGetFallback(context.Background(), u, args)
	if err != nil {
		t.Fatalf("DoGetFallback returned error: %v", err)
	}

	var got struct {
		Method string `json:"method"`
		Values string `json:"values"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.Method != http.MethodGet {
		t.Fatalf("expected GET fallback, got %s", got.Method)
	}
	if got.Values != args.Encode() {
		t.Fatalf("expected encoded args %q, got %q", args.Encode(), got.Values)
	}
}

func TestAPIErrorMappingRegression(t *testing.T) {
	client := &apiClientImpl{client: &testClient{
		T:   t,
		ch:  make(chan apiClientTest, 1),
		req: &http.Request{},
	}}
	tc := client.client.(*testClient)

	tc.ch <- apiClientTest{
		code:     http.StatusInternalServerError,
		response: "boom",
	}
	_, _, _, err := client.Do(context.Background(), tc.req)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiErr.Type != ErrServer || apiErr.Detail != "boom" {
		t.Fatalf("unexpected mapped error: %#v", apiErr)
	}

	tc.ch <- apiClientTest{
		code: http.StatusUnprocessableEntity,
		response: &apiResponse{
			Status:    "error",
			ErrorType: ErrTimeout,
			Error:     "timed out",
			Data:      json.RawMessage(`null`),
		},
	}
	_, _, _, err = client.Do(context.Background(), tc.req)
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiErr.Type != ErrTimeout || apiErr.Msg != "timed out" {
		t.Fatalf("unexpected API-level error mapping: %#v", apiErr)
	}
}

func TestCodecRegistrationRegression(t *testing.T) {
	payload, err := json.Marshal(model.SamplePair{
		Timestamp: 1500,
		Value:     42,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `[1.500,"42"]` {
		t.Fatalf("unexpected encoded sample pair: %s", payload)
	}

	var decoded model.SamplePair
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Timestamp != 1500 || decoded.Value != 42 {
		t.Fatalf("unexpected decoded sample pair: %#v", decoded)
	}

	streamPayload, err := json.Marshal(model.SampleStream{
		Metric: model.Metric{"__name__": "up"},
		Values: []model.SamplePair{{Timestamp: 1, Value: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(streamPayload), `"values":[[0.001,"1"]]`) {
		t.Fatalf("sample stream codec registration not applied: %s", streamPayload)
	}
}
