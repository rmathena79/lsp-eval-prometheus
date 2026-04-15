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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/codec"
	"github.com/prometheus/client_golang/api/prometheus/v1/internal/endpoints"
	internaltransport "github.com/prometheus/client_golang/api/prometheus/v1/internal/transport"
	internaltypes "github.com/prometheus/client_golang/api/prometheus/v1/internal/types"
)

func init() {
	codec.Register()
}

// AlertState models the state of an alert.
type AlertState string

// ErrorType models the different API error types.
type ErrorType string

// HealthStatus models the health status of a scrape target.
type HealthStatus string

// RuleType models the type of a rule.
type RuleType string

// RuleHealth models the health status of a rule.
type RuleHealth string

// MetricType models the type of a metric.
type MetricType string

const (
	// Possible values for AlertState.
	AlertStateFiring   AlertState = "firing"
	AlertStateInactive AlertState = "inactive"
	AlertStatePending  AlertState = "pending"

	// Possible values for ErrorType.
	ErrBadData     ErrorType = "bad_data"
	ErrTimeout     ErrorType = "timeout"
	ErrCanceled    ErrorType = "canceled"
	ErrExec        ErrorType = "execution"
	ErrBadResponse ErrorType = "bad_response"
	ErrServer      ErrorType = "server_error"
	ErrClient      ErrorType = "client_error"

	// Possible values for HealthStatus.
	HealthGood    HealthStatus = "up"
	HealthUnknown HealthStatus = "unknown"
	HealthBad     HealthStatus = "down"

	// Possible values for RuleType.
	RuleTypeRecording RuleType = "recording"
	RuleTypeAlerting  RuleType = "alerting"

	// Possible values for RuleHealth.
	RuleHealthGood    = "ok"
	RuleHealthUnknown = "unknown"
	RuleHealthBad     = "err"

	// Possible values for MetricType
	MetricTypeCounter        MetricType = "counter"
	MetricTypeGauge          MetricType = "gauge"
	MetricTypeHistogram      MetricType = "histogram"
	MetricTypeGaugeHistogram MetricType = "gaugehistogram"
	MetricTypeSummary        MetricType = "summary"
	MetricTypeInfo           MetricType = "info"
	MetricTypeStateset       MetricType = "stateset"
	MetricTypeUnknown        MetricType = "unknown"
)

// Error is an error returned by the API.
type Error struct {
	Type   ErrorType
	Msg    string
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Msg)
}

