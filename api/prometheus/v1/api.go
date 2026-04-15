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
	"net/http"
	"net/url"
	"strconv"
	"time"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/codec"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/endpoints"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/transport"
	itypes "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

func init() {
	codec.Register()
}

// API provides bindings for Prometheus's v1 API.
type API interface {
	// Alerts returns a list of all active alerts.
	Alerts(ctx context.Context) (AlertsResult, error)
	// AlertManagers returns an overview of the current state of the Prometheus alert manager discovery.
	AlertManagers(ctx context.Context) (AlertManagersResult, error)
	// CleanTombstones removes the deleted data from disk and cleans up the existing tombstones.
	CleanTombstones(ctx context.Context) error
	// Config returns the current Prometheus configuration.
	Config(ctx context.Context) (ConfigResult, error)
	// DeleteSeries deletes data for a selection of series in a time range.
	DeleteSeries(ctx context.Context, matches []string, startTime, endTime time.Time) error
	// Flags returns the flag values that Prometheus was launched with.
	Flags(ctx context.Context) (FlagsResult, error)
	// LabelNames returns the unique label names present in the block in sorted order by given time range and matchers.
	LabelNames(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelNames, Warnings, error)
	// LabelValues performs a query for the values of the given label, time range and matchers.
	LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error)
	// Query performs a query for the given time.
	Query(ctx context.Context, query string, ts time.Time, opts ...Option) (model.Value, Warnings, error)
	// QueryRange performs a query for the given range.
	QueryRange(ctx context.Context, query string, r Range, opts ...Option) (model.Value, Warnings, error)
	// QueryExemplars performs a query for exemplars by the given query and time range.
	QueryExemplars(ctx context.Context, query string, startTime, endTime time.Time) ([]ExemplarQueryResult, error)
	// Buildinfo returns various build information properties about the Prometheus server
	Buildinfo(ctx context.Context) (BuildinfoResult, error)
	// Runtimeinfo returns the various runtime information properties about the Prometheus server.
	Runtimeinfo(ctx context.Context) (RuntimeinfoResult, error)
	// Series finds series by label matchers.
	Series(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) ([]model.LabelSet, Warnings, error)
	// Snapshot creates a snapshot of all current data into snapshots/<datetime>-<rand>
	// under the TSDB's data directory and returns the directory as response.
	Snapshot(ctx context.Context, skipHead bool) (SnapshotResult, error)
	// Rules returns a list of alerting and recording rules that are currently loaded.
	Rules(ctx context.Context, matches []string) (RulesResult, error)
	// Targets returns an overview of the current state of the Prometheus target discovery.
	Targets(ctx context.Context) (TargetsResult, error)
	// TargetsMetadata returns metadata about metrics currently scraped by the target.
	TargetsMetadata(ctx context.Context, matchTarget, metric, limit string) ([]MetricMetadata, error)
	// Metadata returns metadata about metrics currently scraped by the metric name.
	Metadata(ctx context.Context, metric, limit string) (map[string][]Metadata, error)
	// TSDB returns the cardinality statistics.
	TSDB(ctx context.Context, opts ...Option) (TSDBResult, error)
	// TSDBBlocks returns the list of currently loaded TSDB blocks and their metadata.
	TSDBBlocks(ctx context.Context) (TSDBBlocksResult, error)
	// WalReplay returns the current replay status of the wal.
	WalReplay(ctx context.Context) (WalReplayStatus, error)
	// FormatQuery formats a PromQL expression in a prettified way.
	FormatQuery(ctx context.Context, query string) (string, error)
}

// NewAPI returns a new API for the client.
//
// It is safe to use the returned API from multiple goroutines.
func NewAPI(c api.Client) API {
	return &httpAPI{client: &apiClientImpl{client: c}}
}

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

type queryResult struct {
	Type   model.ValueType `json:"resultType"`
	Result interface{}     `json:"result"`
	v      model.Value     `json:"-"`
}

func (qr *queryResult) UnmarshalJSON(b []byte) error {
	var decoded itypes.QueryResult
	if err := json.Unmarshal(b, &decoded); err != nil {
		return err
	}
	qr.Type = decoded.Type
	qr.Result = decoded.Result
	qr.v = decoded.Value()
	return nil
}

