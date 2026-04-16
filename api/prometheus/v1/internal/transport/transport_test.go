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

package transport_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	json "github.com/json-iterator/go"

	"github.com/prometheus/client_golang/api/prometheus/v1/internal/transport"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

// ---------------------------------------------------------------------------
// Minimal api.Client implementation for unit tests
// ---------------------------------------------------------------------------

type stubClient struct {
	code     int
	body     []byte
	err      error
	lastReq  *http.Request
}

func (s *stubClient) URL(ep string, args map[string]string) *url.URL {
	return &url.URL{Host: "stub:9090", Path: ep}
}

func (s *stubClient) Do(_ context.Context, req *http.Request) (*http.Response, []byte, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, nil, s.err
	}
	resp := &http.Response{StatusCode: s.code}
	return resp, s.body, nil
}

// ---------------------------------------------------------------------------
// Transport.Do tests — API error handling
// ---------------------------------------------------------------------------

func jsonAPIResp(status string, data interface{}, errType types.ErrorType, errMsg string, warnings []string) []byte {
	r := transport.APIResponse{
		Status:    status,
		ErrorType: errType,
		Error:     errMsg,
		Warnings:  warnings,
	}
	if data != nil {
		b, _ := json.Marshal(data)
		r.Data = b
	}
	out, _ := json.Marshal(r)
	return out
}

func TestDo_NetworkError(t *testing.T) {
	want := errors.New("dial tcp: connection refused")
	c := &stubClient{err: want}
	_, _, _, got := transport.Do(c, context.Background(), &http.Request{})
	if got != want {
		t.Fatalf("expected network error %v, got %v", want, got)
	}
}

func TestDo_HTTPClientError(t *testing.T) {
	c := &stubClient{
		code: http.StatusNotFound,
		body: []byte("404 page not found"),
	}
	_, _, _, err := transport.Do(c, context.Background(), &http.Request{})
	var apiErr *types.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *types.Error, got %T: %v", err, err)
	}
	if apiErr.Type != types.ErrClient {
		t.Errorf("expected ErrClient, got %s", apiErr.Type)
	}
	if apiErr.Detail != "404 page not found" {
		t.Errorf("unexpected detail: %q", apiErr.Detail)
	}
}

func TestDo_HTTPServerError(t *testing.T) {
	c := &stubClient{
		code: http.StatusInternalServerError,
		body: []byte("internal error"),
	}
	_, _, _, err := transport.Do(c, context.Background(), &http.Request{})
	var apiErr *types.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *types.Error, got %T: %v", err, err)
	}
	if apiErr.Type != types.ErrServer {
		t.Errorf("expected ErrServer, got %s", apiErr.Type)
	}
}

func TestDo_APILevelError(t *testing.T) {
	body := jsonAPIResp("error", nil, types.ErrBadData, "bad query", nil)
	c := &stubClient{code: http.StatusUnprocessableEntity, body: body}

	_, _, _, err := transport.Do(c, context.Background(), &http.Request{})
	var apiErr *types.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *types.Error, got %T: %v", err, err)
	}
	if apiErr.Type != types.ErrBadData {
		t.Errorf("expected ErrBadData, got %s", apiErr.Type)
	}
	if apiErr.Msg != "bad query" {
		t.Errorf("expected msg %q, got %q", "bad query", apiErr.Msg)
	}
}

func TestDo_InconsistentBody_422SuccessStatus(t *testing.T) {
	// 422 with status:"success" is contradictory; transport must return an error.
	body := jsonAPIResp("success", `"test"`, "", "", nil)
	c := &stubClient{code: http.StatusUnprocessableEntity, body: body}

	_, _, _, err := transport.Do(c, context.Background(), &http.Request{})
	var apiErr *types.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *types.Error, got %T: %v", err, err)
	}
	if apiErr.Type != types.ErrBadResponse {
		t.Errorf("expected ErrBadResponse, got %s", apiErr.Type)
	}
}

func TestDo_200ErrorStatus(t *testing.T) {
	// 200 with status:"error" should still surface the API error.
	body := jsonAPIResp("error", `"test"`, types.ErrTimeout, "timed out", nil)
	c := &stubClient{code: http.StatusOK, body: body}

	_, _, _, err := transport.Do(c, context.Background(), &http.Request{})
	var apiErr *types.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *types.Error, got %T: %v", err, err)
	}
	if apiErr.Type != types.ErrTimeout {
		t.Errorf("expected ErrTimeout, got %s", apiErr.Type)
	}
}