// Range represents a sliced time range.
type Range struct {
	// The boundaries of the time range.
	Start, End time.Time
	// The maximum time between two slices within the boundaries.
	Step time.Duration
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

// AlertsResult contains the result from querying the alerts endpoint.
type AlertsResult struct {
	Alerts []Alert `json:"alerts"`
}

// AlertManagersResult contains the result from querying the alertmanagers endpoint.
type AlertManagersResult struct {
	Active  []AlertManager `json:"activeAlertManagers"`
	Dropped []AlertManager `json:"droppedAlertManagers"`
}

// AlertManager models a configured Alert Manager.
type AlertManager struct {
	URL string `json:"url"`
}

// ConfigResult contains the result from querying the config endpoint.
type ConfigResult struct {
	YAML string `json:"yaml"`
}

// FlagsResult contains the result from querying the flag endpoint.
type FlagsResult map[string]string

// BuildinfoResult contains the results from querying the buildinfo endpoint.
type BuildinfoResult struct {
	Version   string `json:"version"`
	Revision  string `json:"revision"`
	Branch    string `json:"branch"`
	BuildUser string `json:"buildUser"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
}

// RuntimeinfoResult contains the result from querying the runtimeinfo endpoint.
type RuntimeinfoResult struct {
	StartTime           time.Time `json:"startTime"`
	CWD                 string    `json:"CWD"`
	ReloadConfigSuccess bool      `json:"reloadConfigSuccess"`
	LastConfigTime      time.Time `json:"lastConfigTime"`
	CorruptionCount     int       `json:"corruptionCount"`
	GoroutineCount      int       `json:"goroutineCount"`
	GOMAXPROCS          int       `json:"GOMAXPROCS"`
	GOGC                string    `json:"GOGC"`
	GODEBUG             string    `json:"GODEBUG"`
	StorageRetention    string    `json:"storageRetention"`
}

// SnapshotResult contains the result from querying the snapshot endpoint.
type SnapshotResult struct {
	Name string `json:"name"`
}

// RulesResult contains the result from querying the rules endpoint.
type RulesResult struct {
	Groups []RuleGroup `json:"groups"`
}

// RuleGroup models a rule group that contains a set of recording and alerting rules.
type RuleGroup struct {
	Name     string  `json:"name"`
	File     string  `json:"file"`
	Interval float64 `json:"interval"`
	Rules    Rules   `json:"rules"`
}

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
type Rules []interface{}

// AlertingRule models a alerting rule.
type AlertingRule struct {
	Name           string         `json:"name"`
	Query          string         `json:"query"`
	Duration       float64        `json:"duration"`
	Labels         model.LabelSet `json:"labels"`
	Annotations    model.LabelSet `json:"annotations"`
	Alerts         []*Alert       `json:"alerts"`
	Health         RuleHealth     `json:"health"`
	LastError      string         `json:"lastError,omitempty"`
	EvaluationTime float64        `json:"evaluationTime"`
	LastEvaluation time.Time      `json:"lastEvaluation"`
	State          string         `json:"state"`
}

// RecordingRule models a recording rule.
type RecordingRule struct {
	Name           string         `json:"name"`
	Query          string         `json:"query"`
	Labels         model.LabelSet `json:"labels,omitempty"`
	Health         RuleHealth     `json:"health"`
	LastError      string         `json:"lastError,omitempty"`
	EvaluationTime float64        `json:"evaluationTime"`
	LastEvaluation time.Time      `json:"lastEvaluation"`
}

// Alert models an active alert.
type Alert struct {
	ActiveAt    time.Time `json:"activeAt"`
	Annotations model.LabelSet
	Labels      model.LabelSet
	State       AlertState
	Value       string
}

// TargetsResult contains the result from querying the targets endpoint.
type TargetsResult struct {
	Active  []ActiveTarget  `json:"activeTargets"`
	Dropped []DroppedTarget `json:"droppedTargets"`
}

// ActiveTarget models an active Prometheus scrape target.
type ActiveTarget struct {
	DiscoveredLabels   map[string]string `json:"discoveredLabels"`
	Labels             model.LabelSet    `json:"labels"`
	ScrapePool         string            `json:"scrapePool"`
	ScrapeURL          string            `json:"scrapeUrl"`
	GlobalURL          string            `json:"globalUrl"`
	LastError          string            `json:"lastError"`
	LastScrape         time.Time         `json:"lastScrape"`
	LastScrapeDuration float64           `json:"lastScrapeDuration"`
	Health             HealthStatus      `json:"health"`
}

// DroppedTarget models a dropped Prometheus scrape target.
type DroppedTarget struct {
	DiscoveredLabels map[string]string `json:"discoveredLabels"`
}

// MetricMetadata models the metadata of a metric with its scrape target and name.
type MetricMetadata struct {
	Target map[string]string `json:"target"`
	Metric string            `json:"metric,omitempty"`
	Type   MetricType        `json:"type"`
	Help   string            `json:"help"`
	Unit   string            `json:"unit"`
}

// Metadata models the metadata of a metric.
type Metadata struct {
	Type MetricType `json:"type"`
	Help string     `json:"help"`
	Unit string     `json:"unit"`
}

// queryResult contains result data for a query.
type queryResult struct {
	Type   model.ValueType `json:"resultType"`
	Result interface{}     `json:"result"`
}

type apiResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType ErrorType       `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings,omitempty"`
}

// TSDBResult contains the result from querying the tsdb endpoint.
type TSDBResult struct {
	HeadStats                   TSDBHeadStats `json:"headStats"`
	SeriesCountByMetricName     []Stat        `json:"seriesCountByMetricName"`
	LabelValueCountByLabelName  []Stat        `json:"labelValueCountByLabelName"`
	MemoryInBytesByLabelName    []Stat        `json:"memoryInBytesByLabelName"`
	SeriesCountByLabelValuePair []Stat        `json:"seriesCountByLabelValuePair"`
}

// TSDBHeadStats contains TSDB stats
type TSDBHeadStats struct {
	NumSeries     int `json:"numSeries"`
	NumLabelPairs int `json:"numLabelPairs"`
	ChunkCount    int `json:"chunkCount"`
	MinTime       int `json:"minTime"`
	MaxTime       int `json:"maxTime"`
}

// TSDBBlocksResult contains the results from querying the tsdb blocks endpoint.
type TSDBBlocksResult struct {
	Status string         `json:"status"`
	Data   TSDBBlocksData `json:"data"`
}

// TSDBBlocksData contains the metadata for the tsdb blocks.
type TSDBBlocksData struct {
	Blocks []TSDBBlocksBlockMetadata `json:"blocks"`
}

// TSDBBlocksBlockMetadata contains the metadata for a single tsdb block.
type TSDBBlocksBlockMetadata struct {
	Ulid       string               `json:"ulid"`
	MinTime    int64                `json:"minTime"`
	MaxTime    int64                `json:"maxTime"`
	Stats      TSDBBlocksStats      `json:"stats"`
	Compaction TSDBBlocksCompaction `json:"compaction"`
	Version    int                  `json:"version"`
}

// TSDBBlocksStats contains block stats for a single tsdb block.
type TSDBBlocksStats struct {
	NumSamples int `json:"numSamples"`
	NumSeries  int `json:"numSeries"`
	NumChunks  int `json:"numChunks"`
}

// TSDBBlocksCompaction contains block compaction details for a single block.
type TSDBBlocksCompaction struct {
	Level   int      `json:"level"`
	Sources []string `json:"sources"`
}

// WalReplayStatus represents the wal replay status.
type WalReplayStatus struct {
	Min     int `json:"min"`
	Max     int `json:"max"`
	Current int `json:"current"`
}

// Stat models information about statistic value.
type Stat struct {
	Name  string `json:"name"`
	Value uint64 `json:"value"`
}

func (rg *RuleGroup) UnmarshalJSON(b []byte) error {
	payload, err := codec.DecodeRuleGroupPayload(b)
	if err != nil {
		return err
	}

	rg.Name = payload.Name
	rg.File = payload.File
	rg.Interval = payload.Interval
	rg.Rules = rg.Rules[:0]

	for _, rule := range payload.Rules {
		alertingRule := AlertingRule{}
		if err := json.Unmarshal(rule, &alertingRule); err == nil {
			rg.Rules = append(rg.Rules, alertingRule)
			continue
		}

		recordingRule := RecordingRule{}
		if err := json.Unmarshal(rule, &recordingRule); err == nil {
			rg.Rules = append(rg.Rules, recordingRule)
			continue
		}

		return errors.New("failed to decode JSON into an alerting or recording rule")
	}

	return nil
}

func (r *AlertingRule) UnmarshalJSON(b []byte) error {
	ruleType, err := codec.DecodeRuleType(b)
	if err != nil {
		return err
	}
	if ruleType != string(RuleTypeAlerting) {
		return fmt.Errorf("expected rule of type %s but got %s", string(RuleTypeAlerting), ruleType)
	}

	rule, err := codec.DecodeAlertingRulePayload(b)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rule.Alerts, &r.Alerts); err != nil {
		return err
	}

	r.Health = RuleHealth(rule.Health)
	r.Annotations = rule.Annotations
	r.Name = rule.Name
	r.Query = rule.Query
	r.Duration = rule.Duration
	r.Labels = rule.Labels
	r.LastError = rule.LastError
	r.EvaluationTime = rule.EvaluationTime
	r.LastEvaluation = rule.LastEvaluation
	r.State = rule.State

	return nil
}

