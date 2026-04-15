package v1

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/endpoints"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/transport"
)

type httpAPI struct {
	client apiClient
}

type apiClient interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, Warnings, error)
	DoGetFallback(context.Context, *url.URL, url.Values) (*http.Response, []byte, Warnings, error)
}

type apiClientImpl struct {
	client api.Client
}

type apiResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType ErrorType       `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings,omitempty"`
}

func (h *apiClientImpl) URL(ep string, args map[string]string) *url.URL {
	return h.client.URL(ep, args)
}

func (h *apiClientImpl) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, Warnings, error) {
	resp, body, warnings, err := transport.Do(ctx, h.client, req, newError)
	return resp, body, Warnings(warnings), err
}

func (h *apiClientImpl) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error) {
	resp, body, warnings, err := transport.DoGetFallback(ctx, h.client, u, args, newError)
	return resp, body, Warnings(warnings), err
}

func newError(errorType, msg, detail string) error {
	return &Error{
		Type:   ErrorType(errorType),
		Msg:    msg,
		Detail: detail,
	}
}

func (h *httpAPI) Alerts(ctx context.Context) (AlertsResult, error) {
	u := h.client.URL(endpoints.Alerts, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return AlertsResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return AlertsResult{}, err
	}
	var res AlertsResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) AlertManagers(ctx context.Context) (AlertManagersResult, error) {
	u := h.client.URL(endpoints.AlertManagers, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return AlertManagersResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return AlertManagersResult{}, err
	}
	var res AlertManagersResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) CleanTombstones(ctx context.Context) error {
	u := h.client.URL(endpoints.CleanTombstones, nil)
	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Config(ctx context.Context) (ConfigResult, error) {
	u := h.client.URL(endpoints.Config, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return ConfigResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return ConfigResult{}, err
	}
	var res ConfigResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) DeleteSeries(ctx context.Context, matches []string, startTime, endTime time.Time) error {
	u := h.client.URL(endpoints.DeleteSeries, nil)
	q := u.Query()
	endpoints.AddMatchers(q, matches)
	endpoints.SetOptionalTime(q, "start", startTime)
	endpoints.SetOptionalTime(q, "end", endTime)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Flags(ctx context.Context) (FlagsResult, error) {
	u := h.client.URL(endpoints.Flags, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return FlagsResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return FlagsResult{}, err
	}
	var res FlagsResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Buildinfo(ctx context.Context) (BuildinfoResult, error) {
	u := h.client.URL(endpoints.Buildinfo, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return BuildinfoResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return BuildinfoResult{}, err
	}
	var res BuildinfoResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Runtimeinfo(ctx context.Context) (RuntimeinfoResult, error) {
	u := h.client.URL(endpoints.Runtimeinfo, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return RuntimeinfoResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return RuntimeinfoResult{}, err
	}
	var res RuntimeinfoResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) LabelNames(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelNames, Warnings, error) {
	u := h.client.URL(endpoints.Labels, nil)
	q := addOptionalURLParams(u.Query(), opts)
	endpoints.SetOptionalTime(q, "start", startTime)
	endpoints.SetOptionalTime(q, "end", endTime)
	endpoints.AddMatchers(q, matches)

	_, body, w, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, w, err
	}
	var res model.LabelNames
	err = json.Unmarshal(body, &res)
	return res, w, err
}

func (h *httpAPI) LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error) {
	u := h.client.URL(endpoints.LabelValues, map[string]string{"name": label})
	q := addOptionalURLParams(u.Query(), opts)
	endpoints.SetOptionalTime(q, "start", startTime)
	endpoints.SetOptionalTime(q, "end", endTime)
	endpoints.AddMatchers(q, matches)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	_, body, w, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, w, err
	}
	var res model.LabelValues
	err = json.Unmarshal(body, &res)
	return res, w, err
}

func (h *httpAPI) Query(ctx context.Context, query string, ts time.Time, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(endpoints.Query, nil)
	q := addOptionalURLParams(u.Query(), opts)
	q.Set("query", query)
	endpoints.SetOptionalTime(q, "time", ts)

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var qres queryResult
	return qres.v, warnings, json.Unmarshal(body, &qres)
}

func (h *httpAPI) QueryRange(ctx context.Context, query string, r Range, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(endpoints.QueryRange, nil)
	q := addOptionalURLParams(u.Query(), opts)
	q.Set("query", query)
	q.Set("start", endpoints.FormatTime(r.Start))
	q.Set("end", endpoints.FormatTime(r.End))
	q.Set("step", strconv.FormatFloat(r.Step.Seconds(), 'f', -1, 64))

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var qres queryResult
	return qres.v, warnings, json.Unmarshal(body, &qres)
}

func (h *httpAPI) Series(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) ([]model.LabelSet, Warnings, error) {
	u := h.client.URL(endpoints.Series, nil)
	q := addOptionalURLParams(u.Query(), opts)
	endpoints.AddMatchers(q, matches)
	endpoints.SetOptionalTime(q, "start", startTime)
	endpoints.SetOptionalTime(q, "end", endTime)

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}
	var res []model.LabelSet
	return res, warnings, json.Unmarshal(body, &res)
}

