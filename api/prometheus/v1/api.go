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
//
// It is structured as a thin compatibility facade over four internal
// subpackages; see doc.go for the design rationale.
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
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

// ---------------------------------------------------------------------------
// Public type aliases — the exported surface of this package is defined by
// these aliases.  Every name resolves to its canonical definition in one of
// the internal subpackages so that callers using api/prometheus/v1 directly
// continue to work without any changes.
// ---------------------------------------------------------------------------

// AlertState models the state of an alert.
type AlertState = types.AlertState

// ErrorType models the different API error types.
type ErrorType = types.ErrorType

// HealthStatus models the health status of a scrape target.
type HealthStatus = types.HealthStatus

// RuleType models the type of a rule.
type RuleType = types.RuleType

// RuleHealth models the health status of a rule.
type RuleHealth = types.RuleHealth

// MetricType models the type of a metric.
type MetricType = types.MetricType

const (
	// Possible values for AlertState.
	AlertStateFiring   = types.AlertStateFiring
	AlertStateInactive = types.AlertStateInactive
	AlertStatePending  = types.AlertStatePending

	// Possible values for ErrorType.
	ErrBadData     = types.ErrBadData
	ErrTimeout     = types.ErrTimeout
	ErrCanceled    = types.ErrCanceled
	ErrExec        = types.ErrExec
	ErrBadResponse = types.ErrBadResponse
	ErrServer      = types.ErrServer
	ErrClient      = types.ErrClient

	// Possible values for HealthStatus.
	HealthGood    = types.HealthGood
	HealthUnknown = types.HealthUnknown
	HealthBad     = types.HealthBad

	// Possible values for RuleType.
	RuleTypeRecording = types.RuleTypeRecording
	RuleTypeAlerting  = types.RuleTypeAlerting

	// Possible values for RuleHealth.
	RuleHealthGood    = types.RuleHealthGood
	RuleHealthUnknown = types.RuleHealthUnknown
	RuleHealthBad     = types.RuleHealthBad

	// Possible values for MetricType.
	MetricTypeCounter        = types.MetricTypeCounter
	MetricTypeGauge          = types.MetricTypeGauge
	MetricTypeHistogram      = types.MetricTypeHistogram
	MetricTypeGaugeHistogram = types.MetricTypeGaugeHistogram
	MetricTypeSummary        = types.MetricTypeSummary
	MetricTypeInfo           = types.MetricTypeInfo
	MetricTypeStateset       = types.MetricTypeStateset
	MetricTypeUnknown        = types.MetricTypeUnknown
)

// Error is an error returned by the API.
type Error = types.Error

// Range represents a sliced time range.
type Range = types.Range

// Warnings is an array of non critical errors.
type Warnings = types.Warnings

// StatsValue is a type for the `stats` query parameter.
type StatsValue = types.StatsValue

// AllStatsValue is the query parameter value to return all query statistics.
const AllStatsValue = types.AllStatsValue

// Option is a function that configures per-request API options.
type Option = types.Option

// AlertsResult contains the result from querying the alerts endpoint.
type AlertsResult = types.AlertsResult

// AlertManagersResult contains the result from querying the alertmanagers endpoint.
type AlertManagersResult = types.AlertManagersResult

// AlertManager models a configured Alert Manager.
type AlertManager = types.AlertManager

// ConfigResult contains the result from querying the config endpoint.
type ConfigResult = types.ConfigResult

// FlagsResult contains the result from querying the flag endpoint.
type FlagsResult = types.FlagsResult

// BuildinfoResult contains the results from querying the buildinfo endpoint.
type BuildinfoResult = types.BuildinfoResult

// RuntimeinfoResult contains the result from querying the runtimeinfo endpoint.
type RuntimeinfoResult = types.RuntimeinfoResult

// SnapshotResult contains the result from querying the snapshot endpoint.
type SnapshotResult = types.SnapshotResult

// RulesResult contains the result from querying the rules endpoint.
type RulesResult = types.RulesResult

// RuleGroup models a rule group that contains a set of recording and alerting rules.
type RuleGroup = types.RuleGroup

// Recording and alerting rules are stored in the same slice to preserve the order
// that rules are returned in by the API.
//
// Rule types can be determined using a type switch:
//
//	switch v := rule.(type) {
//	case RecordingRule:
//		fmt.Print("got a recording rule")
//	case AlertingRule:
//		fmt.Print("got a alerting rule")
//	default:
//		fmt.Printf("unknown rule type %s", v)
//	}
type Rules = types.Rules

// AlertingRule models a alerting rule.
type AlertingRule = types.AlertingRule

