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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/transport"
)

// newTestClient wraps an httptest.Server URL as an api.Client.
func newTestClient(t *testing.T, serverURL string) api.Client {
	t.Helper()
	cfg := api.Config{Address: serverURL}
	c, err := api.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestDoGetFallback_PostSucceeds verifies that a POST returning 200 is not
// retried as GET.
func TestDoGetFallback_PostSucceeds(t *testing.T) {
	var seen []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":""}`))
	}))
	defer ts.Close()

	c := newTestClient(t, ts.URL)
	u, _ := url.Parse(ts.URL + "/api/v1/query")
	args := url.Values{"query": []string{"up"}}

	resp, _, err := transport.DoGetFallback(context.Background(), c, u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if len(seen) != 1 || seen[0] != http.MethodPost {
		t.Errorf("expected exactly one POST, got %v", seen)
	}
}

// TestDoGetFallback_405FallsBackToGET verifies that a 405 POST response
// triggers a retry as GET.
func TestDoGetFallback_405FallsBackToGET(t *testing.T) {
	var seen []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method)
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":""}`))
	}))
	defer ts.Close()

	c := newTestClient(t, ts.URL)
	u, _ := url.Parse(ts.URL + "/api/v1/query")
	args := url.Values{"query": []string{"up"}}

	resp, _, err := transport.DoGetFallback(context.Background(), c, u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from GET fallback, got %d", resp.StatusCode)
	}
	if len(seen) != 2 || seen[0] != http.MethodPost || seen[1] != http.MethodGet {
		t.Errorf("expected POST then GET, got %v", seen)
	}
}

// TestDoGetFallback_501FallsBackToGET verifies that a 501 POST response
// triggers a retry as GET.
func TestDoGetFallback_501FallsBackToGET(t *testing.T) {
	var seen []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method)
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNotImplemented)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":""}`))
	}))
	defer ts.Close()

	c := newTestClient(t, ts.URL)
	u, _ := url.Parse(ts.URL + "/api/v1/query")
	args := url.Values{"query": []string{"up"}}

	resp, _, err := transport.DoGetFallback(context.Background(), c, u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from GET fallback, got %d", resp.StatusCode)
	}
	if len(seen) != 2 || seen[0] != http.MethodPost || seen[1] != http.MethodGet {
		t.Errorf("expected POST then GET, got %v", seen)
	}
}

// TestDoGetFallback_ArgsPreservedInGET verifies that query parameters are
// correctly moved to the URL query string when falling back to GET.
func TestDoGetFallback_ArgsPreservedInGET(t *testing.T) {
	var getQuery url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		getQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":""}`))
	}))
	defer ts.Close()

	c := newTestClient(t, ts.URL)
	u, _ := url.Parse(ts.URL + "/api/v1/query")
	args := url.Values{
		"query": []string{"up{job=\"prometheus\"}"},
		"time":  []string{"1234567890"},
	}

	_, _, err := transport.DoGetFallback(context.Background(), c, u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getQuery.Get("query") != `up{job="prometheus"}` {
		t.Errorf("query param not preserved: %q", getQuery.Get("query"))
	}
	if getQuery.Get("time") != "1234567890" {
		t.Errorf("time param not preserved: %q", getQuery.Get("time"))
	}
}

// TestDoGetFallback_OtherErrorsNotRetried verifies that non-405/501 errors
// (e.g. 500) are returned without retrying as GET.
func TestDoGetFallback_OtherErrorsNotRetried(t *testing.T) {
	var seen []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(t, ts.URL)
	u, _ := url.Parse(ts.URL + "/api/v1/query")
	args := url.Values{"query": []string{"up"}}

	resp, _, err := transport.DoGetFallback(context.Background(), c, u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
	if len(seen) != 1 || seen[0] != http.MethodPost {
		t.Errorf("expected exactly one POST (no retry), got %v", seen)
	}
}

// TestDoGetFallback_PostHasCorrectContentType verifies the POST sets
// application/x-www-form-urlencoded.
func TestDoGetFallback_PostHasCorrectContentType(t *testing.T) {
	var ct string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			ct = r.Header.Get("Content-Type")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","data":""}`))
	}))
	defer ts.Close()

	c := newTestClient(t, ts.URL)
	u, _ := url.Parse(ts.URL + "/api/v1/query")
	args := url.Values{"query": []string{"up"}}

	_, _, err := transport.DoGetFallback(context.Background(), c, u, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %q", ct)
	}
}
