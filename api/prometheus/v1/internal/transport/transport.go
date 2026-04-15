package transport

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	json "github.com/json-iterator/go"

	internaltypes "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

type Client interface {
	Do(context.Context, *http.Request) (*http.Response, []byte, error)
}

type ErrorFactory func(errorType, msg, detail string) error

func Do(ctx context.Context, client Client, req *http.Request, newError ErrorFactory) (*http.Response, []byte, []string, error) {
	resp, body, err := client.Do(ctx, req)
	if err != nil {
		return resp, body, nil, err
	}

	code := resp.StatusCode
	if code/100 != 2 && !apiError(code) {
		errorType, errorMsg := errorTypeAndMsgFor(resp)
		return resp, body, nil, newError(errorType, errorMsg, string(body))
	}

	var result internaltypes.APIResponse
	if code != http.StatusNoContent {
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return resp, body, nil, newError("bad_response", jsonErr.Error(), "")
		}
	}

	if apiError(code) && result.Status == "success" {
		err = newError("bad_response", "inconsistent body for response code", "")
	}
	if result.Status == "error" {
		err = newError(result.ErrorType, result.Error, "")
	}

	return resp, []byte(result.Data), result.Warnings, err
}

func DoGetFallback(ctx context.Context, client Client, u *url.URL, args url.Values, newError ErrorFactory) (*http.Response, []byte, []string, error) {
	encodedArgs := args.Encode()
	req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(encodedArgs))
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header["Idempotency-Key"] = nil

	resp, body, warnings, err := Do(ctx, client, req, newError)
	if resp != nil && (resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented) {
		u.RawQuery = encodedArgs
		req, err = http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, nil, warnings, err
		}
		return Do(ctx, client, req, newError)
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
