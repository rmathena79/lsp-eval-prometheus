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
	internalcodec "github.com/prometheus/client_golang/api/prometheus/v1/internal/codec"
	internalendpoints "github.com/prometheus/client_golang/api/prometheus/v1/internal/endpoints"
	internaltransport "github.com/prometheus/client_golang/api/prometheus/v1/internal/transport"
	internaltypes "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

func init() {
	internalcodec.Register()
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
	return &httpAPI{
		client: &apiClientImpl{client: c},
	}
}

type httpAPI struct {
	client apiClient
}

func (h *httpAPI) Alerts(ctx context.Context) (AlertsResult, error) {
	return decodeGet[AlertsResult](ctx, h.client, internalendpoints.AlertsPath)
}

func (h *httpAPI) AlertManagers(ctx context.Context) (AlertManagersResult, error) {
	return decodeGet[AlertManagersResult](ctx, h.client, internalendpoints.AlertManagersPath)
}

func (h *httpAPI) CleanTombstones(ctx context.Context) error {
	req, err := internalendpoints.NewRequest(http.MethodPost, h.client.URL(internalendpoints.CleanTombstonesPath, nil))
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Config(ctx context.Context) (ConfigResult, error) {
	return decodeGet[ConfigResult](ctx, h.client, internalendpoints.ConfigPath)
}

func (h *httpAPI) DeleteSeries(ctx context.Context, matches []string, startTime, endTime time.Time) error {
	u := h.client.URL(internalendpoints.DeleteSeriesPath, nil)
	q := u.Query()
	internalendpoints.AddMatchers(q, matches)
	internalendpoints.AddTimeRange(q, startTime, endTime)
	u.RawQuery = q.Encode()

	req, err := internalendpoints.NewRequest(http.MethodPost, u)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Flags(ctx context.Context) (FlagsResult, error) {
	return decodeGet[FlagsResult](ctx, h.client, internalendpoints.FlagsPath)
}

func (h *httpAPI) Buildinfo(ctx context.Context) (BuildinfoResult, error) {
	return decodeGet[BuildinfoResult](ctx, h.client, internalendpoints.BuildinfoPath)
}

func (h *httpAPI) Runtimeinfo(ctx context.Context) (RuntimeinfoResult, error) {
	return decodeGet[RuntimeinfoResult](ctx, h.client, internalendpoints.RuntimeinfoPath)
}

func (h *httpAPI) LabelNames(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelNames, Warnings, error) {
	u := h.client.URL(internalendpoints.LabelsPath, nil)
	q := addOptionalURLParams(u.Query(), opts)
	internalendpoints.AddTimeRange(q, startTime, endTime)
	internalendpoints.AddMatchers(q, matches)

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var labelNames model.LabelNames
	err = json.Unmarshal(body, &labelNames)
	return labelNames, warnings, err
}

func (h *httpAPI) LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error) {
	u := h.client.URL(internalendpoints.LabelValuesPath, map[string]string{"name": label})
	q := addOptionalURLParams(u.Query(), opts)
	internalendpoints.AddTimeRange(q, startTime, endTime)
	internalendpoints.AddMatchers(q, matches)
	u.RawQuery = q.Encode()

	req, err := internalendpoints.NewRequest(http.MethodGet, u)
	if err != nil {
		return nil, nil, err
	}

	_, body, warnings, err := h.client.Do(ctx, req)
	if err != nil {
		return nil, warnings, err
	}

	var labelValues model.LabelValues
	err = json.Unmarshal(body, &labelValues)
	return labelValues, warnings, err
}

// StatsValue is a type for `stats` query parameter.
type StatsValue string

// AllStatsValue is the query parameter value to return all the query statistics.
const (
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
// https://prometheus.io/docs/prometheus/latest/querying/api/#instant-queries
func WithTimeout(timeout time.Duration) Option {
	return func(o *apiOptions) { o.timeout = timeout }
}

// WithLookbackDelta can be used to provide an optional query lookback delta for Query and QueryRange.
// This URL variable is not documented on Prometheus HTTP API.
// https://github.com/prometheus/prometheus/blob/e04913aea2792a5c8bc7b3130c389ca1b027dd9b/promql/engine.go#L162-L167
func WithLookbackDelta(lookbackDelta time.Duration) Option {
	return func(o *apiOptions) { o.lookbackDelta = lookbackDelta }
}

// WithStats can be used to provide an optional per step stats for Query and QueryRange.
// This URL variable is not documented on Prometheus HTTP API.
// https://github.com/prometheus/prometheus/blob/e04913aea2792a5c8bc7b3130c389ca1b027dd9b/promql/engine.go#L162-L167
func WithStats(stats StatsValue) Option {
	return func(o *apiOptions) { o.stats = stats }
}

// WithLimit provides an optional maximum number of returned entries for APIs that support limit parameter
// e.g. https://prometheus.io/docs/prometheus/latest/querying/api/#instant-querie:~:text=%3A%20End%20timestamp.-,limit%3D%3Cnumber%3E,-%3A%20Maximum%20number%20of
func WithLimit(limit uint64) Option {
	return func(o *apiOptions) { o.limit = limit }
}

func addOptionalURLParams(q url.Values, opts []Option) url.Values {
	opt := &apiOptions{}
	for _, o := range opts {
		o(opt)
	}

	return internalendpoints.AddOptionalURLParams(q, internalendpoints.Options{
		Timeout:       opt.timeout,
		LookbackDelta: opt.lookbackDelta,
		Stats:         string(opt.stats),
		Limit:         opt.limit,
	})
}

func (h *httpAPI) Query(ctx context.Context, query string, ts time.Time, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(internalendpoints.QueryPath, nil)
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
	if err := json.Unmarshal(body, &qres); err != nil {
		return nil, warnings, err
	}
	return qres.Value(), warnings, nil
}

func (h *httpAPI) QueryRange(ctx context.Context, query string, r Range, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(internalendpoints.QueryRangePath, nil)
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
	if err := json.Unmarshal(body, &qres); err != nil {
		return nil, warnings, err
	}
	return qres.Value(), warnings, nil
}

func (h *httpAPI) Series(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) ([]model.LabelSet, Warnings, error) {
	u := h.client.URL(internalendpoints.SeriesPath, nil)
	q := addOptionalURLParams(u.Query(), opts)
	internalendpoints.AddMatchers(q, matches)
	internalendpoints.AddTimeRange(q, startTime, endTime)

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var mset []model.LabelSet
	return mset, warnings, json.Unmarshal(body, &mset)
}

func (h *httpAPI) Snapshot(ctx context.Context, skipHead bool) (SnapshotResult, error) {
	u := h.client.URL(internalendpoints.SnapshotPath, nil)
	q := u.Query()
	q.Set("skip_head", strconv.FormatBool(skipHead))
	u.RawQuery = q.Encode()

	req, err := internalendpoints.NewRequest(http.MethodPost, u)
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
	u := h.client.URL(internalendpoints.RulesPath, nil)
	q := u.Query()
	internalendpoints.AddMatchers(q, matches)
	u.RawQuery = q.Encode()

	req, err := internalendpoints.NewRequest(http.MethodGet, u)
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
	return decodeGet[TargetsResult](ctx, h.client, internalendpoints.TargetsPath)
}

func (h *httpAPI) TargetsMetadata(ctx context.Context, matchTarget, metric, limit string) ([]MetricMetadata, error) {
	u := h.client.URL(internalendpoints.TargetsMetadataPath, nil)
	q := u.Query()
	q.Set("match_target", matchTarget)
	q.Set("metric", metric)
	q.Set("limit", limit)
	u.RawQuery = q.Encode()

	req, err := internalendpoints.NewRequest(http.MethodGet, u)
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
	u := h.client.URL(internalendpoints.MetadataPath, nil)
	q := u.Query()
	q.Set("metric", metric)
	q.Set("limit", limit)
	u.RawQuery = q.Encode()

	req, err := internalendpoints.NewRequest(http.MethodGet, u)
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
	u := h.client.URL(internalendpoints.TSDBPath, nil)
	q := addOptionalURLParams(u.Query(), opts)
	u.RawQuery = q.Encode()

	req, err := internalendpoints.NewRequest(http.MethodGet, u)
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
	return decodeGet[TSDBBlocksResult](ctx, h.client, internalendpoints.TSDBBlocksPath)
}

func (h *httpAPI) WalReplay(ctx context.Context) (WalReplayStatus, error) {
	return decodeGet[WalReplayStatus](ctx, h.client, internalendpoints.WalReplayPath)
}

func (h *httpAPI) QueryExemplars(ctx context.Context, query string, startTime, endTime time.Time) ([]ExemplarQueryResult, error) {
	u := h.client.URL(internalendpoints.QueryExemplarsPath, nil)
	q := u.Query()
	q.Set("query", query)
	internalendpoints.AddTimeRange(q, startTime, endTime)

	_, body, _, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, err
	}

	var res []ExemplarQueryResult
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) FormatQuery(ctx context.Context, query string) (string, error) {
	u := h.client.URL(internalendpoints.FormatQueryPath, nil)
	q := u.Query()
	q.Set("query", query)

	_, body, _, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func decodeGet[T any](ctx context.Context, client apiClient, path string) (T, error) {
	var zero T
	req, err := internalendpoints.NewRequest(http.MethodGet, client.URL(path, nil))
	if err != nil {
		return zero, err
	}
	_, body, _, err := client.Do(ctx, req)
	if err != nil {
		return zero, err
	}
	var res T
	err = json.Unmarshal(body, &res)
	return res, err
}

// Warnings is an array of non critical errors
type Warnings = internaltypes.Warnings

type queryResult = internaltypes.QueryResult
type apiResponse = internaltypes.APIResponse

type apiClient interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, Warnings, error)
	DoGetFallback(context.Context, *url.URL, url.Values) (*http.Response, []byte, Warnings, error)
}

type apiClientImpl struct {
	client api.Client
	inner  internaltransport.Client
}

func (h *apiClientImpl) transport() internaltransport.Client {
	if h.inner == nil {
		h.inner = internaltransport.NewClient(h.client)
	}
	return h.inner
}

func (h *apiClientImpl) URL(ep string, args map[string]string) *url.URL {
	return h.transport().URL(ep, args)
}

func (h *apiClientImpl) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, Warnings, error) {
	return h.transport().Do(ctx, req)
}

func (h *apiClientImpl) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error) {
	return h.transport().DoGetFallback(ctx, u, args)
}

func formatTime(t time.Time) string {
	return internalendpoints.FormatTime(t)
}