func (r *RecordingRule) UnmarshalJSON(b []byte) error {
	ruleType, err := codec.DecodeRuleType(b)
	if err != nil {
		return err
	}
	if ruleType != string(RuleTypeRecording) {
		return fmt.Errorf("expected rule of type %s but got %s", string(RuleTypeRecording), ruleType)
	}

	rule, err := codec.DecodeRecordingRulePayload(b)
	if err != nil {
		return err
	}

	r.Health = RuleHealth(rule.Health)
	r.Labels = rule.Labels
	r.Name = rule.Name
	r.LastError = rule.LastError
	r.Query = rule.Query
	r.EvaluationTime = rule.EvaluationTime
	r.LastEvaluation = rule.LastEvaluation

	return nil
}

// Exemplar is additional information associated with a time series.
type Exemplar struct {
	Labels    model.LabelSet    `json:"labels"`
	Value     model.SampleValue `json:"value"`
	Timestamp model.Time        `json:"timestamp"`
}

type ExemplarQueryResult struct {
	SeriesLabels model.LabelSet `json:"seriesLabels"`
	Exemplars    []Exemplar     `json:"exemplars"`
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
	var res AlertsResult
	err := h.get(ctx, endpoints.Alerts, nil, &res)
	return res, err
}

func (h *httpAPI) AlertManagers(ctx context.Context) (AlertManagersResult, error) {
	var res AlertManagersResult
	err := h.get(ctx, endpoints.AlertManagers, nil, &res)
	return res, err
}

