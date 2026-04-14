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

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
)

const (
	apiPrefix = "/api/v1"

	epAlerts          = apiPrefix + "/alerts"
	epAlertManagers   = apiPrefix + "/alertmanagers"
	epQuery           = apiPrefix + "/query"
	epQueryRange      = apiPrefix + "/query_range"
	epQueryExemplars  = apiPrefix + "/query_exemplars"
	epLabels          = apiPrefix + "/labels"
	epLabelValues     = apiPrefix + "/label/:name/values"
	epSeries          = apiPrefix + "/series"
	epTargets         = apiPrefix + "/targets"
	epTargetsMetadata = apiPrefix + "/targets/metadata"
	epMetadata        = apiPrefix + "/metadata"
	epRules           = apiPrefix + "/rules"
	epSnapshot        = apiPrefix + "/admin/tsdb/snapshot"
	epDeleteSeries    = apiPrefix + "/admin/tsdb/delete_series"
	epCleanTombstones = apiPrefix + "/admin/tsdb/clean_tombstones"
	epConfig          = apiPrefix + "/status/config"
	epFlags           = apiPrefix + "/status/flags"
	epBuildinfo       = apiPrefix + "/status/buildinfo"
	epRuntimeinfo     = apiPrefix + "/status/runtimeinfo"
	epTSDB            = apiPrefix + "/status/tsdb"
	epTSDBBlocks      = apiPrefix + "/status/tsdb/blocks"
	epWalReplay       = apiPrefix + "/status/walreplay"
	epFormatQuery     = apiPrefix + "/format_query"
)

// Range represents a sliced time range.
type Range struct {
	// The boundaries of the time range.
	Start, End time.Time
	// The maximum time between two slices within the boundaries.
	Step time.Duration
}

// API provides bindings for Prometheus's v1 API.
type API interface {
	Alerts(ctx context.Context) (AlertsResult, error)
	AlertManagers(ctx context.Context) (AlertManagersResult, error)
	CleanTombstones(ctx context.Context) error
	Config(ctx context.Context) (ConfigResult, error)
	DeleteSeries(ctx context.Context, matches []string, startTime, endTime time.Time) error
	Flags(ctx context.Context) (FlagsResult, error)
	LabelNames(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelNames, Warnings, error)
	LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error)
	Query(ctx context.Context, query string, ts time.Time, opts ...Option) (model.Value, Warnings, error)
	QueryRange(ctx context.Context, query string, r Range, opts ...Option) (model.Value, Warnings, error)
	QueryExemplars(ctx context.Context, query string, startTime, endTime time.Time) ([]ExemplarQueryResult, error)
	Buildinfo(ctx context.Context) (BuildinfoResult, error)
	Runtimeinfo(ctx context.Context) (RuntimeinfoResult, error)
	Series(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) ([]model.LabelSet, Warnings, error)
	Snapshot(ctx context.Context, skipHead bool) (SnapshotResult, error)
	Rules(ctx context.Context, matches []string) (RulesResult, error)
	Targets(ctx context.Context) (TargetsResult, error)
	TargetsMetadata(ctx context.Context, matchTarget, metric, limit string) ([]MetricMetadata, error)
	Metadata(ctx context.Context, metric, limit string) (map[string][]Metadata, error)
	TSDB(ctx context.Context, opts ...Option) (TSDBResult, error)
	TSDBBlocks(ctx context.Context) (TSDBBlocksResult, error)
	WalReplay(ctx context.Context) (WalReplayStatus, error)
	FormatQuery(ctx context.Context, query string) (string, error)
}

// StatsValue is a type for `stats` query parameter.
type StatsValue string

const (
	// AllStatsValue is the query parameter value to return all the query statistics.
	AllStatsValue StatsValue = "all"
)

type apiOptions struct {
	timeout       time.Duration
	lookbackDelta time.Duration
	stats         StatsValue
	limit         uint64
}

type Option func(c *apiOptions)

// WithTimeout can be used to provide an optional query evaluation timeout for Query and QueryRange.
func WithTimeout(timeout time.Duration) Option {
	return func(o *apiOptions) {
		o.timeout = timeout
	}
}

// WithLookbackDelta can be used to provide an optional query lookback delta for Query and QueryRange.
func WithLookbackDelta(lookbackDelta time.Duration) Option {
	return func(o *apiOptions) {
		o.lookbackDelta = lookbackDelta
	}
}

// WithStats can be used to provide an optional per step stats for Query and QueryRange.
func WithStats(stats StatsValue) Option {
	return func(o *apiOptions) {
		o.stats = stats
	}
}