func TestDo_WarningsPropagated(t *testing.T) {
	body := jsonAPIResp("error", `"test"`, types.ErrTimeout, "timed out", []string{"w1", "w2"})
	c := &stubClient{code: http.StatusOK, body: body}

	_, _, warnings, _ := transport.Do(c, context.Background(), &http.Request{})
	if len(warnings) != 2 || warnings[0] != "w1" || warnings[1] != "w2" {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestDo_BadJSON(t *testing.T) {
	c := &stubClient{code: http.StatusUnprocessableEntity, body: []byte("not json")}
	_, _, _, err := transport.Do(c, context.Background(), &http.Request{})
	var apiErr *types.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *types.Error, got %T: %v", err, err)
	}
	if apiErr.Type != types.ErrBadResponse {
		t.Errorf("expected ErrBadResponse, got %s", apiErr.Type)
	}
}

func TestDo_NoContent(t *testing.T) {
	c := &stubClient{code: http.StatusNoContent, body: nil}
	_, _, warnings, err := transport.Do(c, context.Background(), &http.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if warnings != nil {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
}

// ---------------------------------------------------------------------------
// Transport.DoGetFallback tests — POST-with-GET-fallback strategy
// ---------------------------------------------------------------------------

// httpStubClient wraps a real HTTP test server so we can verify actual HTTP
// method dispatch.
type httpStubClient struct {
	client http.Client
}

func (h *httpStubClient) URL(_ string, _ map[string]string) *url.URL { return nil }

func (h *httpStubClient) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	resp, err := h.client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	buf := make([]byte, 0, 256)
	tmp := make([]byte, 256)
	for {
		n, rerr := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if rerr != nil {
			break
		}
	}
	return resp, buf, nil
}

func startServer(t *testing.T) (*httptest.Server, *url.URL) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		type payload struct {
			Method string `json:"Method"`
			Values string `json:"Values"`
		}
		data, _ := json.Marshal(payload{Method: req.Method, Values: req.Form.Encode()})
		resp := transport.APIResponse{Status: "success"}
		resp.Data = data
		body, _ := json.Marshal(resp)

		switch req.URL.Path {
		case "/block405":
			if req.Method == http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				w.Write(body)
				return
			}
		case "/block501":
			if req.Method == http.MethodPost {
				w.WriteHeader(http.StatusNotImplemented)
				w.Write(body)
				return
			}
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return srv, u
}

func TestDoGetFallback_PostSucceeds(t *testing.T) {
	srv, u := startServer(t)
	c := &httpStubClient{client: *srv.Client()}
	args := url.Values{"k": []string{"v"}}

	_, body, _, err := transport.DoGetFallback(c, context.Background(), u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct{ Method, Values string }
	if jsonErr := json.Unmarshal(body, &got); jsonErr != nil {
		t.Fatalf("unmarshal: %v", jsonErr)
	}
	if got.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", got.Method)
	}
	if got.Values != args.Encode() {
		t.Errorf("values mismatch: want %s got %s", args.Encode(), got.Values)
	}
}

func TestDoGetFallback_405FallsBackToGet(t *testing.T) {
	srv, u := startServer(t)
	c := &httpStubClient{client: *srv.Client()}
	u.Path = "/block405"
	args := url.Values{"k": []string{"v"}}

	_, body, _, err := transport.DoGetFallback(c, context.Background(), u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct{ Method, Values string }
	if jsonErr := json.Unmarshal(body, &got); jsonErr != nil {
		t.Fatalf("unmarshal: %v", jsonErr)
	}
	if got.Method != http.MethodGet {
		t.Errorf("expected fallback GET, got %s", got.Method)
	}
	if got.Values != args.Encode() {
		t.Errorf("values mismatch: want %s got %s", args.Encode(), got.Values)
	}
}

func TestDoGetFallback_501FallsBackToGet(t *testing.T) {
	srv, u := startServer(t)
	c := &httpStubClient{client: *srv.Client()}
	u.Path = "/block501"
	args := url.Values{"k": []string{"v"}}

	_, body, _, err := transport.DoGetFallback(c, context.Background(), u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got struct{ Method, Values string }
	if jsonErr := json.Unmarshal(body, &got); jsonErr != nil {
		t.Fatalf("unmarshal: %v", jsonErr)
	}
	if got.Method != http.MethodGet {
		t.Errorf("expected fallback GET, got %s", got.Method)
	}
}
