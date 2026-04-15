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

// Package v1 provides bindings to the Prometheus HTTP API v1:
// http://prometheus.io/docs/querying/api/
package v1

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/codec"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/endpoints"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/transport"
)

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

type apiOptions struct {
	timeout       time.Duration
	lookbackDelta time.Duration
	stats         StatsValue
	limit         uint64
}

type Option func(c *apiOptions)

// WithTimeout can be used to provide an optional query evaluation timeout for Query and QueryRange.
// https://prometheus.io/docs/prometheus/latest/querying/api/#instant-queries
func WithTimeout(timeout time.Duration) Option {
	return func(o *apiOptions) {
		o.timeout = timeout
	}
}

// WithLookbackDelta can be used to provide an optional query lookback delta for Query and QueryRange.
// This URL variable is not documented on Prometheus HTTP API.
// https://github.com/prometheus/prometheus/blob/e04913aea2792a5c8bc7b3130c389ca1b027dd9b/promql/engine.go#L162-L167
func WithLookbackDelta(lookbackDelta time.Duration) Option {
	return func(o *apiOptions) {
		o.lookbackDelta = lookbackDelta
	}
}

// WithStats can be used to provide an optional per step stats for Query and QueryRange.
// This URL variable is not documented on Prometheus HTTP API.
// https://github.com/prometheus/prometheus/blob/e04913aea2792a5c8bc7b3130c389ca1b027dd9b/promql/engine.go#L162-L167
func WithStats(stats StatsValue) Option {
	return func(o *apiOptions) {
		o.stats = stats
	}
}

// WithLimit provides an optional maximum number of returned entries for APIs that support limit parameter
// e.g. https://prometheus.io/docs/prometheus/latest/querying/api/#instant-querie:~:text=%3A%20End%20timestamp.-,limit%3D%3Cnumber%3E,-%3A%20Maximum%20number%20of
func WithLimit(limit uint64) Option {
	return func(o *apiOptions) {
		o.limit = limit
	}
}

// NewAPI returns a new API for the client.
//
// It is safe to use the returned API from multiple goroutines.
func NewAPI(c api.Client) API {
	return &httpAPI{
		client: transport.NewClient(c),
	}
}

type httpAPI struct {
	client transport.Client
}

type queryResult struct {
	Type   model.ValueType `json:"resultType"`
	Result interface{}     `json:"result"`
}

type apiResponse = transport.APIResponse

type apiClientImpl struct {
	client api.Client
}

func (h *httpAPI) Alerts(ctx context.Context) (AlertsResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.Alerts, nil, nil)
	if err != nil {
		return AlertsResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return AlertsResult{}, err
	}

	var res AlertsResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) AlertManagers(ctx context.Context) (AlertManagersResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.AlertManagers, nil, nil)
	if err != nil {
		return AlertManagersResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return AlertManagersResult{}, err
	}

	var res AlertManagersResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) CleanTombstones(ctx context.Context) error {
	req, err := endpoints.NewPostRequest(h.client, endpoints.CleanTombstones, nil, nil, nil)
	if err != nil {
		return err
	}

	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Config(ctx context.Context) (ConfigResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.Config, nil, nil)
	if err != nil {
		return ConfigResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return ConfigResult{}, err
	}

	var res ConfigResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) DeleteSeries(ctx context.Context, matches []string, startTime, endTime time.Time) error {
	q := url.Values{}
	for _, m := range matches {
		q.Add("match[]", m)
	}
	if !startTime.IsZero() {
		q.Set("start", endpoints.FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", endpoints.FormatTime(endTime))
	}

	req, err := endpoints.NewPostRequest(h.client, endpoints.DeleteSeries, nil, q, nil)
	if err != nil {
		return err
	}

	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Flags(ctx context.Context) (FlagsResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.Flags, nil, nil)
	if err != nil {
		return FlagsResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return FlagsResult{}, err
	}

	var res FlagsResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Buildinfo(ctx context.Context) (BuildinfoResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.Buildinfo, nil, nil)
	if err != nil {
		return BuildinfoResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return BuildinfoResult{}, err
	}

	var res BuildinfoResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Runtimeinfo(ctx context.Context) (RuntimeinfoResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.Runtimeinfo, nil, nil)
	if err != nil {
		return RuntimeinfoResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return RuntimeinfoResult{}, err
	}

	var res RuntimeinfoResult
	err = codec.Unmarshal(body, &res)
	return res, err
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

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var labelNames model.LabelNames
	err = codec.Unmarshal(body, &labelNames)
	return labelNames, warnings, err
}

func (h *httpAPI) LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error) {
	q := addOptionalURLParams(url.Values{}, opts)
	if !startTime.IsZero() {
		q.Set("start", endpoints.FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", endpoints.FormatTime(endTime))
	}
	for _, m := range matches {
		q.Add("match[]", m)
	}

	req, err := endpoints.NewGetRequest(h.client, endpoints.LabelValues, map[string]string{"name": label}, q)
	if err != nil {
		return nil, nil, err
	}

	_, body, warnings, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, warnings, err
	}

	var labelValues model.LabelValues
	err = codec.Unmarshal(body, &labelValues)
	return labelValues, warnings, err
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

	value, err := codec.DecodeQueryResult(body)
	return value, warnings, err
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

	value, err := codec.DecodeQueryResult(body)
	return value, warnings, err
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

	var res []model.LabelSet
	return res, warnings, codec.Unmarshal(body, &res)
}