// RecordingRule models a recording rule.
type RecordingRule = types.RecordingRule

// Alert models an active alert.
type Alert = types.Alert

// TargetsResult contains the result from querying the targets endpoint.
type TargetsResult = types.TargetsResult

// ActiveTarget models an active Prometheus scrape target.
type ActiveTarget = types.ActiveTarget

// DroppedTarget models a dropped Prometheus scrape target.
type DroppedTarget = types.DroppedTarget

// MetricMetadata models the metadata of a metric with its scrape target and name.
type MetricMetadata = types.MetricMetadata

// Metadata models the metadata of a metric.
type Metadata = types.Metadata

// TSDBResult contains the result from querying the tsdb endpoint.
type TSDBResult = types.TSDBResult

// TSDBHeadStats contains TSDB stats.
type TSDBHeadStats = types.TSDBHeadStats

// TSDBBlocksResult contains the results from querying the tsdb blocks endpoint.
type TSDBBlocksResult = types.TSDBBlocksResult

// TSDBBlocksData contains the metadata for the tsdb blocks.
type TSDBBlocksData = types.TSDBBlocksData

// TSDBBlocksBlockMetadata contains the metadata for a single tsdb block.
type TSDBBlocksBlockMetadata = types.TSDBBlocksBlockMetadata

// TSDBBlocksStats contains block stats for a single tsdb block.
type TSDBBlocksStats = types.TSDBBlocksStats

// TSDBBlocksCompaction contains block compaction details for a single block.
type TSDBBlocksCompaction = types.TSDBBlocksCompaction

// WalReplayStatus represents the wal replay status.
type WalReplayStatus = types.WalReplayStatus

// Stat models information about a statistic value.
type Stat = types.Stat

// Exemplar is additional information associated with a time series.
type Exemplar = types.Exemplar

// ExemplarQueryResult contains the result for a single time series from an
// exemplar query.
type ExemplarQueryResult = types.ExemplarQueryResult

// ---------------------------------------------------------------------------
// Option constructors — kept in the public package so go doc shows them here.
// ---------------------------------------------------------------------------

// WithTimeout can be used to provide an optional query evaluation timeout for Query and QueryRange.
// https://prometheus.io/docs/prometheus/latest/querying/api/#instant-queries
func WithTimeout(timeout time.Duration) Option {
	return func(o *types.APIOptions) {
		o.Timeout = timeout
	}
}

// WithLookbackDelta can be used to provide an optional query lookback delta for Query and QueryRange.
// This URL variable is not documented on Prometheus HTTP API.
// https://github.com/prometheus/prometheus/blob/e04913aea2792a5c8bc7b3130c389ca1b027dd9b/promql/engine.go#L162-L167
func WithLookbackDelta(lookbackDelta time.Duration) Option {
	return func(o *types.APIOptions) {
		o.LookbackDelta = lookbackDelta
	}
}

// WithStats can be used to provide an optional per step stats for Query and QueryRange.
// This URL variable is not documented on Prometheus HTTP API.
// https://github.com/prometheus/prometheus/blob/e04913aea2792a5c8bc7b3130c389ca1b027dd9b/promql/engine.go#L162-L167
func WithStats(stats StatsValue) Option {
	return func(o *types.APIOptions) {
		o.Stats = stats
	}
}

// WithLimit provides an optional maximum number of returned entries for APIs that support limit parameter
// e.g. https://prometheus.io/docs/prometheus/latest/querying/api/#instant-querie:~:text=%3A%20End%20timestamp.-,limit%3D%3Cnumber%3E,-%3A%20Maximum%20number%20of
func WithLimit(limit uint64) Option {
	return func(o *types.APIOptions) {
		o.Limit = limit
	}
}

// ---------------------------------------------------------------------------
// API interface
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Internal implementation types
// ---------------------------------------------------------------------------

// apiClient wraps a regular client and processes successful API responses.
// Successful also includes responses that errored at the API level.
type apiClient interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, Warnings, error)
	DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error)
}

// apiClientImpl implements apiClient by delegating HTTP execution to the
// transport subpackage. It keeps its client field unexported so that the
// test suite (which constructs it with a struct literal) does not need to
// change.
type apiClientImpl struct {
	client api.Client
}

// apiResponse is a type alias for transport.APIResponse so that tests can
// refer to it by the original unexported name within the v1 package.
type apiResponse = transport.APIResponse

