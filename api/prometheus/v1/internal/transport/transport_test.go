package transport

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	json "github.com/json-iterator/go"

	internaltypes "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

type testClient struct {
	do func(context.Context, *http.Request) (*http.Response, []byte, error)
}

func (c testClient) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	return c.do(ctx, req)
}

type testError struct {
	errorType string
	msg       string
	detail    string
}

func (e *testError) Error() string {
	return e.errorType + ": " + e.msg
}

func newTestError(errorType, msg, detail string) error {
	return &testError{errorType: errorType, msg: msg, detail: detail}
}

func TestDoGetFallbackOn405And501(t *testing.T) {
	t.Parallel()

	for _, statusCode := range []int{http.StatusMethodNotAllowed, http.StatusNotImplemented} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()

			var methods []string
			client := testClient{
				do: func(_ context.Context, req *http.Request) (*http.Response, []byte, error) {
					methods = append(methods, req.Method)
					if req.Method == http.MethodPost {
						body, _ := json.Marshal(internaltypes.APIResponse{Status: "error", ErrorType: "client_error", Error: "retry with GET"})
						return &http.Response{StatusCode: statusCode}, body, nil
					}

					body, _ := json.Marshal(internaltypes.APIResponse{
						Status:   "success",
						Data:     json.RawMessage(`"ok"`),
						Warnings: []string{"fallback"},
					})
					return &http.Response{StatusCode: http.StatusOK}, body, nil
				},
			}

			u := &url.URL{Scheme: "http", Host: "example.invalid", Path: "/api/v1/query"}
			args := url.Values{"query": []string{"up"}}

			_, body, warnings, err := DoGetFallback(context.Background(), client, u, args, newTestError)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(body) != `"ok"` {
				t.Fatalf("unexpected body: %s", body)
			}
			if !reflect.DeepEqual(warnings, []string{"fallback"}) {
				t.Fatalf("unexpected warnings: %#v", warnings)
			}
			if !reflect.DeepEqual(methods, []string{http.MethodPost, http.MethodGet}) {
				t.Fatalf("unexpected method sequence: %#v", methods)
			}
			if got := u.RawQuery; got != args.Encode() {
				t.Fatalf("unexpected fallback query: %q", got)
			}
		})
	}
}

func TestDoNormalizesAPIAndHTTPErrorResponses(t *testing.T) {
	t.Parallel()

	t.Run("http error", func(t *testing.T) {
		t.Parallel()

		client := testClient{
			do: func(context.Context, *http.Request) (*http.Response, []byte, error) {
				return &http.Response{StatusCode: http.StatusInternalServerError}, []byte("boom"), nil
			},
		}

		_, _, _, err := Do(context.Background(), client, httptestRequest(), newTestError)
		var apiErr *testError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected testError, got %T", err)
		}
		if apiErr.errorType != "server_error" || apiErr.msg != "server error: 500" || apiErr.detail != "boom" {
			t.Fatalf("unexpected error: %#v", apiErr)
		}
	})

	t.Run("api error", func(t *testing.T) {
		t.Parallel()

		client := testClient{
			do: func(context.Context, *http.Request) (*http.Response, []byte, error) {
				body, _ := json.Marshal(internaltypes.APIResponse{
					Status:    "error",
					Data:      json.RawMessage(`"payload"`),
					ErrorType: "bad_data",
					Error:     "bad query",
					Warnings:  []string{"warn"},
				})
				return &http.Response{StatusCode: http.StatusBadRequest}, body, nil
			},
		}

		_, body, warnings, err := Do(context.Background(), client, httptestRequest(), newTestError)
		var apiErr *testError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected testError, got %T", err)
		}
		if apiErr.errorType != "bad_data" || apiErr.msg != "bad query" || apiErr.detail != "" {
			t.Fatalf("unexpected error: %#v", apiErr)
		}
		if string(body) != `"payload"` {
			t.Fatalf("unexpected body: %s", body)
		}
		if !reflect.DeepEqual(warnings, []string{"warn"}) {
			t.Fatalf("unexpected warnings: %#v", warnings)
		}
	})
}

func httptestRequest() *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "http://example.invalid", strings.NewReader(""))
	return req
}