func (h *apiClientImpl) URL(ep string, args map[string]string) *url.URL {
	return h.client.URL(ep, args)
}

func (h *apiClientImpl) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, Warnings, error) {
	resp, body, warnings, err := transport.Do(ctx, h.client, req)
	return resp, body, Warnings(warnings), mapTransportError(err)
}

func (h *apiClientImpl) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error) {
	resp, body, warnings, err := transport.DoGetFallback(ctx, h.client, u, args)
	return resp, body, Warnings(warnings), mapTransportError(err)
}

func mapTransportError(err error) error {
	if err == nil {
		return nil
	}
	apiErr, ok := err.(*itypes.Error)
	if !ok {
		return err
	}
	return &Error{
		Type:   ErrorType(apiErr.Type),
		Msg:    apiErr.Msg,
		Detail: apiErr.Detail,
	}
}

func (h *httpAPI) Alerts(ctx context.Context) (AlertsResult, error) {
	u := h.client.URL(endpoints.Alerts, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return AlertsResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return AlertsResult{}, err
	}
	var res AlertsResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) AlertManagers(ctx context.Context) (AlertManagersResult, error) {
	u := h.client.URL(endpoints.AlertManagers, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return AlertManagersResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return AlertManagersResult{}, err
	}
	var res AlertManagersResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) CleanTombstones(ctx context.Context) error {
	u := h.client.URL(endpoints.CleanTombstones, nil)
	req, err := endpoints.NewRequest(http.MethodPost, u)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Config(ctx context.Context) (ConfigResult, error) {
	u := h.client.URL(endpoints.Config, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return ConfigResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return ConfigResult{}, err
	}
	var res ConfigResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) DeleteSeries(ctx context.Context, matches []string, startTime, endTime time.Time) error {
	u := h.client.URL(endpoints.DeleteSeries, nil)
	q := u.Query()
	for _, m := range matches {
		q.Add("match[]", m)
	}
	if !startTime.IsZero() {
		q.Set("start", endpoints.FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", endpoints.FormatTime(endTime))
	}
	endpoints.SetQuery(u, q)
	req, err := endpoints.NewRequest(http.MethodPost, u)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Flags(ctx context.Context) (FlagsResult, error) {
	u := h.client.URL(endpoints.Flags, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return FlagsResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return FlagsResult{}, err
	}
	var res FlagsResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) Buildinfo(ctx context.Context) (BuildinfoResult, error) {
	u := h.client.URL(endpoints.Buildinfo, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return BuildinfoResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return BuildinfoResult{}, err
	}
	var res BuildinfoResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) Runtimeinfo(ctx context.Context) (RuntimeinfoResult, error) {
	u := h.client.URL(endpoints.Runtimeinfo, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return RuntimeinfoResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return RuntimeinfoResult{}, err
	}
	var res RuntimeinfoResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) LabelNames(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelNames, Warnings, error) {
	u := h.client.URL(endpoints.Labels, nil)
	q := addOptionalURLParams(u.Query(), opts)
	if !startTime.IsZero() {
		q.Set("start", endpoints.FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", endpoints.FormatTime(endTime))
	}
	for _, m := range matches {
		q.Add("match[]", m)
	}
	_, body, w, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, w, err
	}
	var labelNames model.LabelNames
	return labelNames, w, json.Unmarshal(body, &labelNames)
}

func (h *httpAPI) LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error) {
	u := h.client.URL(endpoints.LabelValues, map[string]string{"name": label})
	q := addOptionalURLParams(u.Query(), opts)
	if !startTime.IsZero() {
		q.Set("start", endpoints.FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", endpoints.FormatTime(endTime))
	}
	for _, m := range matches {
		q.Add("match[]", m)
	}
	endpoints.SetQuery(u, q)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return nil, nil, err
	}
	_, body, w, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, w, err
	}
	var labelValues model.LabelValues
	return labelValues, w, json.Unmarshal(body, &labelValues)
}

func (h *httpAPI) Query(ctx context.Context, query string, ts time.Time, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(endpoints.Query, nil)
	q := addOptionalURLParams(u.Query(), opts)
	q.Set("query", query)
	if !ts.IsZero() {
		q.Set("time", endpoints.FormatTime(ts))
	}
	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}
	var qres itypes.QueryResult
	err = json.Unmarshal(body, &qres)
	return qres.Value(), warnings, err
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
	var qres itypes.QueryResult
	err = json.Unmarshal(body, &qres)
	return qres.Value(), warnings, err
}

