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
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
	"unsafe"

	json "github.com/json-iterator/go"

	"github.com/prometheus/common/model"
)

func init() {
	json.RegisterTypeEncoderFunc("model.SamplePair", marshalSamplePairJSON, marshalJSONIsEmpty)
	json.RegisterTypeDecoderFunc("model.SamplePair", unmarshalSamplePairJSON)
	json.RegisterTypeEncoderFunc("model.SampleHistogramPair", marshalSampleHistogramPairJSON, marshalJSONIsEmpty)
	json.RegisterTypeDecoderFunc("model.SampleHistogramPair", unmarshalSampleHistogramPairJSON)
	json.RegisterTypeEncoderFunc("model.SampleStream", marshalSampleStreamJSON, marshalJSONIsEmpty) // Only needed for benchmark.
	json.RegisterTypeDecoderFunc("model.SampleStream", unmarshalSampleStreamJSON)                   // Only needed for benchmark.
}

func unmarshalSamplePairJSON(ptr unsafe.Pointer, iter *json.Iterator) {
	p := (*model.SamplePair)(ptr)
	if !iter.ReadArray() {
		iter.ReportError("unmarshal model.SamplePair", "SamplePair must be [timestamp, value]")
		return
	}
	t := iter.ReadNumber()
	if err := p.Timestamp.UnmarshalJSON([]byte(t)); err != nil {
		iter.ReportError("unmarshal model.SamplePair", err.Error())
		return
	}
	if !iter.ReadArray() {
		iter.ReportError("unmarshal model.SamplePair", "SamplePair missing value")
		return
	}

	f, err := strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		iter.ReportError("unmarshal model.SamplePair", err.Error())
		return
	}
	p.Value = model.SampleValue(f)

	if iter.ReadArray() {
		iter.ReportError("unmarshal model.SamplePair", "SamplePair has too many values, must be [timestamp, value]")
		return
	}
}

func marshalSamplePairJSON(ptr unsafe.Pointer, stream *json.Stream) {
	p := *((*model.SamplePair)(ptr))
	stream.WriteArrayStart()
	marshalTimestamp(p.Timestamp, stream)
	stream.WriteMore()
	marshalFloat(float64(p.Value), stream)
	stream.WriteArrayEnd()
}

func unmarshalSampleHistogramPairJSON(ptr unsafe.Pointer, iter *json.Iterator) {
	p := (*model.SampleHistogramPair)(ptr)
	if !iter.ReadArray() {
		iter.ReportError("unmarshal model.SampleHistogramPair", "SampleHistogramPair must be [timestamp, {histogram}]")
		return
	}
	t := iter.ReadNumber()
	if err := p.Timestamp.UnmarshalJSON([]byte(t)); err != nil {
		iter.ReportError("unmarshal model.SampleHistogramPair", err.Error())
		return
	}
	if !iter.ReadArray() {
		iter.ReportError("unmarshal model.SampleHistogramPair", "SamplePair missing histogram")
		return
	}
	h := &model.SampleHistogram{}
	p.Histogram = h
	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		switch key {
		case "count":
			f, err := strconv.ParseFloat(iter.ReadString(), 64)
			if err != nil {
				iter.ReportError("unmarshal model.SampleHistogramPair", "count of histogram is not a float")
				return
			}
			h.Count = model.FloatString(f)
		case "sum":
			f, err := strconv.ParseFloat(iter.ReadString(), 64)
			if err != nil {
				iter.ReportError("unmarshal model.SampleHistogramPair", "sum of histogram is not a float")
				return
			}
			h.Sum = model.FloatString(f)
		case "buckets":
			for iter.ReadArray() {
				b, err := unmarshalHistogramBucket(iter)
				if err != nil {
					iter.ReportError("unmarshal model.HistogramBucket", err.Error())
					return
				}
				h.Buckets = append(h.Buckets, b)
			}
		default:
			iter.ReportError("unmarshal model.SampleHistogramPair", fmt.Sprint("unexpected key in histogram:", key))
			return
		}
	}
	if iter.ReadArray() {
		iter.ReportError("unmarshal model.SampleHistogramPair", "SampleHistogramPair has too many values, must be [timestamp, {histogram}]")
		return
	}
}

func marshalSampleHistogramPairJSON(ptr unsafe.Pointer, stream *json.Stream) {
	p := *((*model.SampleHistogramPair)(ptr))
	stream.WriteArrayStart()
	marshalTimestamp(p.Timestamp, stream)
	stream.WriteMore()
	marshalHistogram(*p.Histogram, stream)
	stream.WriteArrayEnd()
}

func unmarshalSampleStreamJSON(ptr unsafe.Pointer, iter *json.Iterator) {
	ss := (*model.SampleStream)(ptr)
	for key := iter.ReadObject(); key != ""; key = iter.ReadObject() {
		switch key {
		case "metric":
			metricString := iter.ReadAny().ToString()
			if err := json.UnmarshalFromString(metricString, &ss.Metric); err != nil {
				iter.ReportError("unmarshal model.SampleStream", err.Error())
				return
			}
		case "values":
			for iter.ReadArray() {
				v := model.SamplePair{}
				unmarshalSamplePairJSON(unsafe.Pointer(&v), iter)
				ss.Values = append(ss.Values, v)
			}
		case "histograms":
			for iter.ReadArray() {
				h := model.SampleHistogramPair{}
				unmarshalSampleHistogramPairJSON(unsafe.Pointer(&h), iter)
				ss.Histograms = append(ss.Histograms, h)
			}
		default:
			iter.ReportError("unmarshal model.SampleStream", fmt.Sprint("unexpected key:", key))
			return
		}
	}
}