// queryResult is the internal envelope used to decode query responses.
// It keeps its v field unexported so the existing test suite (which
// constructs it via struct literal without touching v) compiles unchanged.
type queryResult struct {
	Type   model.ValueType `json:"resultType"`
	Result interface{}     `json:"result"`

	// v holds the decoded model.Value after UnmarshalJSON runs.
	v model.Value
}

func (qr *queryResult) UnmarshalJSON(b []byte) error {
	// Delegate to codec.QueryResult to avoid duplicating the decode logic.
	var cr codec.QueryResult
	if err := json.Unmarshal(b, &cr); err != nil {
		return err
	}
	qr.Type = cr.Type
	qr.Result = cr.Result
	qr.v = cr.V
	return nil
}

func (h *apiClientImpl) URL(ep string, args map[string]string) *url.URL {
	return h.client.URL(ep, args)
}

func (h *apiClientImpl) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, Warnings, error) {
	return transport.Do(h.client, ctx, req)
}

func (h *apiClientImpl) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error) {
	return transport.DoGetFallback(h.client, ctx, u, args)
}

// httpAPI is the concrete implementation of API returned by NewAPI.
type httpAPI struct {
	client apiClient
}

// NewAPI returns a new API for the client.
//
// It is safe to use the returned API from multiple goroutines.
func NewAPI(c api.Client) API {
	return &httpAPI{
		client: &apiClientImpl{
			client: c,
		},
	}
}

// ---------------------------------------------------------------------------
// API method implementations
// ---------------------------------------------------------------------------

func (h *httpAPI) Alerts(ctx context.Context) (AlertsResult, error) {
	u := h.client.URL(endpoints.EPAlerts, nil)

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
	u := h.client.URL(endpoints.EPAlertManagers, nil)

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
	u := h.client.URL(endpoints.EPCleanTombstones, nil)

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}

	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Config(ctx context.Context) (ConfigResult, error) {
	u := h.client.URL(endpoints.EPConfig, nil)

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
	u := h.client.URL(endpoints.EPDeleteSeries, nil)
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

	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}

	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Flags(ctx context.Context) (FlagsResult, error) {
	u := h.client.URL(endpoints.EPFlags, nil)

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
	u := h.client.URL(endpoints.EPBuildinfo, nil)

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
	u := h.client.URL(endpoints.EPRuntimeinfo, nil)

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
	u := h.client.URL(endpoints.EPLabels, nil)
	q := endpoints.AddOptionalURLParams(u.Query(), opts)

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
	err = json.Unmarshal(body, &labelNames)
	return labelNames, w, err
}

func (h *httpAPI) LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error) {
	u := h.client.URL(endpoints.EPLabelValues, map[string]string{"name": label})
	q := endpoints.AddOptionalURLParams(u.Query(), opts)

	if !startTime.IsZero() {
		q.Set("start", endpoints.FormatTime(startTime))
	}
	if !endTime.IsZero() {
		q.Set("end", endpoints.FormatTime(endTime))
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
	u := h.client.URL(endpoints.EPQuery, nil)
	q := endpoints.AddOptionalURLParams(u.Query(), opts)

	q.Set("query", query)
	if !ts.IsZero() {
		q.Set("time", endpoints.FormatTime(ts))
	}

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var qres queryResult
	return qres.v, warnings, json.Unmarshal(body, &qres)
}

func (h *httpAPI) QueryRange(ctx context.Context, query string, r Range, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(endpoints.EPQueryRange, nil)
	q := endpoints.AddOptionalURLParams(u.Query(), opts)

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
	u := h.client.URL(endpoints.EPSeries, nil)
	q := endpoints.AddOptionalURLParams(u.Query(), opts)

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
	u := h.client.URL(endpoints.EPSnapshot, nil)
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
	u := h.client.URL(endpoints.EPRules, nil)
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
	u := h.client.URL(endpoints.EPTargets, nil)

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
	u := h.client.URL(endpoints.EPTargetsMetadata, nil)
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
	u := h.client.URL(endpoints.EPMetadata, nil)
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
	u := h.client.URL(endpoints.EPTSDB, nil)
	q := endpoints.AddOptionalURLParams(u.Query(), opts)
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
	u := h.client.URL(endpoints.EPTSDBBlocks, nil)

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
	u := h.client.URL(endpoints.EPWalReplay, nil)

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
	u := h.client.URL(endpoints.EPQueryExemplars, nil)
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
	err = json.Unmarshal(body, &res)
	return res, err
}

func (h *httpAPI) FormatQuery(ctx context.Context, query string) (string, error) {
	u := h.client.URL(endpoints.EPFormatQuery, nil)
	q := u.Query()
	q.Set("query", query)

	_, body, _, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