func (h *httpAPI) Snapshot(ctx context.Context, skipHead bool) (SnapshotResult, error) {
	u := h.client.URL(endpoints.Snapshot, nil)
	q := u.Query()
	q.Set("skip_head", strconv.FormatBool(skipHead))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return SnapshotResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return SnapshotResult{}, err
	}
	var res SnapshotResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Rules(ctx context.Context, matches []string) (RulesResult, error) {
	u := h.client.URL(endpoints.Rules, nil)
	q := u.Query()
	endpoints.AddMatchers(q, matches)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return RulesResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return RulesResult{}, err
	}
	var res RulesResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Targets(ctx context.Context) (TargetsResult, error) {
	u := h.client.URL(endpoints.Targets, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return TargetsResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TargetsResult{}, err
	}
	var res TargetsResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) TargetsMetadata(ctx context.Context, matchTarget, metric, limit string) ([]MetricMetadata, error) {
	u := h.client.URL(endpoints.TargetsMetadata, nil)
	q := u.Query()
	q.Set("match_target", matchTarget)
	q.Set("metric", metric)
	q.Set("limit", limit)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	var res []MetricMetadata
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Metadata(ctx context.Context, metric, limit string) (map[string][]Metadata, error) {
	u := h.client.URL(endpoints.Metadata, nil)
	q := u.Query()
	q.Set("metric", metric)
	q.Set("limit", limit)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	var res map[string][]Metadata
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) TSDB(ctx context.Context, opts ...Option) (TSDBResult, error) {
	u := h.client.URL(endpoints.TSDB, nil)
	u.RawQuery = addOptionalURLParams(u.Query(), opts).Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return TSDBResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TSDBResult{}, err
	}
	var res TSDBResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) TSDBBlocks(ctx context.Context) (TSDBBlocksResult, error) {
	u := h.client.URL(endpoints.TSDBBlocks, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return TSDBBlocksResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TSDBBlocksResult{}, err
	}
	var res TSDBBlocksResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) WalReplay(ctx context.Context) (WalReplayStatus, error) {
	u := h.client.URL(endpoints.WalReplay, nil)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return WalReplayStatus{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return WalReplayStatus{}, err
	}
	var res WalReplayStatus
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) QueryExemplars(ctx context.Context, query string, startTime, endTime time.Time) ([]ExemplarQueryResult, error) {
	u := h.client.URL(endpoints.QueryExemplars, nil)
	q := u.Query()
	q.Set("query", query)
	endpoints.SetOptionalTime(q, "start", startTime)
	endpoints.SetOptionalTime(q, "end", endTime)

	_, body, _, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, err
	}
	var res []ExemplarQueryResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) FormatQuery(ctx context.Context, query string) (string, error) {
	u := h.client.URL(endpoints.FormatQuery, nil)
	q := u.Query()
	q.Set("query", query)
	_, body, _, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func formatTime(t time.Time) string {
	return endpoints.FormatTime(t)
}
