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
	"errors"
	"fmt"
	"time"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

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

	// The decoded value.
	v model.Value
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

func (rg *RuleGroup) UnmarshalJSON(b []byte) error {
	v := struct {
		Name     string            `json:"name"`
		File     string            `json:"file"`
		Interval float64           `json:"interval"`
		Rules    []json.RawMessage `json:"rules"`
	}{}

	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	rg.Name = v.Name
	rg.File = v.File
	rg.Interval = v.Interval

	for _, rule := range v.Rules {
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
	v := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v.Type == "" {
		return errors.New("type field not present in rule")
	}
	if v.Type != string(RuleTypeAlerting) {
		return fmt.Errorf("expected rule of type %s but got %s", string(RuleTypeAlerting), v.Type)
	}

	rule := struct {
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
	}{}
	if err := json.Unmarshal(b, &rule); err != nil {
		return err
	}
	r.Health = rule.Health
	r.Annotations = rule.Annotations
	r.Name = rule.Name
	r.Query = rule.Query
	r.Alerts = rule.Alerts
	r.Duration = rule.Duration
	r.Labels = rule.Labels
	r.LastError = rule.LastError
	r.EvaluationTime = rule.EvaluationTime
	r.LastEvaluation = rule.LastEvaluation
	r.State = rule.State

	return nil
}

func (r *RecordingRule) UnmarshalJSON(b []byte) error {
	v := struct {
		Type string `json:"type"`
	}{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	if v.Type == "" {
		return errors.New("type field not present in rule")
	}
	if v.Type != string(RuleTypeRecording) {
		return fmt.Errorf("expected rule of type %s but got %s", string(RuleTypeRecording), v.Type)
	}

	rule := struct {
		Name           string         `json:"name"`
		Query          string         `json:"query"`
		Labels         model.LabelSet `json:"labels,omitempty"`
		Health         RuleHealth     `json:"health"`
		LastError      string         `json:"lastError,omitempty"`
		EvaluationTime float64        `json:"evaluationTime"`
		LastEvaluation time.Time      `json:"lastEvaluation"`
	}{}
	if err := json.Unmarshal(b, &rule); err != nil {
		return err
	}
	r.Health = rule.Health
	r.Labels = rule.Labels
	r.Name = rule.Name
	r.LastError = rule.LastError
	r.Query = rule.Query
	r.EvaluationTime = rule.EvaluationTime
	r.LastEvaluation = rule.LastEvaluation

	return nil
}

func (qr *queryResult) UnmarshalJSON(b []byte) error {
	v := struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	switch v.Type {
	case model.ValScalar:
		var sv model.Scalar
		err = json.Unmarshal(v.Result, &sv)
		qr.v = &sv

	case model.ValVector:
		var vv model.Vector
		err = json.Unmarshal(v.Result, &vv)
		qr.v = vv

	case model.ValMatrix:
		var mv model.Matrix
		err = json.Unmarshal(v.Result, &mv)
		qr.v = mv

	default:
		err = fmt.Errorf("unexpected value type %q", v.Type)
	}
	return err
}