func (h *httpAPI) CleanTombstones(ctx context.Context) error {
	return h.post(ctx, endpoints.CleanTombstones, nil)
}

func (h *httpAPI) Config(ctx context.Context) (ConfigResult, error) {
	var res ConfigResult
	err := h.get(ctx, endpoints.Config, nil, &res)
	return res, err
}

func (h *httpAPI) DeleteSeries(ctx context.Context, matches []string, startTime, endTime time.Time) error {
	u := h.client.URL(endpoints.DeleteSeries, nil)
	q := u.Query()
	endpoints.AddMatchers(q, matches)
	endpoints.SetTime(q, "start", startTime)
	endpoints.SetTime(q, "end", endTime)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}

func (h *httpAPI) Flags(ctx context.Context) (FlagsResult, error) {
	var res FlagsResult
	err := h.get(ctx, endpoints.Flags, nil, &res)
	return res, err
}

func (h *httpAPI) Buildinfo(ctx context.Context) (BuildinfoResult, error) {
	var res BuildinfoResult
	err := h.get(ctx, endpoints.Buildinfo, nil, &res)
	return res, err
}

func (h *httpAPI) Runtimeinfo(ctx context.Context) (RuntimeinfoResult, error) {
	var res RuntimeinfoResult
	err := h.get(ctx, endpoints.Runtimeinfo, nil, &res)
	return res, err
}

func (h *httpAPI) LabelNames(ctx context.Context, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelNames, Warnings, error) {
	u := h.client.URL(endpoints.Labels, nil)
	q := addOptionalURLParams(u.Query(), opts)
	endpoints.SetTime(q, "start", startTime)
	endpoints.SetTime(q, "end", endTime)
	endpoints.AddMatchers(q, matches)

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var labelNames model.LabelNames
	err = json.Unmarshal(body, &labelNames)
	return labelNames, warnings, err
}