// WithLimit provides an optional maximum number of returned entries for APIs that support the limit parameter.
func WithLimit(limit uint64) Option {
	return func(o *apiOptions) {
		o.limit = limit
	}
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

// NewAPI returns a new API for the client.
//
// It is safe to use the returned API from multiple goroutines.
func NewAPI(c api.Client) API {
	return &httpAPI{
		client: &apiClientImpl{client: c},
	}
}

type httpAPI struct {
	client apiClient
}

func (h *httpAPI) Alerts(ctx context.Context) (AlertsResult, error) {
	u := h.client.URL(epAlerts, nil)
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
	u := h.client.URL(epAlertManagers, nil)
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
	u := h.client.URL(epCleanTombstones, nil)
	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Config(ctx context.Context) (ConfigResult, error) {
	u := h.client.URL(epConfig, nil)
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
	u := h.client.URL(epDeleteSeries, nil)
	q := u.Query()
	for _, m := range matches {
		q.Add("match[]", m)
	}
	if !startTime.IsZero() {
		q.Set("start", formatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", formatTime(endTime))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Flags(ctx context.Context) (FlagsResult, error) {
	u := h.client.URL(epFlags, nil)
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
	u := h.client.URL(epBuildinfo, nil)
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
	u := h.client.URL(epRuntimeinfo, nil)
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
	u := h.client.URL(epLabels, nil)
	q := addOptionalURLParams(u.Query(), opts)
	if !startTime.IsZero() {
		q.Set("start", formatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", formatTime(endTime))
	}
	for _, m := range matches {
		q.Add("match[]", m)
	}

	_, body, w, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, w, err
	}

	var labelNames model.LabelNames
	err = json.Unmarshal(body, &labelNames)
	return labelNames, w, err
}

func (h *httpAPI) LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error) {
	u := h.client.URL(epLabelValues, map[string]string{"name": label})
	q := addOptionalURLParams(u.Query(), opts)
	if !startTime.IsZero() {
		q.Set("start", formatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", formatTime(endTime))
	}
	for _, m := range matches {
		q.Add("match[]", m)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	_, body, w, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, w, err
	}

	var labelValues model.LabelValues
	err = json.Unmarshal(body, &labelValues)
	return labelValues, w, err
}

func (h *httpAPI) Query(ctx context.Context, query string, ts time.Time, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(epQuery, nil)
	q := addOptionalURLParams(u.Query(), opts)
	q.Set("query", query)
	if !ts.IsZero() {
		q.Set("time", formatTime(ts))
	}

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var qres queryResult
	return qres.v, warnings, json.Unmarshal(body, &qres)
}

func (h *httpAPI) QueryRange(ctx context.Context, query string, r Range, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(epQueryRange, nil)
	q := addOptionalURLParams(u.Query(), opts)
	q.Set("query", query)
	q.Set("start", formatTime(r.Start))
	q.Set("end", formatTime(r.End))
	q.Set("step", strconv.FormatFloat(r.Step.Seconds(), 'f', -1, 64))

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var qres queryResult
	return qres.v, warnings, json.Unmarshal(body, &qres)
}

func (h *httpAPI) Series(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) ([]model.LabelSet, Warnings, error) {
	u := h.client.URL(epSeries, nil)
	q := addOptionalURLParams(u.Query(), opts)
	for _, m := range matches {
		q.Add("match[]", m)
	}
	if !startTime.IsZero() {
		q.Set("start", formatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", formatTime(endTime))
	}

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var mset []model.LabelSet
	return mset, warnings, json.Unmarshal(body, &mset)
}

func (h *httpAPI) Snapshot(ctx context.Context, skipHead bool) (SnapshotResult, error) {
	u := h.client.URL(epSnapshot, nil)
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
	u := h.client.URL(epRules, nil)
	q := u.Query()
	for _, m := range matches {
		q.Add("match[]", m)
	}
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
	u := h.client.URL(epTargets, nil)
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
	u := h.client.URL(epTargetsMetadata, nil)
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
	u := h.client.URL(epMetadata, nil)
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
	u := h.client.URL(epTSDB, nil)
	q := addOptionalURLParams(u.Query(), opts)
	u.RawQuery = q.Encode()

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
	u := h.client.URL(epTSDBBlocks, nil)
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
	u := h.client.URL(epWalReplay, nil)
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
	u := h.client.URL(epQueryExemplars, nil)
	q := u.Query()
	q.Set("query", query)
	if !startTime.IsZero() {
		q.Set("start", formatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", formatTime(endTime))
	}

	_, body, _, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, err
	}

	var res []ExemplarQueryResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) FormatQuery(ctx context.Context, query string) (string, error) {
	u := h.client.URL(epFormatQuery, nil)
	q := u.Query()
	q.Set("query", query)

	_, body, _, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func formatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