func (h *httpAPI) Series(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) ([]model.LabelSet, Warnings, error) {
	u := h.client.URL(endpoints.Series, nil)
	q := addOptionalURLParams(u.Query(), opts)
	for _, m := range matches {
		q.Add("match[]", m)
	}
	if !startTime.IsZero() {
		q.Set("start", endpoints.FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", endpoints.FormatTime(endTime))
	}
	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}
	var mset []model.LabelSet
	return mset, warnings, json.Unmarshal(body, &mset)
}

func (h *httpAPI) Snapshot(ctx context.Context, skipHead bool) (SnapshotResult, error) {
	u := h.client.URL(endpoints.Snapshot, nil)
	q := u.Query()
	q.Set("skip_head", strconv.FormatBool(skipHead))
	endpoints.SetQuery(u, q)
	req, err := endpoints.NewRequest(http.MethodPost, u)
	if err != nil {
		return SnapshotResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return SnapshotResult{}, err
	}
	var res SnapshotResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) Rules(ctx context.Context, matches []string) (RulesResult, error) {
	u := h.client.URL(endpoints.Rules, nil)
	q := u.Query()
	for _, m := range matches {
		q.Add("match[]", m)
	}
	endpoints.SetQuery(u, q)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return RulesResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return RulesResult{}, err
	}
	var res RulesResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) Targets(ctx context.Context) (TargetsResult, error) {
	u := h.client.URL(endpoints.Targets, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return TargetsResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TargetsResult{}, err
	}
	var res TargetsResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) TargetsMetadata(ctx context.Context, matchTarget, metric, limit string) ([]MetricMetadata, error) {
	u := h.client.URL(endpoints.TargetsMetadata, nil)
	q := u.Query()
	q.Set("match_target", matchTarget)
	q.Set("metric", metric)
	q.Set("limit", limit)
	endpoints.SetQuery(u, q)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return nil, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	var res []MetricMetadata
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) Metadata(ctx context.Context, metric, limit string) (map[string][]Metadata, error) {
	u := h.client.URL(endpoints.Metadata, nil)
	q := u.Query()
	q.Set("metric", metric)
	q.Set("limit", limit)
	endpoints.SetQuery(u, q)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return nil, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	var res map[string][]Metadata
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) TSDB(ctx context.Context, opts ...Option) (TSDBResult, error) {
	u := h.client.URL(endpoints.TSDB, nil)
	endpoints.SetQuery(u, addOptionalURLParams(u.Query(), opts))
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return TSDBResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TSDBResult{}, err
	}
	var res TSDBResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) TSDBBlocks(ctx context.Context) (TSDBBlocksResult, error) {
	u := h.client.URL(endpoints.TSDBBlocks, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return TSDBBlocksResult{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TSDBBlocksResult{}, err
	}
	var res TSDBBlocksResult
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) WalReplay(ctx context.Context) (WalReplayStatus, error) {
	u := h.client.URL(endpoints.WalReplay, nil)
	req, err := endpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return WalReplayStatus{}, err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return WalReplayStatus{}, err
	}
	var res WalReplayStatus
	return res, json.Unmarshal(body, &res)
}

func (h *httpAPI) QueryExemplars(ctx context.Context, query string, startTime, endTime time.Time) ([]ExemplarQueryResult, error) {
	u := h.client.URL(endpoints.QueryExemplars, nil)
	q := u.Query()
	q.Set("query", query)
	if !startTime.IsZero() {
		q.Set("start", endpoints.FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", endpoints.FormatTime(endTime))
	}
	_, body, _, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, err
	}
	var res []ExemplarQueryResult
	return res, json.Unmarshal(body, &res)
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