func marshalSampleStreamJSON(ptr unsafe.Pointer, stream *json.Stream) {
	ss := *((*model.SampleStream)(ptr))
	stream.WriteObjectStart()
	stream.WriteObjectField(`metric`)
	m, err := json.ConfigCompatibleWithStandardLibrary.Marshal(ss.Metric)
	if err != nil {
		stream.Error = err
		return
	}
	stream.SetBuffer(append(stream.Buffer(), m...))
	if len(ss.Values) > 0 {
		stream.WriteMore()
		stream.WriteObjectField(`values`)
		stream.WriteArrayStart()
		for i, v := range ss.Values {
			if i > 0 {
				stream.WriteMore()
			}
			marshalSamplePairJSON(unsafe.Pointer(&v), stream)
		}
		stream.WriteArrayEnd()
	}
	if len(ss.Histograms) > 0 {
		stream.WriteMore()
		stream.WriteObjectField(`histograms`)
		stream.WriteArrayStart()
		for i, h := range ss.Histograms {
			if i > 0 {
				stream.WriteMore()
			}
			marshalSampleHistogramPairJSON(unsafe.Pointer(&h), stream)
		}
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

func marshalFloat(v float64, stream *json.Stream) {
	stream.WriteRaw(`"`)
	// Taken from https://github.com/json-iterator/go/blob/master/stream_float.go#L71 as a workaround
	// to https://github.com/json-iterator/go/issues/365 (json-iterator, to follow json standard, doesn't allow inf/nan).
	buf := stream.Buffer()
	abs := math.Abs(v)
	fmt := byte('f')
	// Note: Must use float32 comparisons for underlying float32 value to get precise cutoffs right.
	if abs != 0 {
		if abs < 1e-6 || abs >= 1e21 {
			fmt = 'e'
		}
	}
	buf = strconv.AppendFloat(buf, v, fmt, -1, 64)
	stream.SetBuffer(buf)
	stream.WriteRaw(`"`)
}

func marshalTimestamp(timestamp model.Time, stream *json.Stream) {
	t := int64(timestamp)
	// Write out the timestamp as a float divided by 1000.
	// This is ~3x faster than converting to a float.
	if t < 0 {
		stream.WriteRaw(`-`)
		t = -t
	}
	stream.WriteInt64(t / 1000)
	fraction := t % 1000
	if fraction != 0 {
		stream.WriteRaw(`.`)
		if fraction < 100 {
			stream.WriteRaw(`0`)
		}
		if fraction < 10 {
			stream.WriteRaw(`0`)
		}
		stream.WriteInt64(fraction)
	}
}

func unmarshalHistogramBucket(iter *json.Iterator) (*model.HistogramBucket, error) {
	b := model.HistogramBucket{}
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	boundaries, err := iter.ReadNumber().Int64()
	if err != nil {
		return nil, err
	}
	b.Boundaries = int32(boundaries)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err := strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	b.Lower = model.FloatString(f)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err = strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	b.Upper = model.FloatString(f)
	if !iter.ReadArray() {
		return nil, errors.New("HistogramBucket must be [boundaries, lower, upper, count]")
	}
	f, err = strconv.ParseFloat(iter.ReadString(), 64)
	if err != nil {
		return nil, err
	}
	b.Count = model.FloatString(f)
	if iter.ReadArray() {
		return nil, errors.New("HistogramBucket has too many values, must be [boundaries, lower, upper, count]")
	}
	return &b, nil
}

// marshalHistogramBucket writes something like: [ 3, "-0.25", "0.25", "3"]
// See marshalHistogram to understand what the numbers mean
func marshalHistogramBucket(b model.HistogramBucket, stream *json.Stream) {
	stream.WriteArrayStart()
	stream.WriteInt32(b.Boundaries)
	stream.WriteMore()
	marshalFloat(float64(b.Lower), stream)
	stream.WriteMore()
	marshalFloat(float64(b.Upper), stream)
	stream.WriteMore()
	marshalFloat(float64(b.Count), stream)
	stream.WriteArrayEnd()
}

// marshalHistogram writes something like:
//
//	{
//	    "count": "42",
//	    "sum": "34593.34",
//	    "buckets": [
//	      [ 3, "-0.25", "0.25", "3"],
//	      [ 0, "0.25", "0.5", "12"],
//	      [ 0, "0.5", "1", "21"],
//	      [ 0, "2", "4", "6"]
//	    ]
//	}
//
// The 1st element in each bucket array determines if the boundaries are
// inclusive (AKA closed) or exclusive (AKA open):
//
//	0: lower exclusive, upper inclusive
//	1: lower inclusive, upper exclusive
//	2: both exclusive
//	3: both inclusive
//
// The 2nd and 3rd elements are the lower and upper boundary. The 4th element is
// the bucket count.
func marshalHistogram(h model.SampleHistogram, stream *json.Stream) {
	stream.WriteObjectStart()
	stream.WriteObjectField(`count`)
	marshalFloat(float64(h.Count), stream)
	stream.WriteMore()
	stream.WriteObjectField(`sum`)
	marshalFloat(float64(h.Sum), stream)

	bucketFound := false
	for _, bucket := range h.Buckets {
		if bucket.Count == 0 {
			continue // No need to expose empty buckets in JSON.
		}
		stream.WriteMore()
		if !bucketFound {
			stream.WriteObjectField(`buckets`)
			stream.WriteArrayStart()
		}
		bucketFound = true
		marshalHistogramBucket(*bucket, stream)
	}
	if bucketFound {
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

func marshalJSONIsEmpty(ptr unsafe.Pointer) bool {
	return false
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