func (h *httpAPI) Snapshot(ctx context.Context, skipHead bool) (SnapshotResult, error) {
	q := url.Values{}
	q.Set("skip_head", strconv.FormatBool(skipHead))

	req, err := endpoints.NewPostRequest(h.client, endpoints.Snapshot, nil, q, nil)
	if err != nil {
		return SnapshotResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return SnapshotResult{}, err
	}

	var res SnapshotResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Rules(ctx context.Context, matches []string) (RulesResult, error) {
	q := url.Values{}
	for _, m := range matches {
		q.Add("match[]", m)
	}

	req, err := endpoints.NewGetRequest(h.client, endpoints.Rules, nil, q)
	if err != nil {
		return RulesResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return RulesResult{}, err
	}

	var res RulesResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Targets(ctx context.Context) (TargetsResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.Targets, nil, nil)
	if err != nil {
		return TargetsResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TargetsResult{}, err
	}

	var res TargetsResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) TargetsMetadata(ctx context.Context, matchTarget, metric, limit string) ([]MetricMetadata, error) {
	q := url.Values{}
	q.Set("match_target", matchTarget)
	q.Set("metric", metric)
	q.Set("limit", limit)

	req, err := endpoints.NewGetRequest(h.client, endpoints.TargetsMetadata, nil, q)
	if err != nil {
		return nil, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	var res []MetricMetadata
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) Metadata(ctx context.Context, metric, limit string) (map[string][]Metadata, error) {
	q := url.Values{}
	q.Set("metric", metric)
	q.Set("limit", limit)

	req, err := endpoints.NewGetRequest(h.client, endpoints.Metadata, nil, q)
	if err != nil {
		return nil, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	var res map[string][]Metadata
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) TSDB(ctx context.Context, opts ...Option) (TSDBResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.TSDB, nil, addOptionalURLParams(url.Values{}, opts))
	if err != nil {
		return TSDBResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TSDBResult{}, err
	}

	var res TSDBResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) TSDBBlocks(ctx context.Context) (TSDBBlocksResult, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.TSDBBlocks, nil, nil)
	if err != nil {
		return TSDBBlocksResult{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return TSDBBlocksResult{}, err
	}

	var res TSDBBlocksResult
	err = codec.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) WalReplay(ctx context.Context) (WalReplayStatus, error) {
	req, err := endpoints.NewGetRequest(h.client, endpoints.WalReplay, nil, nil)
	if err != nil {
		return WalReplayStatus{}, err
	}

	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return WalReplayStatus{}, err
	}

	var res WalReplayStatus
	err = codec.Unmarshal(body, &res)
	return res, err
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
	err = codec.Unmarshal(body, &res)
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

func addOptionalURLParams(q url.Values, opts []Option) url.Values {
	opt := &apiOptions{}
	for _, o := range opts {
		o(opt)
	}

	if opt.timeout > 0 {
		q.Set("timeout", opt.timeout.String())
	}
	if opt.lookbackDelta > 0 {
		q.Set("lookback_delta", opt.lookbackDelta.String())
	}
	if opt.stats != "" {
		q.Set("stats", string(opt.stats))
	}
	if opt.limit > 0 {
		q.Set("limit", strconv.FormatUint(opt.limit, 10))
	}

	return q
}

func (h *apiClientImpl) URL(ep string, args map[string]string) *url.URL {
	return h.client.URL(ep, args)
}

func (h *apiClientImpl) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, Warnings, error) {
	return transport.NewClient(h.client).Do(ctx, req)
}

func (h *apiClientImpl) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error) {
	return transport.NewClient(h.client).DoGetFallback(ctx, u, args)
}

func (qr *queryResult) UnmarshalJSON(b []byte) error {
	value, err := codec.DecodeQueryResult(b)
	if err != nil {
		return err
	}
	qr.Result = value
	switch v := value.(type) {
	case *model.Scalar:
		qr.Type = model.ValScalar
		qr.Result = v
	case model.Vector:
		qr.Type = model.ValVector
	case model.Matrix:
		qr.Type = model.ValMatrix
	}
	return nil
}

func (qr queryResult) Value() model.Value {
	switch v := qr.Result.(type) {
	case *model.Scalar:
		return v
	case model.Vector:
		return v
	case model.Matrix:
		return v
	default:
		return nil
	}
}
