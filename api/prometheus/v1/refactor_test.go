package v1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

func TestDoGetFallbackDoesNotFallbackOnAPIError(t *testing.T) {
	var methods []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		methods = append(methods, req.Method)
		body, err := json.Marshal(&apiResponse{
			Status:    "error",
			ErrorType: ErrBadData,
			Error:     "bad query",
		})
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &apiClientImpl{client: &httpTestClient{client: *server.Client()}}
	_, _, _, err = client.DoGetFallback(context.Background(), u, url.Values{"query": []string{"up"}})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected v1.Error, got %T", err)
	}
	if apiErr.Type != ErrBadData || apiErr.Msg != "bad query" {
		t.Fatalf("unexpected api error: %#v", apiErr)
	}
	if !reflect.DeepEqual(methods, []string{http.MethodPost}) {
		t.Fatalf("expected only POST attempt, got %v", methods)
	}
}

func TestAPIClientDoMapsHTTPAndAPIErrors(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		body    interface{}
		wantErr *Error
	}{
		{
			name: "http server error",
			code: http.StatusInternalServerError,
			body: "boom",
			wantErr: &Error{
				Type:   ErrServer,
				Msg:    "server error: 500",
				Detail: "boom",
			},
		},
		{
			name: "api level error",
			code: http.StatusOK,
			body: &apiResponse{
				Status:    "error",
				ErrorType: ErrTimeout,
				Error:     "timed out",
			},
			wantErr: &Error{
				Type: ErrTimeout,
				Msg:  "timed out",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &apiClientImpl{client: &stubAPIClient{
				resp: &http.Response{StatusCode: tt.code},
				body: tt.body,
			}}
			_, _, _, err := client.Do(context.Background(), &http.Request{})
			if err == nil {
				t.Fatal("expected error")
			}

			var apiErr *Error
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected v1.Error, got %T", err)
			}
			if !reflect.DeepEqual(apiErr, tt.wantErr) {
				t.Fatalf("expected %#v, got %#v", tt.wantErr, apiErr)
			}
		})
	}
}

func TestCustomCodecRegistrationRoundTrip(t *testing.T) {
	stream := model.SampleStream{
		Metric: model.Metric{"__name__": "up", "job": "prometheus"},
		Values: []model.SamplePair{
			{Timestamp: 1234, Value: 42},
		},
	}

	b, err := json.Marshal(stream)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"values":[[1.234,"42"]]`) {
		t.Fatalf("expected custom sample stream encoding, got %s", string(b))
	}

	var got model.SampleStream
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stream, got) {
		t.Fatalf("expected %#v, got %#v", stream, got)
	}
}

type stubAPIClient struct {
	resp *http.Response
	body interface{}
}

func (c *stubAPIClient) URL(_ string, _ map[string]string) *url.URL {
	return &url.URL{}
}

func (c *stubAPIClient) Do(_ context.Context, _ *http.Request) (*http.Response, []byte, error) {
	switch body := c.body.(type) {
	case string:
		return c.resp, []byte(body), nil
	default:
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		return c.resp, b, nil
	}
}