func (h *httpAPI) LabelValues(ctx context.Context, label string, matches []string, startTime, endTime time.Time, opts ...Option) (model.LabelValues, Warnings, error) {
	u := h.client.URL(endpoints.LabelValues, map[string]string{"name": label})
	q := addOptionalURLParams(u.Query(), opts)
	endpoints.SetTime(q, "start", startTime)
	endpoints.SetTime(q, "end", endTime)
	endpoints.AddMatchers(q, matches)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
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

type apiOptions = internaltypes.APIOptions

type Option func(c *apiOptions)

// WithTimeout can be used to provide an optional query evaluation timeout for Query and QueryRange.
// https://prometheus.io/docs/prometheus/latest/querying/api/#instant-queries
func WithTimeout(timeout time.Duration) Option {
	return func(o *apiOptions) {
		o.Timeout = timeout
	}
}

// WithLookbackDelta can be used to provide an optional query lookback delta for Query and QueryRange.
// This URL variable is not documented on Prometheus HTTP API.
// https://github.com/prometheus/prometheus/blob/e04913aea2792a5c8bc7b3130c389ca1b027dd9b/promql/engine.go#L162-L167
func WithLookbackDelta(lookbackDelta time.Duration) Option {
	return func(o *apiOptions) {
		o.LookbackDelta = lookbackDelta
	}
}

// WithStats can be used to provide an optional per step stats for Query and QueryRange.
// This URL variable is not documented on Prometheus HTTP API.
// https://github.com/prometheus/prometheus/blob/e04913aea2792a5c8bc7b3130c389ca1b027dd9b/promql/engine.go#L162-L167
func WithStats(stats StatsValue) Option {
	return func(o *apiOptions) {
		o.Stats = string(stats)
	}
}

// WithLimit provides an optional maximum number of returned entries for APIs that support limit parameter
// e.g. https://prometheus.io/docs/prometheus/latest/querying/api/#instant-querie:~:text=%3A%20End%20timestamp.-,limit%3D%3Cnumber%3E,-%3A%20Maximum%20number%20of
func WithLimit(limit uint64) Option {
	return func(o *apiOptions) {
		o.Limit = limit
	}
}

func addOptionalURLParams(q url.Values, opts []Option) url.Values {
	options := &apiOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return endpoints.ApplyOptions(q, *options)
}

func (h *httpAPI) Query(ctx context.Context, query string, ts time.Time, opts ...Option) (model.Value, Warnings, error) {
	u := h.client.URL(endpoints.Query, nil)
	q := addOptionalURLParams(u.Query(), opts)
	q.Set("query", query)
	endpoints.SetTime(q, "time", ts)

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
	endpoints.AddMatchers(q, matches)
	endpoints.SetTime(q, "start", startTime)
	endpoints.SetTime(q, "end", endTime)

	_, body, warnings, err := h.client.DoGetFallback(ctx, u, q)
	if err != nil {
		return nil, warnings, err
	}

	var result []model.LabelSet
	err = json.Unmarshal(body, &result)
	return result, warnings, err
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
	var res TargetsResult
	err := h.get(ctx, endpoints.Targets, nil, &res)
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
	var res TSDBBlocksResult
	err := h.get(ctx, endpoints.TSDBBlocks, nil, &res)
	return res, err
}

func (h *httpAPI) WalReplay(ctx context.Context) (WalReplayStatus, error) {
	var res WalReplayStatus
	err := h.get(ctx, endpoints.WalReplay, nil, &res)
	return res, err
}

func (h *httpAPI) QueryExemplars(ctx context.Context, query string, startTime, endTime time.Time) ([]ExemplarQueryResult, error) {
	u := h.client.URL(endpoints.QueryExemplars, nil)
	q := u.Query()
	q.Set("query", query)
	endpoints.SetTime(q, "start", startTime)
	endpoints.SetTime(q, "end", endTime)

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

// Warnings is an array of non critical errors
type Warnings []string

type apiClient interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, Warnings, error)
	DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error)
}

type apiClientImpl struct {
	client api.Client
}

func (c *apiClientImpl) URL(ep string, args map[string]string) *url.URL {
	return c.client.URL(ep, args)
}

func (c *apiClientImpl) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, Warnings, error) {
	resp, body, warnings, err := internaltransport.NewAPIClient(c.client).Do(ctx, req)
	return resp, body, Warnings(warnings), normalizeError(err)
}

func (c *apiClientImpl) DoGetFallback(ctx context.Context, u *url.URL, args url.Values) (*http.Response, []byte, Warnings, error) {
	resp, body, warnings, err := internaltransport.NewAPIClient(c.client).DoGetFallback(ctx, u, args)
	return resp, body, Warnings(warnings), normalizeError(err)
}

func normalizeError(err error) error {
	if err == nil {
		return nil
	}

	var transportErr *internaltransport.Error
	if !errors.As(err, &transportErr) {
		return err
	}

	return &Error{
		Type:   ErrorType(transportErr.Type),
		Msg:    transportErr.Msg,
		Detail: transportErr.Detail,
	}
}

func (h *httpAPI) get(ctx context.Context, endpoint string, args map[string]string, dst interface{}) error {
	u := h.client.URL(endpoint, args)
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	_, body, _, err := h.client.Do(ctx, req)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

func (h *httpAPI) post(ctx context.Context, endpoint string, args map[string]string) error {
	u := h.client.URL(endpoint, args)
	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return err
	}
	_, _, _, err = h.client.Do(ctx, req)
	return err
}
